package routes

import (
	"io/fs"

	"github.com/emailservice/internal/handler"
	fiber "github.com/gofiber/fiber/v2"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Email Service API - Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: "/swagger.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ]
      });
    };
  </script>
</body>
</html>
`

func Email(app *fiber.App, h *handler.EmailHandler) {
	app.Post("/emails/send", h.SendEmail)
	app.Post("/send-email", h.SendEmailHandler)

	app.Get("/swagger.json", func(c *fiber.Ctx) error {
		b, _ := fs.ReadFile(specFS, "spec/swagger.json")
		c.Set("Content-Type", "application/json")
		return c.Send(b)
	})
	app.Get("/swagger.yaml", func(c *fiber.Ctx) error {
		b, _ := fs.ReadFile(specFS, "spec/swagger.yaml")
		c.Set("Content-Type", "application/yaml")
		return c.Send(b)
	})
	app.Get("/swagger", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(swaggerUIHTML)
	})
}
