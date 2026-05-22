package taptap

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"connectrpc.com/connect"
)

// authInterceptor stamps every outgoing request with a bearer token
// drawn from the API key. Applied to both unary and streaming calls.
type authInterceptor struct {
	apiKey string
}

func newAuthInterceptor(apiKey string) connect.Interceptor {
	return &authInterceptor{apiKey: apiKey}
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", "Bearer "+i.apiKey)
		return next(ctx, req)
	}
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", "Bearer "+i.apiKey)
		return conn
	}
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// userAgentInterceptor sets a SDK-identifying User-Agent so the API
// can attribute load to integrators and surface SDK version issues in
// support. Format: "taptap-sdk-go/<ver> (go<goVer>; <os>/<arch>) <user>".
type userAgentInterceptor struct {
	value string
}

func newUserAgentInterceptor(extra string) connect.Interceptor {
	parts := []string{
		fmt.Sprintf("taptap-sdk-go/%s", Version),
		fmt.Sprintf("(%s; %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
	if extra != "" {
		parts = append(parts, extra)
	}
	return &userAgentInterceptor{value: strings.Join(parts, " ")}
}

func (i *userAgentInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("User-Agent", i.value)
		return next(ctx, req)
	}
}

func (i *userAgentInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("User-Agent", i.value)
		return conn
	}
}

func (i *userAgentInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
