package statuscode

import "github.com/gofiber/fiber/v2"

const (
	defaultStatusCode    = fiber.StatusNotFound
	defaultStatusMessage = "Not Found"
)

// Config defines the config for middleware.
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool
	// Response status code
	//
	// Optional. Default: 404
	StatusCode int
	// Response status message
	//
	// Optional. Default: Not Found
	StatusMessage string
}

// ConfigDefault is the default config.
func ConfigDefault() Config {
	return Config{
		Next:          nil,
		StatusCode:    defaultStatusCode,
		StatusMessage: defaultStatusMessage,
	}
}

// Helper function to set default values.
func configDefault(config ...Config) Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault()
	}

	// Override default config
	cfg := config[0]

	if cfg.StatusCode == 0 {
		cfg.StatusCode = defaultStatusCode
	}
	if cfg.StatusMessage == "" {
		cfg.StatusMessage = defaultStatusMessage
	}

	return cfg
}
