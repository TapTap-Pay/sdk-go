package taptap

import (
	"context"
	"iter"

	commonv1 "github.com/TapTap-Pay/sdk-go/gen/v1/common"
)

// Page is one chunk of items returned by a List endpoint, plus the
// pagination metadata returned by the server. Used by Iter to expose
// every page of a List call as a Go 1.23 range-over-func iterator.
type Page[T any] struct {
	Items []T
	Meta  *commonv1.PaginatedResponseMeta
}

// FetchPageFunc is the per-page callback the caller supplies to Iter.
// It receives a 1-indexed page number and returns the items on that
// page plus the server's pagination meta.
type FetchPageFunc[T any] func(ctx context.Context, page int32) (items []T, meta *commonv1.PaginatedResponseMeta, err error)

// Iter walks every page of a List endpoint, surfacing one Page at a
// time. Use Items if you'd rather flatten to individual items.
//
// Example:
//
//	for page, err := range taptap.Iter(ctx, func(ctx context.Context, page int32) ([]*typesv1.PaymentLink, *commonv1.PaginatedResponseMeta, error) {
//	    resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&payment_linksv1.ListPaymentLinksRequest{
//	        Pagination: &commonv1.PaginationRequestData{Page: page, PageSize: 100},
//	    }))
//	    if err != nil {
//	        return nil, nil, err
//	    }
//	    return resp.Msg.Links, resp.Msg.Meta, nil
//	}) {
//	    if err != nil {
//	        return err
//	    }
//	    for _, link := range page.Items {
//	        fmt.Println(link.Id)
//	    }
//	}
func Iter[T any](ctx context.Context, fetch FetchPageFunc[T]) iter.Seq2[Page[T], error] {
	return func(yield func(Page[T], error) bool) {
		var page int32 = 1
		for {
			items, meta, err := fetch(ctx, page)
			if err != nil {
				yield(Page[T]{}, err)
				return
			}
			if !yield(Page[T]{Items: items, Meta: meta}, nil) {
				return
			}
			if meta == nil || page >= meta.TotalPages {
				return
			}
			page++
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
//	    fmt.Println(link.Id)
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
