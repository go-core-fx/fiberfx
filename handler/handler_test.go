package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-core-fx/fiberfx/handler"
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
