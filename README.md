# TapTap-Pay Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/TapTap-Pay/sdk-go.svg)](https://pkg.go.dev/github.com/TapTap-Pay/sdk-go)
[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)

The official Go SDK for the [TapTap-Pay](https://usetaptap.com) API.

It wraps the generated [Connect-Go](https://connectrpc.com/) clients with
API-key authentication, transient-error retries with exponential backoff,
typed error helpers, and a Go 1.23 range-over-func pagination iterator.

The SDK exposes only the programmatic API surface (the
`taptap.<context>.programmatic.v1` packages) — the API-key authenticated
endpoints meant for server-to-server integrations.

## Install

```bash
go get github.com/TapTap-Pay/sdk-go
```

Requires Go 1.25+ (matches the upstream API).

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "connectrpc.com/connect"

    "github.com/TapTap-Pay/sdk-go"
    commonv1 "github.com/TapTap-Pay/sdk-go/gen/taptap/common/v1"
    paymentv1 "github.com/TapTap-Pay/sdk-go/gen/taptap/payment/programmatic/v1"
)

func main() {
    client := taptap.New(taptap.Options{
        APIKey: os.Getenv("TAPTAP_API_KEY"),
    })

    resp, err := client.PaymentLinks.CreatePaymentLink(context.Background(),
        connect.NewRequest(&paymentv1.CreatePaymentLinkRequest{
            VendorId: os.Getenv("TAPTAP_VENDOR_ID"),
            Title:    "Premium plan",
            Amount:   &commonv1.Money{Currency: "EUR", AmountMinor: 2999},
            Usage: &paymentv1.PaymentLinkUsage{
                Usage: &paymentv1.PaymentLinkUsage_Unlimited{Unlimited: true},
            },
        }))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("link id:", resp.Msg.GetPaymentLink().GetId())
}
```

## Services

| Client field | Service | Go package under `gen/taptap/` |
|---|---|---|
| `Banking` | `BankingService` | `banking/programmatic/v1` |
| `Customers` | `CustomerService` | `payment/programmatic/v1` |
| `InboundDocuments` | `InboundDocumentService` | `invoicing/programmatic/v1` |
| `Invoices` | `InvoiceService` | `invoicing/programmatic/v1` |
| `PaymentLinks` | `PaymentLinkService` | `payment/programmatic/v1` |
| `Payments` | `PaymentService` | `payment/programmatic/v1` |
| `Refunds` | `RefundService` | `payment/programmatic/v1` |
| `Subvendors` | `SubvendorService` | `marketplace/programmatic/v1` |
| `Transactions` | `TransactionService` | `payment/programmatic/v1` |
| `Wallets` | `WalletService` | `payment/programmatic/v1` |
| `Webhooks` | `WebhookService` | `developer/programmatic/v1` |

Every generated message package is named `programmaticv1`, so import each
with an alias (`paymentv1`, `invoicingv1`, `developerv1`, `bankingv1`,
`marketplacev1`).

## Authentication

API keys are minted in the [dashboard](https://app.usetaptap.com), in your
vendor's settings under API keys. A key looks like `taptap_live_` followed
by a lookup part, a dot and a secret part. The SDK sends it as
`Authorization: Bearer <key>` on every request.

## Vendor id

A key belongs to one vendor, and every request names that vendor in its
`VendorId` field (`ven_` followed by a UUID). The SDK does not fill it in:
set it on each request. Naming a vendor the key does not belong to is
answered like a vendor that does not exist (`NotFound`, code
`VENDOR_NOT_FOUND`).

## Configuration

```go
client := taptap.New(taptap.Options{
    APIKey:         "taptap_live_...",   // required
    BaseURL:        "https://api.usetaptap.com", // optional override
    MaxRetries:     3,                    // default 3
    RetryBaseDelay: 500 * time.Millisecond, // default 500ms
    UserAgent:      "my-app/1.4.0",       // optional, appended to SDK UA
    HTTPClient:     myClient,             // optional, shares connection pool
})
```

## Idempotency

State-changing RPCs accept an `idempotency_key` field on their request
message (`RefundPaymentRequest` calls it `request_key`). Send the same key
to safely retry a write — the API dedupes and returns the original result.
Generate one with the SDK helper:

```go
key := taptap.NewIdempotencyKey() // UUID v4
resp, err := client.Payments.CreatePayment(ctx, connect.NewRequest(&paymentv1.CreatePaymentRequest{
    VendorId:       vendorID,
    IdempotencyKey: key,
    // ...
}))
```

The SDK's retry layer reuses the same request on every attempt, so the
key is preserved naturally across automatic retries. Reusing a key for a
request with different content is refused with `Aborted` and the code
`IDEMPOTENCY_KEY_CONFLICT` (`taptap.IsAborted`).

## Retries

Transient errors (`Unavailable`, `DeadlineExceeded`, `ResourceExhausted`,
network failures) are retried up to `MaxRetries` times with exponential
backoff and full jitter. All other Connect codes surface to the caller
on the first attempt. Streaming RPCs are not retried — Connect bidi
streams aren't safely replayable without application-level checkpoints.

## Errors

```go
resp, err := client.PaymentLinks.GetPaymentLink(ctx, req)
switch {
case taptap.IsNotFound(err):
    // 404
case taptap.IsInvalidArgument(err):
    // the request was rejected as malformed
case taptap.IsRateLimited(err):
    // 429 — retry budget exhausted
case taptap.IsFailedPrecondition(err):
    // resource in wrong state for the operation
case err != nil:
    return err
}
```

Every refusal also carries a `taptap.common.v1.ErrorInfo` detail: a stable
machine code, the parameters around it, and an English debug message.
Branch on the code, never on the message:

```go
if info, ok := taptap.ErrorInfo(err); ok {
    log.Println(info.GetCode(), info.GetParams()) // e.g. PAYMENT_LINK_NOT_FOUND
}
if taptap.ErrorCode(err) == "REFUND_EXCEEDS_REMAINDER" {
    // offer a smaller refund
}
```

The raw `*connect.Error` is always reachable via `errors.As` for
inspecting `Code()`, `Message()`, and `Details()`.

## Pagination

List endpoints return one page at a time and a cursor for the next. Use
the SDK's iterator to walk every page or every item lazily. The callback
receives the cursor (empty for the first page) and returns the items and
the next cursor:

```go
import (
    "github.com/TapTap-Pay/sdk-go"
    paymentv1 "github.com/TapTap-Pay/sdk-go/gen/taptap/payment/programmatic/v1"
)

fetch := func(ctx context.Context, pageToken string) ([]*paymentv1.PaymentLink, string, error) {
    resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&paymentv1.ListPaymentLinksRequest{
        VendorId:  vendorID,
        PageSize:  100,
        PageToken: pageToken,
    }))
    if err != nil {
        return nil, "", err
    }
    return resp.Msg.GetPaymentLinks(), resp.Msg.GetNextPageToken(), nil
}

