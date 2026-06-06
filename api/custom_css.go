package api

import (
	"github.com/TwiN/gatus/v5/config"
	"github.com/gofiber/fiber/v2"
)

type CustomCSSHandler struct {
	cfg *config.Config
}

func (handler CustomCSSHandler) GetCustomCSS(c *fiber.Ctx) error {
	css := handler.cfg.UI.CustomCSS
	if t := handler.cfg.GetTenantByDomain(c.Hostname()); t != nil && t.UI != nil {
		css = t.UI.CustomCSS
	}
	c.Set("Content-Type", "text/css")
	return c.Status(200).SendString(css)
}
