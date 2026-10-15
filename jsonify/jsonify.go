package jsonify

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func New() fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}

		contentType := string(c.Response().Header.ContentType())
		if strings.Contains(contentType, fiber.MIMEApplicationJSON) {
			return nil
		}

		body := c.Response().Body()

		if c.Response().StatusCode() < fiber.StatusBadRequest {
			// Only normalize if client accepts JSON.
			if !strings.Contains(c.Get(fiber.HeaderAccept), fiber.MIMEApplicationJSON) {
				return nil
			}
			if len(body) == 0 {
				c.Type(fiber.MIMEApplicationJSON)
				return nil
			}
			// If body is already valid JSON, just set the header; do not re-marshal []byte.
			if json.Valid(body) {
				c.Type(fiber.MIMEApplicationJSON)
				return nil
			}
			// Leave non-JSON bodies untouched to avoid corrupting content.
			return nil
		}

		return nil
	}
}
