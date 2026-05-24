package taptap

import (
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/invoices/invoicesv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payins/payinsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payment_links/payment_linksv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payments/paymentsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/payouts/payoutsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/refunds/refundsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/transactions/transactionsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/transfers/transfersv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/wallets/walletsv1connect"
	"github.com/TapTap-Pay/sdk-go/gen/v1/programmatic/webhooks/webhooksv1connect"
)

// Environment URLs. CI rewrites these from secrets at release time.
const (
	ProdBaseURL    = "https://api.taptap.rs"
	SandboxBaseURL = "https://api.usetaptap.dev"
)

// Options configures a Client. APIKey is required; everything else has
// sensible defaults.
type Options struct {
	// APIKey is the secret key minted in the dashboard.
	APIKey string

	// Mode selects the environment: "production" (default) or "sandbox".
	// Ignored when BaseURL is set explicitly.
	Mode string

	// BaseURL overrides the API endpoint. When empty, resolved from Mode.
	BaseURL string

	// HTTPClient is used for transport. Leave nil for a sensible
	// default (60s timeout). Pass your own to share connection pools
	// or to plug in observability.
	HTTPClient *http.Client

	// MaxRetries caps automatic retries on transient errors
	// (Unavailable, DeadlineExceeded, ResourceExhausted, network
	// failures). Zero disables retries; default is 3.
	MaxRetries int

	// RetryBaseDelay is the initial backoff between retries; each
	// subsequent attempt doubles the delay (with jitter). Defaults to
	// 500ms.
	RetryBaseDelay time.Duration

	// UserAgent is appended to the SDK's own User-Agent header. Useful
	// for identifying your integration in support requests.
	UserAgent string
}

// Client is the entry point for the TapTap-Pay SDK. Construct one with
// New; the per-service sub-clients are safe for concurrent use.
type Client struct {
	Invoices     invoicesv1connect.InvoicesServiceClient
	PayIns       payinsv1connect.PayInsServiceClient
	PaymentLinks payment_linksv1connect.PaymentLinksServiceClient
	Payments     paymentsv1connect.PaymentsServiceClient
	PayOuts      payoutsv1connect.PayOutsServiceClient
	Refunds      refundsv1connect.RefundsServiceClient
	Transactions transactionsv1connect.TransactionsServiceClient
	Transfers    transfersv1connect.TransfersServiceClient
	Wallets      walletsv1connect.WalletsServiceClient
	Webhooks     webhooksv1connect.WebhooksServiceClient
}

// New constructs a Client. APIKey is required; everything else is
// optional. Panics if APIKey is empty — there's no recovery path and
// catching it later would just delay the failure to the first request.
func New(opts Options) *Client {
	if opts.APIKey == "" {
		panic("taptap: Options.APIKey is required")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		if opts.Mode == "sandbox" {
			baseURL = SandboxBaseURL
		} else {
			baseURL = ProdBaseURL
		}
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	maxRetries := opts.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	baseDelay := opts.RetryBaseDelay
	if baseDelay == 0 {
		baseDelay = 500 * time.Millisecond
	}

	clientOpts := connect.WithClientOptions(
		connect.WithInterceptors(
			newUserAgentInterceptor(opts.UserAgent),
			newAuthInterceptor(opts.APIKey),
			newRetryInterceptor(maxRetries, baseDelay),
		),
	)

	return &Client{
		Invoices:     invoicesv1connect.NewInvoicesServiceClient(httpClient, baseURL, clientOpts),
		PayIns:       payinsv1connect.NewPayInsServiceClient(httpClient, baseURL, clientOpts),
		PaymentLinks: payment_linksv1connect.NewPaymentLinksServiceClient(httpClient, baseURL, clientOpts),
		Payments:     paymentsv1connect.NewPaymentsServiceClient(httpClient, baseURL, clientOpts),
		PayOuts:      payoutsv1connect.NewPayOutsServiceClient(httpClient, baseURL, clientOpts),
		Refunds:      refundsv1connect.NewRefundsServiceClient(httpClient, baseURL, clientOpts),
		Transactions: transactionsv1connect.NewTransactionsServiceClient(httpClient, baseURL, clientOpts),
		Transfers:    transfersv1connect.NewTransfersServiceClient(httpClient, baseURL, clientOpts),
		Wallets:      walletsv1connect.NewWalletsServiceClient(httpClient, baseURL, clientOpts),
		Webhooks:     webhooksv1connect.NewWebhooksServiceClient(httpClient, baseURL, clientOpts),
	}
}
