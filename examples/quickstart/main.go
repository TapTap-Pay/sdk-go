// Example: mint a payment link, fetch it back, then iterate every
// existing link in the vendor's account.
//
// Run with:
//
//	TAPTAP_API_KEY=taptap_live_... TAPTAP_VENDOR_ID=ven_<uuid> \
//	    go run ./examples/quickstart
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
	apiKey := os.Getenv("TAPTAP_API_KEY")
	vendorID := os.Getenv("TAPTAP_VENDOR_ID")
	if apiKey == "" || vendorID == "" {
		log.Fatal("set TAPTAP_API_KEY and TAPTAP_VENDOR_ID")
	}

	ctx := context.Background()
	client := taptap.New(taptap.Options{APIKey: apiKey})

	created, err := client.PaymentLinks.CreatePaymentLink(ctx, connect.NewRequest(&paymentv1.CreatePaymentLinkRequest{
		VendorId: vendorID,
		Title:    "Premium plan",
		Amount:   &commonv1.Money{Currency: "EUR", AmountMinor: 2999},
		Usage: &paymentv1.PaymentLinkUsage{
			Usage: &paymentv1.PaymentLinkUsage_Unlimited{Unlimited: true},
		},
		IdempotencyKey: taptap.NewIdempotencyKey(),
	}))
	if err != nil {
		if info, ok := taptap.ErrorInfo(err); ok {
			log.Fatalf("refused: %s %v", info.GetCode(), info.GetParams())
		}
		log.Fatal(err)
	}
	linkID := created.Msg.GetPaymentLink().GetId()
	fmt.Println("created:", linkID)

	fetched, err := client.PaymentLinks.GetPaymentLink(ctx, connect.NewRequest(&paymentv1.GetPaymentLinkRequest{
		VendorId:      vendorID,
		PaymentLinkId: linkID,
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("fetched:", fetched.Msg.GetPaymentLink().GetTitle())

	fetchPage := func(ctx context.Context, pageToken string) ([]*paymentv1.PaymentLink, string, error) {
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

	count := 0
	for _, err := range taptap.Items(taptap.Iter(ctx, fetchPage)) {
		if err != nil {
			log.Fatal(err)
		}
		count++
	}
	fmt.Println("total links:", count)
}
