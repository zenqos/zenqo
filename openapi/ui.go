package openapi

import "github.com/zenqos/zenqo/core"

type openAPICtrl struct {
	core.BaseController
}

// swaggerUIHTML is a minimal Swagger UI page that loads from jsDelivr CDN.
// Format args:
//   - %s = API title (HTML-escaped)
//   - %s = spec URL (JSON-encoded string literal, including surrounding quotes)
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>%s — API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: %s,
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout',
      tryItOutEnabled: %s,
    });
  </script>
</body>
</html>`