for link, err := range taptap.Items(taptap.Iter(ctx, fetch)) {
    if err != nil {
        return err
    }
    fmt.Println(link.GetId())
}
```

The cursor is spelled three ways across the API, and the callback is where
you map it:

| Lists | Request | Response |
|---|---|---|
| Most (`ListPayments`, `ListPaymentLinks`, `ListRefunds`, `ListCustomers`, `ListUnappliedReceipts`, `ListWalletTransactions`, `ListBankTransfers`, `ListSubscriptions`, `ListDeliveries`) | `PageToken` | `NextPageToken` |
| `ListInvoices`, `ListInboundDocuments` | `Cursor` | `NextCursor` |
| `ListSubvendors` | `Page: &commonv1.PageRequest{PageToken: ...}` | `GetPage().GetNextPageToken()` |

`ListWallets` returns every wallet in one response.

## Versioning

This module has its own semver tags. After each release of the
[TapTap-Pay monorepo](https://github.com/TapTap-Pay/monorepo), `gen/` is
regenerated from its programmatic protos and pushed here, which cuts the
next SDK release.

## Contributing

The generated code in `gen/` is overwritten by CI on every release.
Don't hand-edit it — change the source `.proto` under
`proto/taptap/*/programmatic/` in the
[`monorepo`](https://github.com/TapTap-Pay/monorepo) instead.

Hand-written ergonomics (everything outside `gen/`) is fair game for
PRs.

## License

Apache 2.0 — see [LICENSE](LICENSE).
