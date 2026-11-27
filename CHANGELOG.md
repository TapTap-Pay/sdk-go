# Changelog

All notable changes to the TapTap-Pay Go SDK are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
`gen/` is regenerated from the
[TapTap-Pay monorepo](https://github.com/TapTap-Pay/monorepo) after each of
its releases.

## [Unreleased]

### Changed (breaking)

- The SDK now targets the contract the monorepo generates: proto packages
  `taptap.<context>.programmatic.v1`, Go packages under `gen/taptap/`.
- `Client` fields follow the new services: `Banking`, `Customers`,
  `InboundDocuments`, `Invoices`, `PaymentLinks`, `Payments`, `Refunds`,
  `Subvendors`, `Transactions`, `Wallets`, `Webhooks`. `PayIns`, `PayOuts`
  and `Transfers` are gone; payouts are read through `Banking`.
- Every request names the vendor in `VendorId`.
- Pagination is cursor based: `FetchPageFunc` takes a page token and
  returns the next one, and `Page` carries `NextPageToken` instead of
  `Meta`.

### Added

- `ErrorInfo` and `ErrorCode` decode the `taptap.common.v1.ErrorInfo`
  detail attached to every refusal.
- `IsAborted`, for an idempotency key reused with different content.
