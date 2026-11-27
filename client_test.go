package taptap_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	taptap "github.com/TapTap-Pay/sdk-go"
	commonv1 "github.com/TapTap-Pay/sdk-go/gen/taptap/common/v1"
	paymentv1 "github.com/TapTap-Pay/sdk-go/gen/taptap/payment/programmatic/v1"
	paymentv1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/payment/programmatic/v1/programmaticv1connect"
)

const (
	testAPIKey   = "taptap_live_000000000000.test"
	testVendorID = "ven_00000000-0000-0000-0000-000000000001"
)

// paymentLinks is a stand-in for the API: two pages of links, and a
// refusal carrying the ErrorInfo detail for any link lookup.
type paymentLinks struct {
	paymentv1connect.UnimplementedPaymentLinkServiceHandler
	authorization []string
	pageTokens    []string
}

func (h *paymentLinks) GetPaymentLink(_ context.Context, req *connect.Request[paymentv1.GetPaymentLinkRequest]) (*connect.Response[paymentv1.GetPaymentLinkResponse], error) {
	h.authorization = append(h.authorization, req.Header().Get("Authorization"))
	refusal := connect.NewError(connect.CodeNotFound, errors.New("payment link: not found"))
	detail, err := connect.NewErrorDetail(&commonv1.ErrorInfo{
		Code:    "PAYMENT_LINK_NOT_FOUND",
		Params:  map[string]string{"payment_link_id": req.Msg.GetPaymentLinkId()},
		Message: "payment link: not found",
	})
	if err != nil {
		return nil, err
	}
	refusal.AddDetail(detail)
	return nil, refusal
}

func (h *paymentLinks) ListPaymentLinks(_ context.Context, req *connect.Request[paymentv1.ListPaymentLinksRequest]) (*connect.Response[paymentv1.ListPaymentLinksResponse], error) {
	h.pageTokens = append(h.pageTokens, req.Msg.GetPageToken())
	if req.Msg.GetVendorId() != testVendorID {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("vendor: not found"))
	}
	if req.Msg.GetPageToken() == "" {
		return connect.NewResponse(&paymentv1.ListPaymentLinksResponse{
			PaymentLinks:  []*paymentv1.PaymentLink{{Id: "plink_1"}, {Id: "plink_2"}},
			NextPageToken: "page-2",
		}), nil
	}
	return connect.NewResponse(&paymentv1.ListPaymentLinksResponse{
		PaymentLinks: []*paymentv1.PaymentLink{{Id: "plink_3"}},
	}), nil
}

func newTestClient(t *testing.T, handler *paymentLinks) *taptap.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(paymentv1connect.NewPaymentLinkServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return taptap.New(taptap.Options{APIKey: testAPIKey, BaseURL: server.URL, HTTPClient: server.Client()})
}

func TestErrorInfoDecodesTheRefusalDetail(t *testing.T) {
	handler := &paymentLinks{}
	client := newTestClient(t, handler)

	_, err := client.PaymentLinks.GetPaymentLink(context.Background(), connect.NewRequest(&paymentv1.GetPaymentLinkRequest{
		VendorId:      testVendorID,
		PaymentLinkId: "plink_missing",
	}))
	if !taptap.IsNotFound(err) {
		t.Fatalf("IsNotFound(%v) = false, want true", err)
	}
	info, ok := taptap.ErrorInfo(err)
	if !ok {
		t.Fatalf("ErrorInfo(%v) found no detail", err)
	}
	if got := info.GetCode(); got != "PAYMENT_LINK_NOT_FOUND" {
		t.Errorf("code = %q, want PAYMENT_LINK_NOT_FOUND", got)
	}
	if got := info.GetParams()["payment_link_id"]; got != "plink_missing" {
		t.Errorf("params[payment_link_id] = %q, want plink_missing", got)
	}
	if got := taptap.ErrorCode(err); got != "PAYMENT_LINK_NOT_FOUND" {
		t.Errorf("ErrorCode = %q, want PAYMENT_LINK_NOT_FOUND", got)
	}
	if len(handler.authorization) != 1 || handler.authorization[0] != "Bearer "+testAPIKey {
		t.Errorf("Authorization headers = %q, want one bearer header (NotFound is not retried)", handler.authorization)
	}
}

func TestErrorInfoWithoutADetail(t *testing.T) {
	for name, err := range map[string]error{
		"nil":                nil,
		"plain error":        errors.New("dial tcp: connection refused"),
		"connect, no detail": connect.NewError(connect.CodeUnavailable, errors.New("down")),
	} {
		if info, ok := taptap.ErrorInfo(err); ok || info != nil {
			t.Errorf("%s: ErrorInfo = (%v, %v), want (nil, false)", name, info, ok)
		}
		if got := taptap.ErrorCode(err); got != "" {
			t.Errorf("%s: ErrorCode = %q, want empty", name, got)
		}
	}
}

func TestIterFollowsNextPageTokenUntilEmpty(t *testing.T) {
	handler := &paymentLinks{}
	client := newTestClient(t, handler)
	ctx := context.Background()

	fetch := func(ctx context.Context, pageToken string) ([]*paymentv1.PaymentLink, string, error) {
		resp, err := client.PaymentLinks.ListPaymentLinks(ctx, connect.NewRequest(&paymentv1.ListPaymentLinksRequest{
			VendorId:  testVendorID,
			PageToken: pageToken,
		}))
		if err != nil {
			return nil, "", err
		}
		return resp.Msg.GetPaymentLinks(), resp.Msg.GetNextPageToken(), nil
	}

	var ids []string
	for link, err := range taptap.Items(taptap.Iter(ctx, fetch)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, link.GetId())
	}
	if want := []string{"plink_1", "plink_2", "plink_3"}; !equal(ids, want) {
		t.Errorf("items = %v, want %v", ids, want)
	}
	if want := []string{"", "page-2"}; !equal(handler.pageTokens, want) {
		t.Errorf("page tokens sent = %q, want %q", handler.pageTokens, want)
	}
}

func TestIterSurfacesAFetchError(t *testing.T) {
	boom := errors.New("boom")
	calls := 0
	fetch := func(context.Context, string) ([]string, string, error) {
		calls++
		return nil, "", boom
	}
	for _, err := range taptap.Items(taptap.Iter(context.Background(), fetch)) {
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1", calls)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
