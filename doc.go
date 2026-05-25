// Package taptap is the official Go SDK for the TapTap-Pay API.
//
// The SDK wraps the generated Connect-Go clients with API-key
// authentication, retry-on-transient-error, and page iteration helpers.
//
// Quick start:
//
//	client := taptap.New(taptap.Options{APIKey: os.Getenv("TAPTAP_SECRET")})
//
//	resp, err := client.PaymentLinks.CreatePaymentLink(ctx, connect.NewRequest(&payment_linksv1.CreatePaymentLinkRequest{
//	    Title:          "Premium plan",
//	    Amount:         &commonv1.Money{AmountMinor: 2999, Currency: "EUR"},
//	    TargetWalletId: walletID,
//	}))
//
// See https://docs.usetaptap.com for the full API reference.
package taptap
