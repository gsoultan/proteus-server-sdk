# proteus-server-sdk

Clients for [proteus-server](https://github.com/gsoultan/proteus-server), the
document conversion and template service: TypeScript (`ts/`, published as
`@gsoultan/proteus-client`) and Go (`go/`, module
`github.com/gsoultan/proteus-server-sdk/go`). Both are generated from the
server's protobuf definition in `proto/` with [buf](https://buf.build) and
speak Connect, gRPC-Web or gRPC through [ConnectRPC](https://connectrpc.com).

## TypeScript

```sh
bun add @gsoultan/proteus-client @bufbuild/protobuf @connectrpc/connect @connectrpc/connect-web
```

```ts
import { createProteus, dataProblems, InputFormat, OutputFormat } from "@gsoultan/proteus-client";

const proteus = createProteus({ baseUrl: "https://proteus.example.com", apiKey: import.meta.env.VITE_PROTEUS_KEY });

// Convert a document.
const res = await proteus.conversion.convert({
  document: { format: InputFormat.MARKDOWN, source: { case: "content", value: new TextEncoder().encode("# Hello") } },
  outputFormat: OutputFormat.PDF,
});
download(res.content, res.fileName, res.mimeType);

// Start from a starter template, fill it and show field problems on a form.
const { starters } = await proteus.templates.listStarterTemplates({ category: "invoice" });
try {
  await proteus.templates.convertTemplate({ template: starters[0], data: myInvoice });
} catch (err) {
  for (const p of dataProblems(err)) console.log(p.path, p.message); // "items[2].qty must be a number"
}
```

The default transport is the Connect protocol over `fetch`, which browsers and
Node 18+ speak without extra packages; responses are JSON unless
`useBinaryFormat: true`. Pass `transport` (for example from
`@connectrpc/connect-node` with `createGrpcTransport`) to use something else;
add `apiKeyInterceptor(key)` to its interceptors then. Everything in the
generated code is re-exported: message types, enums, `ConversionService` and
`TemplateService` descriptors and the `*Schema` values for `create()`.

## Go

```sh
go get github.com/gsoultan/proteus-server-sdk/go
```

```go
import (
    proteus "github.com/gsoultan/proteus-server-sdk/go"
    proteusv1 "github.com/gsoultan/proteus-server-sdk/go/gen/proteus/v1"
)

c := proteus.New("https://proteus.example.com", os.Getenv("PROTEUS_KEY"))
res, err := c.Templates.ConvertTemplate(ctx, &proteusv1.ConvertTemplateRequest{Id: id, Data: data})
if problems := proteus.DataProblems(err); problems != nil {
    for _, p := range problems {
        fmt.Println(p.GetPath(), p.GetMessage())
    }
}
```

`New` speaks the Connect protocol over `http.DefaultClient`; `WithGRPC()` and
`WithGRPCWeb()` switch protocols (gRPC needs an HTTP/2 client: TLS, or h2c
with `http.Protocols.SetUnencryptedHTTP2`), `WithHTTPClient`,
`WithTransportOptions` (compression, codecs) and `WithInterceptors` tune the
rest. `c.Conversion` and `c.Templates` are the generated
`proteusv1connect` clients.

## Keeping up with the server

`proto/` is a copy of `proteus-server/proto`. After the API changes there:

```sh
scripts/sync-proto.sh ../proteus-server   # copies the protos, lints, regenerates both clients
(cd go && go test ./...) && (cd ts && bun install && bun run check && bun test)
```

Generation needs `buf`, `protoc-gen-go`, `protoc-gen-connect-go` on the path
and `bun install` in `ts/` (which provides `protoc-gen-es`). The generated code
is committed; CI fails when it is stale.
