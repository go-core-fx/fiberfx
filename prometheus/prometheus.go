package prometheus

import (
	fiberprom "github.com/gofiber/contrib/v3/prometheus"
	"github.com/gofiber/fiber/v3"
	promclient "github.com/prometheus/client_golang/prometheus"
)

// Register mounts the Prometheus metrics middleware on the app.
//
// The registry is the one the middleware registers its metrics into and scrapes
// from. A nil registry selects the process-wide default registry.
func Register(app *fiber.App, registry *promclient.Registry) {
	registerer, gatherer := promclient.DefaultRegisterer, promclient.DefaultGatherer
	// The default registry already holds the Go and process collectors, while a
	// caller-provided one starts empty and needs the middleware to register its own.
	disableCollectors := true

	if registry != nil {
		registerer, gatherer = registry, registry
		disableCollectors = false
	}

	app.Use(fiberprom.New(fiberprom.Config{
		Registerer:              registerer,
		Gatherer:                gatherer,
		DisableGoCollector:      disableCollectors,
		DisableProcessCollector: disableCollectors,
		MetricsPath:             "/metrics",
	}))
}
