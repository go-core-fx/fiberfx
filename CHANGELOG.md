# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking Changes

- **Fiber v3** — the server is now built on `github.com/gofiber/fiber/v3`. In Fiber v3 the request context is an interface, so every handler signature changes from `func(c *fiber.Ctx) error` to `func(c fiber.Ctx) error`.
- **Request binding tags** — the Fiber v3 binder reads `uri` for route parameters and `header` for request headers. The Fiber v2 tag names `params` and `reqHeader` are no longer read, so request structs must be updated.
- **`fiberfx.New` signature** — the constructor takes a `*promclient.Registry` between the options and the logger: `fiberfx.New(config, options, registry, logger)`. The registry stays optional in the Fx graph.
- **`prometheus.Register` signature** — the function takes the registry to register into: `prometheus.Register(app, registry)`. A `nil` registry selects the process-wide default registry.
- **Metrics middleware** — the Prometheus middleware is now the Fiber v3 contrib package instead of `ansrivas/fiberprometheus`. `http_requests_total` and `http_request_duration_seconds` keep their names and labels, the in-flight gauge is renamed from `http_requests_in_progress_total` to `http_requests_in_progress`, and three new families are added: `http_requests_status_class_total`, `http_request_size_bytes` and `http_response_size_bytes`.
- **Proxy configuration** — the Fiber v3 `TrustProxy` / `TrustProxyConfig` options replace the v2 `EnableTrustedProxyCheck` / `TrustedProxies` pair. `fiberfx.Config` keeps the same `Proxies` and `ProxyHeader` fields.
- **`WithViews` argument** — the template engine is passed as a `fiber.Views` value instead of a `*fiber.Views` pointer.
- **Options for the Fiber configuration** — `WithFiberConfig` replaces `WithGetOnly`, `WithViews` and `WithErrorHandler`. The callback receives the assembled `fiber.Config` after the package defaults are applied, so the error handler, the template engine, the body limit and every other Fiber setting are configured in one place. Existing callers must move those three settings into the callback.
- **Minimum Go version** — the module now requires Go 1.26.
- **Startup banner** — Fiber v3 moved `DisableStartupMessage` from the application configuration to the listen configuration. The Fx module still suppresses the banner, but an application that builds the server with `fiberfx.New` and calls `app.Listen` itself now sees it.

### New Features

- **Configurable request log fields** — `WithRequestLogFields` replaces the fields written to the request log. An empty list keeps the package defaults.

## [0.6.0] - 2026-08-01

### New Features

- **Custom validation rules** — a request type can implement a `Validate() error` method for rules that struct tags cannot express. The method runs after the tag validation, and its error is reported with the same detail as a tag failure.

## [0.5.1] - 2026-07-07

### Bug Fixes

- **Escaped request paths** — request paths are always unescaped before routing, so a path containing percent-encoded characters now matches the route the client asked for.

## [0.5.0] - 2026-01-24

### New Features

#### OpenAPI Documentation

- **Swagger UI** — a new `openapi` package serves a Swagger UI and the generated specification from a `swag.Spec`. The handler can be enabled or disabled, and the public host and base path written into the specification are configurable. Registered on a group, it serves the UI at `<group>/index.html` and the specification at `<group>/doc.json`.

### Breaking Changes

- **`health.NewHandler` result** — the constructor returns `*health.Handler` instead of the `handler.Handler` interface, so the concrete type is available to the caller.

## [0.4.0] - 2026-01-09

### Breaking Changes

- **Health version source** — the health handler no longer takes a version. The version and release ID in the probe response are read from the health service, so they always match the build it reports. `health.NewHandler` now takes only the health service.

## [0.3.0] - 2026-01-09

### New Features

#### Request Validation

- **Validation middleware** — a new `validation` package validates request bodies and answers a failed validation with `400` and the failing fields, their tags and a human-readable message.
- **Validated body in the request context** — the validated request is stored in the request context, so a handler can receive it as an argument instead of parsing it again.
- **Inline validation** — a request body can be validated and handled in a single call, without the middleware.

## [0.2.0] - 2026-01-05

### New Features

#### Health Probes

- **Health endpoints** — a new `health` package serves liveness, readiness and startup probes backed by `healthfx`. A passing probe answers `200`, a failing one `503`, and the response reports the overall status, the version and the individual checks.

## [0.1.0] - 2025-12-04

### New Features

#### Prometheus Metrics

- **Metrics endpoint** — a new `prometheus` package exposes the Fiber Prometheus middleware, serving the request counters and the latency histogram at `/metrics`.

### Breaking Changes

- **Validator type** — `handler.Base.Validator` is now a `*validator.Validate` from `github.com/go-playground/validator/v10` instead of the wrapper type from `github.com/go-core-fx/validator`.

## [0.0.1] - 2025-10-31

### New Features

- **HTTP server module** — the initial release: an Fx module that provides a Fiber application, opens the listener on the Fx start hook and shuts the server down gracefully on the stop hook.
- **JSON error contract** — every error the server returns uses one shape, with a message, a status code and optional details.
- **Custom error handlers** — the JSON error handler can be replaced by a custom payload formatter, or by a template page rendered through the configured view engine.
- **Request logging** — every request is logged with its request ID, latency, status, method, URL, IP, user agent and error. Health and metrics paths are left out to keep the log readable.
- **Handler helpers** — the `handler` package binds and validates a body, query string, route parameters or request headers in a single call.
- **JSON content type** — the `jsonify` middleware sets the JSON content type on responses that carry an empty or already valid JSON body, and leaves other bodies untouched.
- **Status code middleware** — the `statuscode` package answers with a configurable status code and message.
