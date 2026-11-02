// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

import { describe, expect, test } from "bun:test";
import { Code, ConnectError, createRouterTransport } from "@connectrpc/connect";

import {
  ConversionService,
  TemplateDataErrorSchema,
  TemplateService,
  apiKeyInterceptor,
  createProteus,
  dataProblems,
  problemsByPath,
} from "./index.js";

describe("createProteus", () => {
  let auth: string | null = null;
  // A transport given to createProteus replaces the default one, and with
  // it the API key interceptor; the last test adds it by hand.
  const transport = createRouterTransport(
    ({ service }) => {
      service(ConversionService, {
        listFormats(_req, ctx) {
          auth = ctx.requestHeader.get("Authorization");
          return { conversions: [{ mimeType: "application/pdf" }] };
        },
      });
      service(TemplateService, {
        renderTemplate() {
          throw new ConnectError("data does not match", Code.InvalidArgument, undefined, [
            {
              desc: TemplateDataErrorSchema,
              value: {
                problems: [
                  { path: "items[2].qty", message: "must be a number" },
                  { path: "items[2].qty", message: "is required" },
                ],
              },
            },
          ]);
        },
      });
    },
  );

  test("calls both services", async () => {
    const proteus = createProteus({ baseUrl: "http://unused", apiKey: "k", transport });
    const res = await proteus.conversion.listFormats({});
    expect(res.conversions).toHaveLength(1);
    expect(proteus.templates).toBeDefined();
  });

  test("decodes template data problems", async () => {
    const proteus = createProteus({ baseUrl: "http://unused", transport });
    let caught: unknown;
    try {
      await proteus.templates.renderTemplate({});
    } catch (e) {
      caught = e;
    }
    const problems = dataProblems(caught);
    expect(problems.map((p) => p.path)).toEqual(["items[2].qty", "items[2].qty"]);
    expect(problemsByPath(problems).get("items[2].qty")).toEqual(["must be a number", "is required"]);
    expect(dataProblems(new Error("x"))).toEqual([]);
  });

  test("the key interceptor sets the bearer header", async () => {
    const withKey = createRouterTransport(
      ({ service }) => {
        service(ConversionService, {
          listFormats(_req, ctx) {
            auth = ctx.requestHeader.get("Authorization");
            return {};
          },
        });
      },
      { transport: { interceptors: [apiKeyInterceptor("secret")] } },
    );
    await createProteus({ baseUrl: "http://unused", transport: withKey }).conversion.listFormats({});
    expect(auth).toBe("Bearer secret");

    const noKey = createRouterTransport(
      ({ service }) => {
        service(ConversionService, {
          listFormats(_req, ctx) {
            auth = ctx.requestHeader.get("Authorization");
            return {};
          },
        });
      },
      { transport: { interceptors: [apiKeyInterceptor("")] } },
    );
    await createProteus({ baseUrl: "http://unused", transport: noKey }).conversion.listFormats({});
    expect(auth).toBeNull();
  });
});
