// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/connect/v2/connectproto"

	proteusv1 "github.com/gsoultan/proteus-server-sdk/go/gen/proteus/v1"
	"github.com/gsoultan/proteus-server-sdk/go/gen/proteus/v1/proteusv1connect"
)

type fake struct {
	proteusv1connect.UnimplementedConversionServiceHandler
	proteusv1connect.UnimplementedTemplateServiceHandler
	auth string
}

func (f *fake) ListFormats(ctx context.Context, _ *proteusv1.ListFormatsRequest) (*proteusv1.ListFormatsResponse, error) {
	if info, ok := connect.CallInfoForServerContext(ctx); ok {
		f.auth = info.RequestHeader().Get("Authorization")
	}
	return &proteusv1.ListFormatsResponse{Conversions: []*proteusv1.Conversion{{MimeType: "application/pdf"}}}, nil
}

func (f *fake) RenderTemplate(context.Context, *proteusv1.RenderTemplateRequest) (*proteusv1.RenderTemplateResponse, error) {
	err := connect.NewError(connect.CodeInvalidArgument, "data does not match")
	detail, derr := connectproto.NewErrorDetail(&proteusv1.TemplateDataError{Problems: []*proteusv1.DataProblem{
		{Path: "items[2].qty", Message: "must be a number"},
	}})
	if derr != nil {
		return nil, derr
	}
	return nil, err.WithDetail(detail)
}

func newServer(t *testing.T) (*fake, string) {
	t.Helper()
	f := &fake{}
	server := connect.NewServer()
	proteusv1connect.RegisterConversionServiceHandler(server, f)
	proteusv1connect.RegisterTemplateServiceHandler(server, f)
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	srv := httptest.NewUnstartedServer(mux)
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	srv.Config.Protocols = p
	srv.Start()
	t.Cleanup(srv.Close)
	return f, srv.URL
}

func TestClient(t *testing.T) {
	f, url := newServer(t)
	ctx := context.Background()

	c := New(url, "secret")
	res, err := c.Conversion.ListFormats(ctx, &proteusv1.ListFormatsRequest{})
	if err != nil || len(res.GetConversions()) != 1 {
		t.Fatalf("ListFormats = %v, %v", res, err)
	}
	if f.auth != "Bearer secret" {
		t.Fatalf("authorization = %q", f.auth)
	}

	_, err = c.Templates.RenderTemplate(ctx, &proteusv1.RenderTemplateRequest{})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err = %v", err)
	}
	problems := DataProblems(err)
	if len(problems) != 1 || problems[0].GetPath() != "items[2].qty" {
		t.Fatalf("problems = %v", problems)
	}
	if DataProblems(nil) != nil || DataProblems(connect.NewError(connect.CodeNotFound, "x")) != nil {
		t.Fatal("problems from an error without detail")
	}

	// No key sends no header; gRPC over h2c works too.
	p := new(http.Protocols)
	p.SetUnencryptedHTTP2(true)
	g := New(url, "", WithGRPC(), WithHTTPClient(&http.Client{Transport: &http.Transport{Protocols: p}}))
	if _, err := g.Conversion.ListFormats(ctx, &proteusv1.ListFormatsRequest{}); err != nil {
		t.Fatal(err)
	}
	if f.auth != "" {
		t.Fatalf("authorization without key = %q", f.auth)
	}
}
