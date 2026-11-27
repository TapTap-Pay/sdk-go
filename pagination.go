package taptap

import (
	"context"
	"iter"
)

// Page is one chunk of items returned by a List endpoint, plus the
// cursor the server returned for the page after it. Used by Iter to
// expose every page of a List call as a Go 1.23 range-over-func
// iterator.
type Page[T any] struct {
	Items []T
	// NextPageToken is the cursor for the following page; empty on the
	// last page.
	NextPageToken string
}

// FetchPageFunc is the per-page callback the caller supplies to Iter.
// It receives the cursor of the page to fetch (empty for the first
// page) and returns the items on that page plus the cursor of the next
// one, empty when there are no more pages.
//
// The cursor is spelled differently across the List endpoints, which is
// why the callback does the mapping:
//
//   - most requests take PageToken and answer NextPageToken;
//   - Invoices.ListInvoices and InboundDocuments.ListInboundDocuments
//     take Cursor and answer NextCursor;
//   - Subvendors.ListSubvendors takes a commonv1.PageRequest and answers
//     a commonv1.PageResponse (resp.Msg.GetPage().GetNextPageToken()).
type FetchPageFunc[T any] func(ctx context.Context, pageToken string) (items []T, nextPageToken string, err error)

// Iter walks every page of a List endpoint, surfacing one Page at a
// time. Use Items if you'd rather flatten to individual items.
//
// Example:
//
//	for page, err := range taptap.Iter(ctx, func(ctx context.Context, pageToken string) ([]*paymentv1.PaymentLink, string, error) {
//	    resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&paymentv1.ListPaymentLinksRequest{
//	        VendorId:  vendorID,
//	        PageSize:  100,
//	        PageToken: pageToken,
//	    }))
//	    if err != nil {
//	        return nil, "", err
//	    }
//	    return resp.Msg.GetPaymentLinks(), resp.Msg.GetNextPageToken(), nil
//	}) {
//	    if err != nil {
//	        return err
//	    }
//	    for _, link := range page.Items {
//	        fmt.Println(link.GetId())
//	    }
//	}
func Iter[T any](ctx context.Context, fetch FetchPageFunc[T]) iter.Seq2[Page[T], error] {
	return func(yield func(Page[T], error) bool) {
		pageToken := ""
		for {
			items, next, err := fetch(ctx, pageToken)
			if err != nil {
				yield(Page[T]{}, err)
				return
			}
			if !yield(Page[T]{Items: items, NextPageToken: next}, nil) {
				return
			}
			if next == "" {
				return
			}
			pageToken = next
		}
	}
}

// Items flattens a paged iterator to one item at a time. Errors from
// the underlying fetch surface as a (zero, err) pair and terminate
// iteration.
//
// Example:
//
//	for link, err := range taptap.Items(taptap.Iter(ctx, fetch)) {
//	    if err != nil {
//	        return err
//	    }
//	    fmt.Println(link.GetId())
//	}
func Items[T any](pages iter.Seq2[Page[T], error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for page, err := range pages {
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}
