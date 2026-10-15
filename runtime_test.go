package fiberfx_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/fiberfx/health"
	"github.com/go-core-fx/fiberfx/jsonify"
	"github.com/go-core-fx/fiberfx/openapi"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-core-fx/healthfx"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/swaggo/swag"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type smokeBody struct {
	Name string `json:"name" validate:"required"`
}

// smokeSwaggerTemplate is a minimal but valid Swagger document, so the
// smoke test asserts on a usable spec instead of an empty one.
const smokeSwaggerTemplate = `{
  "swagger": "2.0",
  "info": {"title": "{{.Title}}", "version": "{{.Version}}"},
  "host": "{{.Host}}",
  "basePath": "{{.BasePath}}",
  "paths": {}
}`

func newSmokeApp(t *testing.T, withMetrics bool, registry *promclient.Registry) *fiber.App {
	t.Helper()

	var opts fiberfx.Options
	if withMetrics {
		opts.WithMetrics()
	}

	app := fiberfx.New(fiberfx.Config{}, opts, registry, zap.NewNop())

	app.Use(jsonify.New())

	svc := healthfx.NewService(nil, healthfx.Version{Version: "test", ReleaseID: 1}, zap.NewNop())
	health.NewHandler(svc).Register(app)

	app.Post(
		"/validate",
		validation.Middleware,
		validation.NewBody[smokeBody](validator.New()),
		func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	return app
}

func request(t *testing.T, app *fiber.App, method, target, body string, headers map[string]string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	return resp
}

func TestSmokeHealthLiveness(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	resp := request(t, app, http.MethodGet, "/health", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != "pass" {
		t.Fatalf("expected status pass, got %q", payload.Status)
	}
}

func TestSmokeValidationError(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	resp := request(t, app, http.MethodPost, "/validate", `{}`, map[string]string{
		fiber.HeaderContentType: fiber.MIMEApplicationJSON,
	})
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var payload fiberfx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, payload.Code)
	}
}

func TestSmokeValidationOK(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	resp := request(t, app, http.MethodPost, "/validate", `{"name":"John"}`, map[string]string{
		fiber.HeaderContentType: fiber.MIMEApplicationJSON,
	})
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestSmokeMetrics(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, true, nil)

	// Generate at least one instrumented request before scraping.
	healthResp := request(t, app, http.MethodGet, "/health", "", nil)
	healthResp.Body.Close()

	resp := request(t, app, http.MethodGet, "/metrics", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "http_requests_total") {
		t.Fatalf("expected http_requests_total in metrics output")
	}
}

func TestSmokeMetricsInjectedRegistry(t *testing.T) {
	t.Parallel()

	registry := promclient.NewRegistry()
	app := newSmokeApp(t, true, registry)

	// Generate at least one instrumented request before scraping.
	healthResp := request(t, app, http.MethodGet, "/health", "", nil)
	healthResp.Body.Close()

	resp := request(t, app, http.MethodGet, "/metrics", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "http_requests_total") {
		t.Fatalf("expected http_requests_total in metrics output")
	}

	// The middleware registered into the injected registry, not the default one.
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	var found bool
	for _, family := range families {
		if family.GetName() == "http_requests_total" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected http_requests_total in the injected registry")
	}
}

func TestModuleGraphWithoutRegistry(t *testing.T) {
	t.Parallel()

	// The registry is an optional dependency, so the graph boots without one.
	fxtest.New(t,
		fiberfx.Module(),
		fx.Supply(zap.NewNop()),
		fx.Supply(fiberfx.Config{}),
		fx.Supply(fiberfx.Options{}),
	)
}

func TestModuleGraphWithRegistry(t *testing.T) {
	t.Parallel()

	opts := fiberfx.Options{}
	opts.WithMetrics()

	registry := promclient.NewRegistry()

	var app *fiber.App

	fxtest.New(t,
		fiberfx.Module(),
		fx.Supply(zap.NewNop()),
		fx.Supply(fiberfx.Config{}),
		fx.Supply(opts),
		fx.Supply(registry),
		fx.Populate(&app),
	)

	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	// The middleware registered into the registry from the graph, not the default one.
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	var found bool
	for _, family := range families {
		if family.GetName() == "http_requests_total" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected http_requests_total in the injected registry")
	}
}

func TestSmokeJSONify(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	resp := request(t, app, http.MethodGet, "/health", "", map[string]string{
		fiber.HeaderAccept: fiber.MIMEApplicationJSON,
	})
	defer resp.Body.Close()

	if ct := resp.Header.Get(fiber.HeaderContentType); !strings.Contains(ct, fiber.MIMEApplicationJSON) {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
}

func TestSmokeNotFound(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	resp := request(t, app, http.MethodGet, "/does-not-exist", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected %d, got %d", fiber.StatusNotFound, resp.StatusCode)
	}
}

func TestSmokeOpenAPI(t *testing.T) {
	t.Parallel()

	app := newSmokeApp(t, false, nil)

	spec := &swag.Spec{
		Version:          "test",
		Host:             "",
		BasePath:         "",
		Schemes:          nil,
		Title:            "Test",
		Description:      "",
		InfoInstanceName: "",
		SwaggerTemplate:  smokeSwaggerTemplate,
		LeftDelim:        "",
		RightDelim:       "",
	}
	swag.Register(swag.Name, spec)

	openapi.NewHandler(
		openapi.Config{Enabled: true, PublicHost: "api.example.com", PublicPath: "/v1"},
		healthfx.Version{Version: "test", ReleaseID: 0, BuildDate: "", GitCommit: "", GoVersion: ""},
		spec,
	).Register(app.Group("/swagger"))

	resp := request(t, app, http.MethodGet, "/swagger/doc.json", "", nil)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("/swagger/doc.json: expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	var doc struct {
		Swagger  string `json:"swagger"`
		Host     string `json:"host"`
		BasePath string `json:"basePath"`
		Info     struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode doc.json: %v", err)
	}

	if doc.Swagger != "2.0" {
		t.Fatalf("expected swagger 2.0, got %q", doc.Swagger)
	}
	if doc.Info.Title != "Test" {
		t.Fatalf("expected title Test, got %q", doc.Info.Title)
	}
	if doc.Info.Version != "test" {
		t.Fatalf("expected version test, got %q", doc.Info.Version)
	}
	if doc.Host != "api.example.com" {
		t.Fatalf("expected host api.example.com, got %q", doc.Host)
	}
	if doc.BasePath != "/v1" {
		t.Fatalf("expected basePath /v1, got %q", doc.BasePath)
	}

	indexResp := request(t, app, http.MethodGet, "/swagger/index.html", "", nil)
	defer indexResp.Body.Close()

	if indexResp.StatusCode != fiber.StatusOK {
		t.Fatalf("/swagger/index.html: expected %d, got %d", fiber.StatusOK, indexResp.StatusCode)
	}
}
