// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

/**
 * TypeScript client for proteus-server.
 *
 * ```ts
 * const proteus = createProteus({ baseUrl: "https://proteus.example.com", apiKey });
 * const res = await proteus.conversion.convert({ document: {...}, outputFormat: OutputFormat.PDF });
 * ```
 *
 * The default transport is the Connect protocol over fetch, which browsers
 * and Node 18+ speak without extra packages. Pass your own `transport` (for
 * example from `@connectrpc/connect-node` with gRPC) to change that.
 */
import {
  Code,
  ConnectError,
  createClient,
  type Client,
  type Interceptor,
  type Transport,
} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { ConversionService } from "./gen/proteus/v1/conversion_pb.js";
import {
  TemplateDataErrorSchema,
  TemplateService,
  type DataProblem,
} from "./gen/proteus/v1/template_pb.js";

export * from "./gen/proteus/v1/conversion_pb.js";
export * from "./gen/proteus/v1/template_pb.js";
export { Code, ConnectError } from "@connectrpc/connect";

export interface ProteusOptions {
  /** Scheme and host of the server, such as "https://proteus.example.com". */
  baseUrl: string;
  /** API key, sent as a bearer token. Omit for a server without keys. */
  apiKey?: string;
  /** Replaces the default Connect-over-fetch transport; baseUrl is then unused. */
  transport?: Transport;
  /** Interceptors run after the API key one. */
  interceptors?: Interceptor[];
  /** Protobuf binary instead of JSON on the wire; smaller for documents. */
  useBinaryFormat?: boolean;
  /** A fetch implementation; the global one otherwise. */
  fetch?: typeof globalThis.fetch;
}

export interface Proteus {
  conversion: Client<typeof ConversionService>;
  templates: Client<typeof TemplateService>;
  transport: Transport;
}

/** An interceptor that sends the API key as a bearer token. */
export function apiKeyInterceptor(apiKey: string): Interceptor {
  return (next) => (req) => {
    if (apiKey) {
      req.header.set("Authorization", `Bearer ${apiKey}`);
    }
    return next(req);
  };
}

/** Creates clients for both services over one transport. */
export function createProteus(options: ProteusOptions): Proteus {
  const interceptors = [apiKeyInterceptor(options.apiKey ?? ""), ...(options.interceptors ?? [])];
  const transport =
    options.transport ??
    createConnectTransport({
      baseUrl: options.baseUrl,
      useBinaryFormat: options.useBinaryFormat ?? false,
      interceptors,
      ...(options.fetch ? { fetch: options.fetch } : {}),
    });
  return {
    conversion: createClient(ConversionService, transport),
    templates: createClient(TemplateService, transport),
    transport,
  };
}

/**
 * The field problems the server attached to an INVALID_ARGUMENT error from a
 * template render, or an empty list. Each one names a data path such as
 * "items[2].qty".
 */
export function dataProblems(err: unknown): DataProblem[] {
  const cerr = ConnectError.from(err);
  if (cerr.code !== Code.InvalidArgument) {
    return [];
  }
  return cerr.findDetails(TemplateDataErrorSchema).flatMap((d) => d.problems);
}

/** Groups problems by path, for showing them next to form fields. */
export function problemsByPath(problems: DataProblem[]): Map<string, string[]> {
  const out = new Map<string, string[]>();
  for (const p of problems) {
    const list = out.get(p.path) ?? [];
    list.push(p.message);
    out.set(p.path, list);
  }
  return out;
}
