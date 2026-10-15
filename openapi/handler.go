package openapi

import (
	"sync"

	"github.com/go-core-fx/healthfx"
	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/swaggo/swag"
)

type Handler struct {
	config  Config
	version healthfx.Version
	docs    *swag.Spec

	once sync.Once
}

func NewHandler(config Config, version healthfx.Version, docs *swag.Spec) *Handler {
	return &Handler{
		config:  config,
		version: version,
		docs:    docs,

		once: sync.Once{},
	}
}

func (h *Handler) Register(router fiber.Router) {
	if !h.config.Enabled {
		return
	}

	h.docs.Version = h.version.Version
	h.docs.Host = h.config.PublicHost
	if h.config.PublicPath != "" {
		h.docs.BasePath = h.config.PublicPath
	}

	router.Use("*",
		// Pre-middleware: set host/scheme dynamically
		func(c fiber.Ctx) error {
			h.once.Do(func() {
				if h.docs.Host == "" {
					h.docs.Host = c.Hostname()
				}

				if len(h.docs.Schemes) == 0 {
					h.docs.Schemes = []string{c.Protocol()}
				}
			})

			return c.Next()
		},
		etag.New(etag.Config{Weak: true}),
		swaggo.New(swaggo.Config{Layout: "BaseLayout", URL: "doc.json"}),
	)
}
