package prometheus

import (
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App) {
	promhandler := fiberprometheus.NewWithDefaultRegistry("")
	promhandler.RegisterAt(app, "/metrics")
	app.Use(promhandler.Middleware)
}
