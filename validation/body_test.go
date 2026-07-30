package validation_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/fiberfx/validation"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

func newTestValidator() *validator.Validate {
	return validator.New()
}

type validatableBody struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age"  validate:"min=0"`
}

func (b *validatableBody) Validate() error {
	if b.Age < 18 {
		return errors.New("must be at least 18 years old")
	}
	return nil
}

type validatableBodyWithValidationErrors struct {
	Name string `json:"name" validate:"required"`
}

func (b *validatableBodyWithValidationErrors) Validate() error {
	return newTestValidator().Var("", "required")
}

type nonValidatableBody struct {
	Name string `json:"name" validate:"required"`
}

func okHandler(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

func TestValidateBody_ValidatableReturnsNil(t *testing.T) {
	app := fiber.New()
	app.Post("/", validation.Middleware, validation.NewBody[validatableBody](newTestValidator()), okHandler)

	body := `{"name":"John","age":25}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestValidateBody_ValidatableReturnsPlainError(t *testing.T) {
	app := fiber.New()
	app.Post("/", validation.Middleware, validation.NewBody[validatableBody](newTestValidator()), okHandler)

	body := `{"name":"John","age":15}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var errResp fiberfx.ErrorResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
		t.Fatalf("failed to decode response: %v", decodeErr)
	}
	if errResp.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, errResp.Code)
	}
	if errResp.Message != "validation failed: _: must be at least 18 years old" {
		t.Fatalf("unexpected message: %s", errResp.Message)
	}
}

func TestValidateBody_ValidatableReturnsValidationErrors(t *testing.T) {
	app := fiber.New()
	app.Post(
		"/",
		validation.Middleware,
		validation.NewBody[validatableBodyWithValidationErrors](newTestValidator()),
		okHandler,
	)

	body := `{"name":"John"}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var errResp fiberfx.ErrorResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
		t.Fatalf("failed to decode response: %v", decodeErr)
	}
	if errResp.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, errResp.Code)
	}
}

func TestValidateBody_NonValidatable(t *testing.T) {
	app := fiber.New()
	app.Post("/", validation.Middleware, validation.NewBody[nonValidatableBody](newTestValidator()), okHandler)

	body := `{"name":"John"}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestValidateBody_NonValidatable_FailsStructTag(t *testing.T) {
	app := fiber.New()
	app.Post("/", validation.Middleware, validation.NewBody[nonValidatableBody](newTestValidator()), okHandler)

	body := `{}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	var errResp fiberfx.ErrorResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr != nil {
		t.Fatalf("failed to decode response: %v", decodeErr)
	}
	if errResp.Code != fiber.StatusBadRequest {
		t.Fatalf("expected code %d, got %d", fiber.StatusBadRequest, errResp.Code)
	}
}

func TestNewErrors_PlainError(t *testing.T) {
	plainErr := errors.New("something went wrong")
	result := validation.NewErrors(plainErr)

	errs, ok := lo.ErrorsAs[validation.Errors](result)
	if !ok {
		t.Fatal("expected Errors type")
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Field != "_" {
		t.Fatalf("expected field '_', got %s", errs[0].Field)
	}
	if errs[0].Message != "something went wrong" {
		t.Fatalf("expected message 'something went wrong', got %s", errs[0].Message)
	}
}

func TestNewErrors_ValidationErrors(t *testing.T) {
	ve := newTestValidator().Var("", "required")
	result := validation.NewErrors(ve)

	errs, ok := lo.ErrorsAs[validation.Errors](result)
	if !ok {
		t.Fatal("expected Errors type")
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Tag != "required" {
		t.Fatalf("expected tag 'required', got %s", errs[0].Tag)
	}
}
