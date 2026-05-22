// Example: mint a payment link, fetch it back, then iterate every
// existing link in the vendor's account.
//
// Run with:
//
//	TAPTAP_SECRET=sk_test_... TAPTAP_WALLET_ID=<uuid> \
//	    go run ./examples/quickstart
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
	typesv1 "github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/types"
)

func main() {
	secret := os.Getenv("TAPTAP_SECRET")
	wallet := os.Getenv("TAPTAP_WALLET_ID")
	if secret == "" || wallet == "" {
		log.Fatal("set TAPTAP_SECRET and TAPTAP_WALLET_ID")
	}

	ctx := context.Background()
	client := taptap.New(taptap.Options{APIKey: secret})

	created, err := client.PaymentLinks.CreatePaymentLink(ctx, connect.NewRequest(&payment_linksv1.CreatePaymentLinkRequest{
		Title:          "Premium plan",
		Amount:         &commonv1.Money{AmountMinor: 2999, Currency: "EUR"},
		TargetWalletId: wallet,
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created:", created.Msg.Link.Id)

	fetched, err := client.PaymentLinks.GetPaymentLink(ctx, connect.NewRequest(&payment_linksv1.GetPaymentLinkRequest{
		Id: created.Msg.Link.Id,
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("fetched:", fetched.Msg.Link.Title)

	fetchPage := func(ctx context.Context, page int32) ([]*typesv1.PaymentLink, *commonv1.PaginatedResponseMeta, error) {
		resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&payment_linksv1.ListPaymentLinksRequest{
			Pagination: &commonv1.PaginationRequestData{Page: page, PageSize: 100},
		}))
		if err != nil {
			return nil, nil, err
		}
		return resp.Msg.Links, resp.Msg.Meta, nil
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
