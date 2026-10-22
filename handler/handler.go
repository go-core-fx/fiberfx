package handler

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
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
func (b *Base) BodyParserValidator(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().Body(out); err != nil {
		return fmt.Errorf("failed to parse body: %w", err)
	}

	return b.validate(out)
}

// QueryParserValidator parses the query string and validates the resulting object.
// It returns an error if parsing or validation fails.
func (b *Base) QueryParserValidator(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().Query(out); err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	return b.validate(out)
}

// ParamParserValidator parses the request parameters and validates the resulting object.
// It returns an error if parsing or validation fails.
//
// The route parameter is bound from the "uri" struct tag, the Fiber v3 name for
// the tag Fiber v2 read as "params".
func (b *Base) ParamParserValidator(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().URI(out); err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}

	return b.validate(out)
}

// HeaderParserValidator parses the request headers and validates the resulting object.
// It returns an error if parsing or validation fails.
//
// The header is bound from the "header" struct tag, the Fiber v3 name for the
// tag Fiber v2 read as "reqHeader".
func (b *Base) HeaderParserValidator(ctx fiber.Ctx, out any) error {
	if err := ctx.Bind().Header(out); err != nil {
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
