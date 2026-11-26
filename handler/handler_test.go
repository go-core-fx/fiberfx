package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type uriParams struct {
	ID string `uri:"id" validate:"required"`
}

type bodyParams struct {
	Name string `json:"name" validate:"required"`
}

type queryParams struct {
	Name string `query:"name" validate:"required"`
}

type headerParams struct {
	Token string `header:"X-Token" validate:"required"`
}

// legacyParams keeps the Fiber v2 tag on an aliased field, which the v3 binder
// does not read.
type legacyParams struct {
	UserID string `params:"user_id" validate:"required"`
}

func newBase() *handler.Base {
	return &handler.Base{Validator: validator.New()}
}

type parseFunc func(c fiber.Ctx, base *handler.Base) error

func parseBody(c fiber.Ctx, base *handler.Base) error {
	var out bodyParams
	if err := base.BodyParserValidator(c, &out); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func parseQuery(c fiber.Ctx, base *handler.Base) error {
	var out queryParams
	if err := base.QueryParserValidator(c, &out); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func parseParam(c fiber.Ctx, base *handler.Base) error {
	var out legacyParams
	if err := base.ParamParserValidator(c, &out); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func parseHeader(c fiber.Ctx, base *handler.Base) error {
	var out headerParams
	if err := base.HeaderParserValidator(c, &out); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

func assertBadRequest(t *testing.T, app *fiber.App, req *http.Request, wantMessage string) {
	t.Helper()

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var errResp fiberfx.ErrorResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}
	if errResp.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, errResp.Code)
	}
	if !strings.Contains(errResp.Message, wantMessage) {
		t.Fatalf("expected message containing %q, got %q", wantMessage, errResp.Message)
	}
}

func TestParserValidatorsReturnBadRequestOnValidationFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		path        string
		body        string
		parse       parseFunc
		wantMessage string
	}{
		{
			name: "body", method: http.MethodPost, path: "/body", body: `{}`,
			parse: parseBody, wantMessage: "validation failed",
		},
		{
			name: "query", method: http.MethodGet, path: "/query",
			parse: parseQuery, wantMessage: "validation failed",
		},
		{
			name: "param", method: http.MethodGet, path: "/params/42",
			parse: parseParam, wantMessage: "validation failed",
		},
		{
			name: "header", method: http.MethodGet, path: "/header",
			parse: parseHeader, wantMessage: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := newBase()
			app := fiber.New(fiber.Config{ErrorHandler: fiberfx.NewJSONErrorHandler(zap.NewNop())})
			app.Add([]string{tt.method}, tt.path, func(c fiber.Ctx) error {
				return tt.parse(c, base)
			})

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			assertBadRequest(t, app, req, tt.wantMessage)
		})
	}
}

func TestBodyParserValidatorReturnsBadRequestOnMalformedJSON(t *testing.T) {
	t.Parallel()

	base := newBase()
	app := fiber.New(fiber.Config{ErrorHandler: fiberfx.NewJSONErrorHandler(zap.NewNop())})
	app.Post("/body", func(c fiber.Ctx) error {
		return parseBody(c, base)
	})

	req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	assertBadRequest(t, app, req, "failed to parse body")
}

func TestParamParserValidator(t *testing.T) {
	t.Parallel()

	base := newBase()

	app := fiber.New()
	app.Get("/users/:id", func(c fiber.Ctx) error {
		var out uriParams
		if err := base.ParamParserValidator(c, &out); err != nil {
			return err
		}

		return c.SendString(out.ID)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/users/42", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != "42" {
		t.Fatalf("expected 42, got %q", string(body))
	}
}

func TestHeaderParserValidator(t *testing.T) {
	t.Parallel()

	base := newBase()

	app := fiber.New()
	app.Get("/token", func(c fiber.Ctx) error {
		var out headerParams
		if err := base.HeaderParserValidator(c, &out); err != nil {
			return err
		}

		return c.SendString(out.Token)
	})

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("X-Token", "secret")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != "secret" {
		t.Fatalf("expected secret, got %q", string(body))
	}
}

// TestParamParserValidator_LegacyParamsTagDoesNotBind pins the Fiber v3
// binding contract: the v2 "params" tag is not read, so an aliased field still
// using it gets zero values instead of the route parameter.
func TestParamParserValidator_LegacyParamsTagDoesNotBind(t *testing.T) {
	t.Parallel()

	base := newBase()

	app := fiber.New()
	app.Get("/users/:user_id", func(c fiber.Ctx) error {
		var out legacyParams
		if err := base.ParamParserValidator(c, &out); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		return c.SendString(out.UserID)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/users/42", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}
}
