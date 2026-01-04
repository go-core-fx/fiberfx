package handler

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type validatable interface {
	Validate() error
}

type Handler interface {
	Register(r fiber.Router)
}

type Base struct {
	Validator *validator.Validate
}

// BodyParserValidator parses the request body and validates the resulting object.
// It returns an error if parsing or validation fails.
func (b *Base) BodyParserValidator(ctx *fiber.Ctx, out any) error {
	if err := ctx.BodyParser(out); err != nil {
		return fmt.Errorf("failed to parse body: %w", err)
	}

	return b.validate(out)
}

// QueryParserValidator parses the query string and validates the resulting object.
// It returns an error if parsing or validation fails.
func (b *Base) QueryParserValidator(ctx *fiber.Ctx, out any) error {
	if err := ctx.QueryParser(out); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	return b.validate(out)
}

// ParamParserValidator parses the request parameters and validates the resulting object.
// It returns an error if parsing or validation fails.
func (b *Base) ParamParserValidator(ctx *fiber.Ctx, out any) error {
	if err := ctx.ParamsParser(out); err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}

	return b.validate(out)
}

// HeaderParserValidator parses the request headers and validates the resulting object.
// It returns an error if parsing or validation fails.
func (b *Base) HeaderParserValidator(ctx *fiber.Ctx, out any) error {
	if err := ctx.ReqHeaderParser(out); err != nil {
		return fmt.Errorf("failed to parse headers: %w", err)
	}

	return b.validate(out)
}

func (b *Base) validate(out any) error {
	if err := b.Validator.Var(out, "required,dive"); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if v, ok := out.(validatable); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return nil
}
