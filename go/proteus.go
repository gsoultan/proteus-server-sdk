// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package proteus is the Go client for proteus-server. It wraps the generated
// Connect clients for the conversion and template services with the API key
// header, and decodes the template data errors the server attaches.
//
//	c := proteus.New("https://proteus.example.com", apiKey)
//	res, err := c.Conversion.Convert(ctx, &proteusv1.ConvertRequest{...})
//
// By default the client speaks the Connect protocol over HTTP/1.1 or HTTP/2,
// so it works through any proxy. WithGRPC switches to gRPC, which needs an
// HTTP/2 client (TLS, or h2c with http.Protocols SetUnencryptedHTTP2).
package proteus

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/connect/v2/connectproto"

	proteusv1 "github.com/gsoultan/proteus-server-sdk/go/gen/proteus/v1"
	"github.com/gsoultan/proteus-server-sdk/go/gen/proteus/v1/proteusv1connect"
)

// Client holds one client per service, sharing a transport.
type Client struct {
	Conversion proteusv1connect.ConversionServiceClient
	Templates  proteusv1connect.TemplateServiceClient
}

type options struct {
	httpClient   *http.Client
	transport    []connecthttp.Option
	interceptors []connect.ClientInterceptor
}

// Option configures New.
type Option func(*options)

// WithHTTPClient sets the HTTP client; http.DefaultClient otherwise.
func WithHTTPClient(c *http.Client) Option { return func(o *options) { o.httpClient = c } }

// WithGRPC speaks gRPC instead of the Connect protocol.
func WithGRPC() Option {
	return func(o *options) { o.transport = append(o.transport, connecthttp.WithGRPC()) }
}

// WithGRPCWeb speaks gRPC-Web instead of the Connect protocol.
func WithGRPCWeb() Option {
	return func(o *options) { o.transport = append(o.transport, connecthttp.WithGRPCWeb()) }
}

// WithTransportOptions adds connecthttp transport options, such as
// compression or a protocol choice.
func WithTransportOptions(opts ...connecthttp.Option) Option {
	return func(o *options) { o.transport = append(o.transport, opts...) }
}

// WithInterceptors adds client interceptors after the API key one.
func WithInterceptors(i ...connect.ClientInterceptor) Option {
	return func(o *options) { o.interceptors = append(o.interceptors, i...) }
}

// New returns a client for the server at baseURL (scheme and host, no
// path). apiKey may be empty for a server running with PROTEUS_ALLOW_NO_AUTH.
func New(baseURL, apiKey string, opts ...Option) *Client {
	o := options{httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(&o)
	}
	interceptors := append([]connect.ClientInterceptor{APIKey(apiKey)}, o.interceptors...)
	t := connecthttp.NewTransport(o.httpClient, baseURL, o.transport...)
	c := connect.NewClient(t, interceptors...)
	return &Client{
		Conversion: proteusv1connect.NewConversionServiceClient(c),
		Templates:  proteusv1connect.NewTemplateServiceClient(c),
	}
}

// APIKey is a client interceptor that sends key as a bearer token. An
// empty key sends nothing.
func APIKey(key string) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			if key != "" {
				if info, ok := connect.CallInfoForClientContext(ctx); ok {
					info.RequestHeader().Set("Authorization", "Bearer "+key)
				}
			}
			return next(ctx, spec)
		}
	}
}

// DataProblems returns the field problems the server attached to an
// INVALID_ARGUMENT error from a template render, or nil when err carries
// none. Each problem names the data path, such as "items[2].qty".
func DataProblems(err error) []*proteusv1.DataProblem {
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		return nil
	}
	for _, d := range cerr.Details() {
		msg, uerr := connectproto.UnmarshalErrorDetail(d)
		if uerr != nil {
			continue
		}
		if de, ok := msg.(*proteusv1.TemplateDataError); ok {
			return de.GetProblems()
		}
	}
	return nil
}
