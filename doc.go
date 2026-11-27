// Package taptap is the official Go SDK for the TapTap-Pay API.
//
// The SDK wraps the generated Connect-Go clients with API-key
// authentication, retry-on-transient-error, typed error decoding, and
// page iteration helpers.
//
// Quick start:
//
//	client := taptap.New(taptap.Options{APIKey: os.Getenv("TAPTAP_API_KEY")})
//
//	resp, err := client.PaymentLinks.CreatePaymentLink(ctx, connect.NewRequest(&paymentv1.CreatePaymentLinkRequest{
//	    VendorId: os.Getenv("TAPTAP_VENDOR_ID"),
//	    Title:    "Premium plan",
//	    Amount:   &commonv1.Money{Currency: "EUR", AmountMinor: 2999},
//	    Usage: &paymentv1.PaymentLinkUsage{
//	        Usage: &paymentv1.PaymentLinkUsage_Unlimited{Unlimited: true},
//	    },
//	}))
//
// Every request names the vendor the API key belongs to in VendorId.
//
// See https://docs.usetaptap.com for the full API reference.
package taptap
