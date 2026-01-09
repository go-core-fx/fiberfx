package health

import (
	"github.com/go-core-fx/fiberfx/handler"
	"github.com/go-core-fx/healthfx"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

type Handler struct {
	healthSvc *healthfx.Service
}

func NewHandler(
	healthSvc *healthfx.Service,
) handler.Handler {
	return &Handler{
		healthSvc: healthSvc,
	}
}

func (h *Handler) Register(router fiber.Router) {
	router = router.Group("/health")
	router.Use(func(c *fiber.Ctx) error {
		if h.healthSvc == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return c.Next()
	})

	router.Get("", h.getLiveness)
	router.Get("live", h.getLiveness)
	router.Get("ready", h.getReadiness)
	router.Get("startup", h.getStartup)
}

//	@Summary		Liveness probe
//	@Description	Checks if service is running (liveness probe)
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	HealthResponse	"Service is alive"
//	@Failure		503	{object}	HealthResponse	"Service is not alive"
//	@Router			/health [get]
//	@Router			/health/live [get]
//
// Liveness probe.
func (h *Handler) getLiveness(c *fiber.Ctx) error {
	return h.writeProbe(c, h.healthSvc.CheckLiveness(c.Context()))
}

//	@Summary		Readiness probe
//	@Description	Checks if service is ready to serve traffic (readiness probe)
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	HealthResponse	"Service is ready"
//	@Failure		503	{object}	HealthResponse	"Service is not ready"
//	@Router			/health/ready [get]
//
// Readiness probe.
func (h *Handler) getReadiness(c *fiber.Ctx) error {
	return h.writeProbe(c, h.healthSvc.CheckReadiness(c.Context()))
}

//	@Summary		Startup probe
//	@Description	Checks if service has completed initialization (startup probe)
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	HealthResponse	"Service has completed initialization"
//	@Failure		503	{object}	HealthResponse	"Service has not completed initialization"
//	@Router			/health/startup [get]
//
// Startup probe.
func (h *Handler) getStartup(c *fiber.Ctx) error {
	return h.writeProbe(c, h.healthSvc.CheckStartup(c.Context()))
}

func (h *Handler) writeProbe(c *fiber.Ctx, r healthfx.CheckResult) error {
	status := fiber.StatusOK
	if r.Status() == healthfx.StatusFail {
		status = fiber.StatusServiceUnavailable
	}
	return c.Status(status).JSON(h.makeResponse(r))
}

func (h *Handler) makeResponse(result healthfx.CheckResult) Response {
	version := h.healthSvc.Version()

	return Response{
		Status:    Status(result.Status()),
		Version:   version.Version,
		ReleaseID: version.ReleaseID,
		Checks: lo.MapValues(
			result.Checks,
			func(value healthfx.CheckDetail, _ string) Check {
				return Check{
					Description:   value.Description,
					ObservedUnit:  value.ObservedUnit,
					ObservedValue: value.ObservedValue,
					Status:        Status(value.Status),
				}
			},
		),
	}
}
