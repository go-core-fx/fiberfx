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
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type uriParams struct {
	ID string `uri:"id" validate:"required"`
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

func TestHandlerBaseValidationMiddlewareReturnsBadRequest(t *testing.T) {
	t.Parallel()

	base := newBase()

	app := fiber.New()
	app.Post("/users",
		validation.Middleware,
		func(c fiber.Ctx) error {
			var out struct {
				Name string `json:"name" validate:"required"`
			}
			if err := base.BodyParserValidator(c, &out); err != nil {
				return err
			}

			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var payload fiberfx.ErrorResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}
	if payload.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, payload.Code)
	}
	if details, ok := payload.Details.([]any); !ok || len(details) == 0 {
		t.Fatalf("expected details array, got %#v", payload.Details)
	}
}

func TestHandlerBaseBindErrorMiddlewareResponseShape(t *testing.T) {
	t.Parallel()

	base := newBase()

	app := fiber.New()
	app.Post("/users",
		validation.Middleware,
		func(c fiber.Ctx) error {
			var out struct {
				Name string `json:"name" validate:"required"`
			}
			if err := base.BodyParserValidator(c, &out); err != nil {
				return err
			}

			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var payload struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
		Details []struct {
			Field   string `json:"field"`
			Tag     string `json:"tag"`
			Message string `json:"message"`
		} `json:"details"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}
	if payload.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, payload.Code)
	}
	if len(payload.Details) != 1 {
		t.Fatalf("expected one detail, got %d", len(payload.Details))
	}
	if payload.Details[0].Tag != "body" {
		t.Fatalf("expected tag body, got %q", payload.Details[0].Tag)
	}
	if payload.Details[0].Message != "bind from body: unexpected end of JSON input" {
		t.Fatalf("unexpected detail message %q", payload.Details[0].Message)
	}
	if payload.Message != "validation failed: : bind from body: unexpected end of JSON input" {
		t.Fatalf("unexpected top-level message %q", payload.Message)
	}
}
