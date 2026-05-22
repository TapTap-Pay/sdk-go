# TapTap-Pay Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/TapTap-Pay/sdk-go.svg)](https://pkg.go.dev/github.com/TapTap-Pay/sdk-go)
[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)

The official Go SDK for the [TapTap-Pay](https://taptap.rs) API.

It wraps the generated [Connect-Go](https://connectrpc.com/) clients with
API-key authentication, transient-error retries with exponential backoff,
typed error helpers, and a Go 1.23 range-over-func pagination iterator.

The SDK exposes only the `programmatic/*` API surface — the API-key
authenticated endpoints meant for server-to-server integrations.

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
    commonv1 "github.com/TapTap-Pay/sdk-go/gen/v1/common"
    payment_linksv1 "github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payment_links"
)

func main() {
    client := taptap.New(taptap.Options{
        APIKey: os.Getenv("TAPTAP_SECRET"),
    })

    resp, err := client.PaymentLinks.CreatePaymentLink(context.Background(),
        connect.NewRequest(&payment_linksv1.CreatePaymentLinkRequest{
            Title:          "Premium plan",
            Amount:         &commonv1.Money{AmountMinor: 2999, Currency: "EUR"},
            TargetWalletId: os.Getenv("TAPTAP_WALLET_ID"),
        }))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("link id:", resp.Msg.Link.Id)
}
```

## Authentication

API keys are minted in the [dashboard](https://app.taptap.rs). Sandbox
keys are prefixed `sk_test_`, live keys `sk_live_`. The SDK sends them
as `Authorization: Bearer <key>` on every request.

## Configuration

```go
client := taptap.New(taptap.Options{
    APIKey:         "sk_live_...",       // required
    BaseURL:        "https://api.taptap.rs", // optional override
    MaxRetries:     3,                    // default 3
    RetryBaseDelay: 500 * time.Millisecond, // default 500ms
    UserAgent:      "my-app/1.4.0",       // optional, appended to SDK UA
    HTTPClient:     myClient,             // optional, shares connection pool
})
```

## Idempotency

Every state-changing RPC accepts an `idempotency_key` field on its
request message. Send the same key to safely retry a write — the API
dedupes and returns the original result. Generate one with the
SDK helper:

```go
key := taptap.NewIdempotencyKey() // UUID v4
resp, err := client.Payments.CreatePayment(ctx, connect.NewRequest(&paymentsv1.CreatePaymentRequest{
    IdempotencyKey: key,
    // ...
}))
```

The SDK's retry layer reuses the same request on every attempt, so the
key is preserved naturally across automatic retries.

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
    // protovalidate rejected the request
case taptap.IsRateLimited(err):
    // 429 — retry budget exhausted
case taptap.IsFailedPrecondition(err):
    // resource in wrong state for the operation
case err != nil:
    return err
}
```

The raw `*connect.Error` is always reachable via `errors.As` for
inspecting `Code()`, `Message()`, and `Details()`.

## Pagination

List endpoints return one page at a time. Use the SDK's iterator to
walk every page or every item lazily:

```go
import (
    "github.com/TapTap-Pay/sdk-go"
    payment_linksv1 "github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payment_links"
    types_v1 "github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/types"
)

fetch := func(ctx context.Context, page int32) ([]*types_v1.PaymentLink, *commonv1.PaginatedResponseMeta, error) {
    resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&payment_linksv1.ListPaymentLinksRequest{
        Pagination: &commonv1.PaginationRequestData{Page: page, PageSize: 100},
    }))
    if err != nil {
        return nil, nil, err
    }
    return resp.Msg.Links, resp.Msg.Meta, nil
}

for link, err := range taptap.Items(taptap.Iter(ctx, fetch)) {
    if err != nil {
        return err
    }
    fmt.Println(link.Id)
}
```

## Versioning

Releases follow the upstream [TapTap-Pay API](https://github.com/TapTap-Pay/api)
tag exactly — `v0.0.32` of the API ships as `v0.0.32` of every SDK.
Generated code is regenerated and pushed on each API release; the
hand-written ergonomics layer is left untouched.

## Contributing

The generated code in `gen/` is overwritten by CI on every release.
Don't hand-edit it — change the source `.proto` in the
[`api`](https://github.com/TapTap-Pay/api) repo instead.

Hand-written ergonomics (everything outside `gen/`) is fair game for
PRs.

## License

Apache 2.0 — see [LICENSE](LICENSE).
