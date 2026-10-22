<a id="readme-top"></a>

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![Apache 2.0 License][license-shield]][license-url]

<br />
<div align="center">
  <h1>fiberfx</h1>
  <p>
    Uber Fx module for <code>Fiber v3</code> — HTTP server with structured logging, health probes, Prometheus metrics, OpenAPI docs and request validation.
  </p>
  <a href="https://github.com/go-core-fx/fiberfx/issues/new">Report Bug</a>
  &middot;
  <a href="https://github.com/go-core-fx/fiberfx/issues/new">Request Feature</a>
</div>

## Table of Contents
- [Table of Contents](#table-of-contents)
- [About The Project](#about-the-project)
	- [Features](#features)
	- [Built With](#built-with)
- [Getting Started](#getting-started)
	- [Prerequisites](#prerequisites)
	- [Installation](#installation)
- [Usage](#usage)
	- [Module Setup](#module-setup)
	- [Configuration Reference](#configuration-reference)
	- [Options](#options)
	- [Fiber Configuration](#fiber-configuration)
	- [Request Logging](#request-logging)
	- [Error Handling](#error-handling)
	- [Health Probes](#health-probes)
	- [Prometheus Metrics](#prometheus-metrics)
	- [OpenAPI Documentation](#openapi-documentation)
	- [Request Validation](#request-validation)
	- [Handler Base Helpers](#handler-base-helpers)
	- [JSON Content Type](#json-content-type)
	- [Status Code Middleware](#status-code-middleware)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)


---

## About The Project

`fiberfx` is an [Uber Fx](https://uber-go.github.io/fx/) module that builds and runs a [Fiber v3](https://github.com/gofiber/fiber) HTTP server. It wires the server into the Fx lifecycle — the listener starts on `OnStart` and shuts down gracefully on `OnStop` — and ships the cross-cutting pieces most services end up writing by hand:

- **Lifecycle-managed server** — `Module()` provides `*fiber.App`, opens the listener on the Fx start hook and drains connections on shutdown
- **Structured request logging** — request ID, latency, status, method, URL, IP, user agent and error fields, rendered through `zap`
- **JSON error contract** — one `ErrorResponse` shape for every error the server returns
- **Health probes** — liveness, readiness and startup endpoints backed by [`healthfx`](https://github.com/go-core-fx/healthfx)
- **Prometheus metrics** — opt-in `/metrics` endpoint, on an injected registry or the process-wide default
- **OpenAPI documentation** — Swagger UI and `doc.json` served from the generated spec
- **Request validation** — struct-tag validation with per-field error details and custom `Validate()` hooks

This is a library, not an application: it reads **no environment variables** and has no configuration file. Everything is configured through Go structs (`fiberfx.Config`, `fiberfx.Options` and the per-feature config types), which the host application supplies to the Fx graph.

### Features

- **Fiber v3** — built on `github.com/gofiber/fiber/v3`
- **Uber Fx module** — `fiberfx.Module()` provides `*fiber.App` and manages the listener lifecycle
- **Graceful shutdown** — `app.ShutdownWithContext(ctx)` on the Fx stop hook; a listener failure triggers `fx.Shutdown`
- **Request ID propagation** — the `X-Request-ID` response header is set per request and logged
- **Panic recovery** — the Fiber `recover` middleware is installed by default
- **IP validation** — enabled by default, so `c.IP()` returns a single validated address instead of a raw `X-Forwarded-For` chain
- **Always-unescaped paths** — request paths are unescaped before routing
- **Proxy trust** — optional trusted-proxy list and proxy header
- **Full Fiber configuration** — any `fiber.Config` field (error handler, template engine, body limit, GET-only mode) can be overridden through a callback
- **Configurable log fields** — the request log field set can be replaced
- **Health, metrics and OpenAPI** — the three operational endpoints every service needs

### Built With

- [![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
- [![Uber Fx](https://img.shields.io/badge/Uber%20Fx-000000?style=for-the-badge)](https://uber-go.github.io/fx/)
- [![Fiber](https://img.shields.io/badge/Fiber-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://github.com/gofiber/fiber)
- [![Zap](https://img.shields.io/badge/Zap-FFAC00?style=for-the-badge&logo=go&logoColor=black)](https://github.com/uber-go/zap)
- [![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=prometheus&logoColor=white)](https://github.com/prometheus/client_golang)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Getting Started

### Prerequisites

- Go 1.26+
- An application using [Uber Fx](https://uber-go.github.io/fx/) for dependency injection
- A `*zap.Logger` in the graph (for example from [`github.com/go-core-fx/logger`](https://github.com/go-core-fx/logger))

### Installation

```sh
go get github.com/go-core-fx/fiberfx@latest
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Usage

### Module Setup

`fiberfx.Module()` provides `*fiber.App`. The graph must also contain `fiberfx.Config`, `fiberfx.Options` and a base `*zap.Logger`; a `*promclient.Registry` is optional and only needed for a custom metrics registry.

```go
package main

import (
	"github.com/go-core-fx/fiberfx"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		// Base logger. The module decorates it with the name "http".
		fx.Provide(zap.NewProduction),

		fx.Provide(func() fiberfx.Config {
			return fiberfx.Config{
				Address: ":8080",
			}
		}),

		fx.Provide(func(logger *zap.Logger) fiberfx.Options {
			var opts fiberfx.Options
			opts.WithFiberConfig(func(cfg *fiber.Config) {
				cfg.ErrorHandler = fiberfx.NewJSONErrorHandler(logger)
			})
			opts.WithMetrics()
			return opts
		}),

		fiberfx.Module(),
	).Run()
}
```

`Options` is configured through pointer receivers, so build it as a local variable and supply the value:

```go
opts := fiberfx.Options{}
opts.WithMetrics()

fx.Supply(opts)
```

The module starts the listener on the Fx start hook and shuts it down on the stop hook, logging
`starting server`, `server listening`, `shutting down server` and `server shutdown completed` through the
`http` logger.

The `*fiber.App` is available for route registration, either by injecting it or by populating it from the graph:

```go
var app *fiber.App

fx.New(
	// ...
	fiberfx.Module(),
	fx.Populate(&app),
).Run()
```

### Configuration Reference

| Field         | Type       | Default | Description                                                                                      |
| ------------- | ---------- | ------- | ------------------------------------------------------------------------------------------------ |
| `Address`     | `string`   | —       | Listen address passed to `net.Listen` (for example `":8080"` or `"127.0.0.1:8080"`). No default. |
| `ProxyHeader` | `string`   | `""`    | Request header holding the client IP. Only used when `Proxies` is non-empty.                     |
| `Proxies`     | `[]string` | `nil`   | Trusted proxy IPs or CIDR ranges. When empty, proxy headers are ignored entirely.                |

Proxy trust is enabled only when `Proxies` is non-empty — with an empty list the client IP is taken from the
remote address and `ProxyHeader` has no effect.

### Options

| Option                            | Effect                                                                                                                      |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `WithFiberConfig(configure)`      | Applies changes to the Fiber configuration after the package defaults — see [Fiber Configuration](#fiber-configuration).    |
| `WithRequestLogFields(fields...)` | Replaces the fields written to the request log. An empty list keeps the defaults — see [Request Logging](#request-logging). |
| `WithMetrics()`                   | Mounts the Prometheus metrics endpoint at `/metrics`.                                                                       |

### Fiber Configuration

`WithFiberConfig` hands the assembled `fiber.Config` to a callback, so anything Fiber supports can be
overridden — the error handler, the template engine, the body limit, the read/write timeouts and so on:

```go
opts := fiberfx.Options{}
opts.WithFiberConfig(func(cfg *fiber.Config) {
	cfg.ErrorHandler = fiberfx.NewJSONErrorHandler(logger)
	cfg.Views = views
	cfg.BodyLimit = 30 * 1024 * 1024
	cfg.GETOnly = true
})
```

The callback runs after the package defaults are applied (`EnableIPValidation`, the trusted proxies from
`fiberfx.Config` and `UnescapePath`), so it can override them.

### Request Logging

Every request is logged with `requestId`, `latency`, `status`, `method`, `url`, `ip`, `ua`, `body` and `error`.
`WithRequestLogFields` replaces that list; an empty list keeps the defaults:

```go
opts := fiberfx.Options{}
opts.WithRequestLogFields("requestId", "status", "error")
```

Health and metrics paths are never logged.

### Error Handling

All errors the server returns use one shape:

```json
{
  "message": "Not Found",
  "code": 404
}
```

`message` and `code` are always present; `details` is optional and carries structured context when there is any.

Three handlers are provided, and are installed through `WithFiberConfig`:

```go
// JSON error response — the default choice for APIs.
fiberfx.NewJSONErrorHandler(logger)

// JSON error response with a custom payload.
fiberfx.NewCustomJSONErrorHandler(logger, func(err error, code int) any {
	return fiberfx.NewErrorResponse("something went wrong", code, nil)
})

// HTML error page rendered through a fiber.Views engine.
fiberfx.NewViewsErrorHandler(logger, "error", "layouts/main")
```

Status codes are derived from the error:

| Source                     | Status                                      |
| -------------------------- | ------------------------------------------- |
| `*fiber.Error`             | the error's own code                        |
| `context.Canceled`         | `499` (`fiberfx.StatusClientClosedRequest`) |
| `context.DeadlineExceeded` | `408`                                       |
| anything else              | `500`                                       |

For a `4xx` the raw error message is returned, except for `408` and `499` which get a fixed text; for a `5xx`
the standard status text is returned instead, so internal details never leak to clients. Errors with a `5xx` status
are logged through the logger that was passed to the handler.

`499` is non-standard but widely used. If your infrastructure only accepts standard codes, map
`fiberfx.StatusClientClosedRequest` to `fiber.StatusRequestTimeout` in a custom handler.

### Health Probes

The `health` package exposes liveness, readiness and startup probes backed by
[`healthfx`](https://github.com/go-core-fx/healthfx):

```go
app := fiberfx.New(cfg, opts, registry, logger)

// /health, /health/live, /health/ready, /health/startup
health.NewHandler(healthSvc).Register(app)
```

| Route             | Meaning                           |
| ----------------- | --------------------------------- |
| `/health`         | alias of `/health/live`           |
| `/health/live`    | the process is running            |
| `/health/ready`   | the process can serve traffic     |
| `/health/startup` | the process finished initializing |

A failed probe responds `503`, a passing one `200`:

```json
{
  "status": "pass",
  "version": "1.0.0",
  "releaseId": 42,
  "checks": {
    "system:goroutines": {
      "description": "Number of goroutines",
      "observedUnit": "goroutines",
      "observedValue": 12,
      "status": "pass"
    }
  }
}
```

`version` and `releaseId` come from the `healthfx.Version` held by the health service. Check names are prefixed
with the provider name. The overall `status` is `pass`, `warn` or `fail`; only `fail` maps to `503`.

### Prometheus Metrics

`WithMetrics()` mounts the Fiber Prometheus middleware at `/metrics`:

```go
opts := fiberfx.Options{}
opts.WithMetrics()
```

Metrics land in the process-wide default registry unless a `*promclient.Registry` is present in the graph:

```go
fx.Provide(func() *promclient.Registry {
	return promclient.NewRegistry()
})
```

An injected registry starts empty, so the middleware registers the Go and process collectors into it itself; the
default registry already contains them. The exposed families are `http_requests_total`,
`http_requests_status_class_total`, `http_request_duration_seconds`, `http_request_size_bytes`,
`http_response_size_bytes` and `http_requests_in_progress`.

### OpenAPI Documentation

The `openapi` package serves a Swagger UI and the generated spec from a `swag.Spec`:

```go
spec := &swag.Spec{
	SwaggerTemplate: `{
		"swagger": "2.0",
		"info": {"title": "My API", "version": "{{.Version}}"},
		"host": "{{.Host}}",
		"basePath": "{{.BasePath}}",
		"paths": {}
	}`,
}
swag.Register(swag.Name, spec)

openapi.NewHandler(
	openapi.Config{
		Enabled:    true,
		PublicHost: "api.example.com",
		PublicPath: "/v1",
	},
	healthfx.Version{Version: "1.0.0", ReleaseID: 1},
	spec,
).Register(app.Group("/swagger"))
```

| Field        | Default | Description                                                                       |
| ------------ | ------- | --------------------------------------------------------------------------------- |
| `Enabled`    | `false` | When false the handler registers no routes at all.                                |
| `PublicHost` | `""`    | Host written into the spec. When empty, the host of the first request is used.    |
| `PublicPath` | `""`    | Base path written into the spec. When empty, the base path from the spec is kept. |

With the group mounted at `/swagger`, the UI is served at `/swagger/index.html` and the spec at
`/swagger/doc.json`. Responses carry a weak ETag.

### Request Validation

The `validation` package validates request bodies and returns per-field error details:

```go
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

app.Post("/users",
	validation.Middleware,
	validation.NewBody[CreateUserRequest](validator.New()),
	validation.DecorateWithBody(func(c fiber.Ctx, body *CreateUserRequest) error {
		return c.Status(fiber.StatusCreated).JSON(body)
	}),
)
```

`validation.Middleware` converts a validation failure into a `400` response:

```json
{
  "message": "validation failed: name: name is required",
  "code": 400,
  "details": [
    {
      "field": "name",
      "tag": "required",
      "message": "name is required"
    }
  ]
}
```

| Function                                        | Purpose                                                               |
| ----------------------------------------------- | --------------------------------------------------------------------- |
| `validation.Middleware`                         | Middleware that renders validation failures as a `400` JSON response. |
| `validation.NewBody[T](validate)`               | Validates the body and stores the result in the request context.      |
| `validation.ValidateBody[T](c, validate)`       | Validates the body and returns it directly.                           |
| `validation.GetValidatedBody[T](c)`             | Reads the validated body stored by `NewBody`.                         |
| `validation.DecorateWithBody[T](h)`             | Wraps a handler so it receives the validated body.                    |
| `validation.DecorateWithBodyEx[T](validate, h)` | Validates and wraps in one step, without the middleware.              |

A request type may implement `Validate() error` for rules that struct tags cannot express. It runs after the
tag validation and its error is reported the same way:

```go
func (r *CreateUserRequest) Validate() error {
	if r.Name == "admin" {
		return errors.New("name is reserved")
	}
	return nil
}
```

### Handler Base Helpers

`handler.Base` binds and validates a request in one call. It is useful when a route needs the parsed values
directly instead of a validated body in the context:

```go
base := &handler.Base{Validator: validator.New()}

app.Get("/users/:id", func(c fiber.Ctx) error {
	var out struct {
		ID string `uri:"id" validate:"required"`
	}
	if err := base.ParamParserValidator(c, &out); err != nil {
		return err
	}
	return c.SendString(out.ID)
})
```

| Method                          | Binds from       | Struct tag (Fiber v3) | Fiber v2 tag |
| ------------------------------- | ---------------- | --------------------- | ------------ |
| `BodyParserValidator(c, out)`   | request body     | `json`                | `json`       |
| `QueryParserValidator(c, out)`  | query string     | `query`               | `query`      |
| `ParamParserValidator(c, out)`  | route parameters | `uri`                 | `params`     |
| `HeaderParserValidator(c, out)` | request headers  | `header`              | `reqHeader`  |

The Fiber v3 binder reads `uri` and `header`; the v2 tag names are not read.

### JSON Content Type

`jsonify` makes responses advertise `application/json` when the client accepts JSON and the body is empty or
already valid JSON. Bodies that are not valid JSON are left untouched:

```go
app.Use(jsonify.New())
```

### Status Code Middleware

`statuscode` answers with a fixed status code and message, for example to return `404` for unmatched routes
instead of Fiber's default response:

```go
app.Use(statuscode.New(statuscode.Config{
	StatusCode:    fiber.StatusNotFound,
	StatusMessage: "Not Found",
}))
```

Both fields are optional; they default to `404` and `Not Found`. `Next` skips the middleware for matching
requests:

```go
app.Use(statuscode.New(statuscode.Config{
	Next: func(c fiber.Ctx) bool {
		return c.Path() == "/health"
	},
}))
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Roadmap

- [x] Fiber v3 server wired into the Fx lifecycle
- [x] Graceful shutdown
- [x] Structured request logging with request ID
- [x] Configurable request log fields
- [x] JSON error contract and customizable error handlers
- [x] Health probes (liveness, readiness, startup)
- [x] Prometheus metrics with an optional injected registry
- [x] OpenAPI / Swagger UI
- [x] Request validation with per-field errors
- [x] Trusted proxy support
- [x] Full Fiber configuration override

See the [open issues](https://github.com/go-core-fx/fiberfx/issues) for a full list of proposed features.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## License

Distributed under the Apache License 2.0. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Acknowledgments

- [Uber Fx](https://uber-go.github.io/fx/) — dependency injection framework
- [Fiber](https://github.com/gofiber/fiber) — HTTP framework
- [healthfx](https://github.com/go-core-fx/healthfx) — health check service
- [go-playground/validator](https://github.com/go-playground/validator) — struct tag validation
- [swaggo](https://github.com/swaggo/swag) — Swagger specification generation
- [Best-README-Template](https://github.com/othneildrew/Best-README-Template) — README structure

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
[contributors-shield]: https://img.shields.io/github/contributors/go-core-fx/fiberfx.svg?style=for-the-badge
[contributors-url]: https://github.com/go-core-fx/fiberfx/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/go-core-fx/fiberfx.svg?style=for-the-badge
[forks-url]: https://github.com/go-core-fx/fiberfx/network/members
[stars-shield]: https://img.shields.io/github/stars/go-core-fx/fiberfx.svg?style=for-the-badge
[stars-url]: https://github.com/go-core-fx/fiberfx/stargazers
[issues-shield]: https://img.shields.io/github/issues/go-core-fx/fiberfx.svg?style=for-the-badge
[issues-url]: https://github.com/go-core-fx/fiberfx/issues
[license-shield]: https://img.shields.io/github/license/go-core-fx/fiberfx.svg?style=for-the-badge
[license-url]: https://github.com/go-core-fx/fiberfx/blob/master/LICENSE
