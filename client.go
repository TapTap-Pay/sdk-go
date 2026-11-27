package taptap

import (
	"net/http"
	"time"

	"connectrpc.com/connect"

	bankingv1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/banking/programmatic/v1/programmaticv1connect"
	developerv1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/developer/programmatic/v1/programmaticv1connect"
	invoicingv1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/invoicing/programmatic/v1/programmaticv1connect"
	marketplacev1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/marketplace/programmatic/v1/programmaticv1connect"
	paymentv1connect "github.com/TapTap-Pay/sdk-go/gen/taptap/payment/programmatic/v1/programmaticv1connect"
)

// Environment URLs. CI rewrites these from secrets at release time.
const (
	ProdBaseURL    = "https://api.usetaptap.com/"
	SandboxBaseURL = "https://api.usetaptap.dev/"
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
//
// Every request names the vendor the API key belongs to in its VendorId
// field (ven_ followed by a UUID). The SDK does not fill it in.
type Client struct {
	Banking          bankingv1connect.BankingServiceClient
	Customers        paymentv1connect.CustomerServiceClient
	InboundDocuments invoicingv1connect.InboundDocumentServiceClient
	Invoices         invoicingv1connect.InvoiceServiceClient
	PaymentLinks     paymentv1connect.PaymentLinkServiceClient
	Payments         paymentv1connect.PaymentServiceClient
	Refunds          paymentv1connect.RefundServiceClient
	Subvendors       marketplacev1connect.SubvendorServiceClient
	Transactions     paymentv1connect.TransactionServiceClient
	Wallets          paymentv1connect.WalletServiceClient
	Webhooks         developerv1connect.WebhookServiceClient
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
		Banking:          bankingv1connect.NewBankingServiceClient(httpClient, baseURL, clientOpts),
		Customers:        paymentv1connect.NewCustomerServiceClient(httpClient, baseURL, clientOpts),
		InboundDocuments: invoicingv1connect.NewInboundDocumentServiceClient(httpClient, baseURL, clientOpts),
		Invoices:         invoicingv1connect.NewInvoiceServiceClient(httpClient, baseURL, clientOpts),
		PaymentLinks:     paymentv1connect.NewPaymentLinkServiceClient(httpClient, baseURL, clientOpts),
		Payments:         paymentv1connect.NewPaymentServiceClient(httpClient, baseURL, clientOpts),
		Refunds:          paymentv1connect.NewRefundServiceClient(httpClient, baseURL, clientOpts),
		Subvendors:       marketplacev1connect.NewSubvendorServiceClient(httpClient, baseURL, clientOpts),
		Transactions:     paymentv1connect.NewTransactionServiceClient(httpClient, baseURL, clientOpts),
		Wallets:          paymentv1connect.NewWalletServiceClient(httpClient, baseURL, clientOpts),
		Webhooks:         developerv1connect.NewWebhookServiceClient(httpClient, baseURL, clientOpts),
	}
}
