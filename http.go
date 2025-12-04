package fiberfx

import (
	"strings"

	"github.com/go-core-fx/fiberfx/prometheus"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"
)

func New(config Config, option Options, logger *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage:   true,
		EnableIPValidation:      true,
		EnableTrustedProxyCheck: len(config.Proxies) > 0,
		ErrorHandler:            option.errorHandler,
		GETOnly:                 option.getOnly,
		ProxyHeader:             config.ProxyHeader,
		TrustedProxies:          config.Proxies,
		Views:                   option.views,
	})
	app.Use(requestid.New())
	app.Use(fiberzap.New(fiberzap.Config{
		Next: func(c *fiber.Ctx) bool {
			p := c.Path()
			// Normalize trailing slash
			for len(p) > 1 && p[len(p)-1] == '/' {
				p = p[:len(p)-1]
			}

			return p == "/health" || p == "/metrics" ||
				strings.HasPrefix(p, "/health/") || strings.HasPrefix(p, "/metrics/")
		},
		SkipBody: func(c *fiber.Ctx) bool {
			return c.Response().StatusCode() < fiber.StatusBadRequest
		},
		Logger: logger,
		Fields: []string{"requestId", "latency", "status", "method", "url", "ip", "ua", "body", "error"},
	}))
	app.Use(recover.New())

	if option.withMetrics {
		prometheus.Register(app)
	}

	return app
}
