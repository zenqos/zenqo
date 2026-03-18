package openapi

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenqos/zenqo/core"
)

// autoErrorResponsesEnabled reports whether automatic error-response injection
// is active for cfg. The field is a pointer so that nil means "use default",
// which is enabled. Only an explicit false pointer disables it.
func autoErrorResponsesEnabled(cfg Config) bool {
	return cfg.AutoErrorResponses == nil || *cfg.AutoErrorResponses
}

// pathParamRe matches chi-style path parameters: {id}, {id:\d+}, {name:[a-z]+}
// Capture group 1 is the parameter name only (strips the regex constraint).
var pathParamRe = regexp.MustCompile(`\{([^}:]+)[^}]*\}`)

func buildSpec(sb *schemaBuilder, cfg Config, routes []core.RouteEntry) *Spec {
	spec := &Spec{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       cfg.Title,
			Version:     cfg.Version,
			Description: cfg.Description,
		},
		Paths: make(map[string]*PathItem),
	}

	// Register global security schemes.
	if len(cfg.Security) > 0 {
		secSchemes := make(map[string]*SecurityScheme, len(cfg.Security))
		var secReqs []map[string][]string
		for _, sd := range cfg.Security {
			secSchemes[sd.name] = sd.scheme
			secReqs = append(secReqs, map[string][]string{sd.name: {}})
		}
		if spec.Components == nil {
			spec.Components = &Components{}
		}
		spec.Components.SecuritySchemes = secSchemes
		spec.Security = secReqs
	}

	for _, re := range routes {
		oaPath := toOpenAPIPath(re.FullPath)
		item, ok := spec.Paths[oaPath]
		if !ok {
			item = &PathItem{}
			spec.Paths[oaPath] = item
		}
		op := buildOperation(sb, cfg, re, oaPath)
		switch re.Method {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		case "PUT":
			item.Put = op
		case "PATCH":
			item.Patch = op
		case "DELETE":
			item.Delete = op
		}
	}
	return spec
}

// toOpenAPIPath converts a chi-style path to an OpenAPI path.
// e.g. /users/{id:\d+} → /users/{id}
func toOpenAPIPath(chiPath string) string {
	result := pathParamRe.ReplaceAllString(chiPath, "{$1}")
	if result != "/" && strings.HasSuffix(result, "/") {
		result = strings.TrimSuffix(result, "/")
	}
	return result
}

func buildOperation(sb *schemaBuilder, cfg Config, re core.RouteEntry, oaPath string) *Operation {
	meta := re.Definition.Meta
	op := &Operation{
		Summary:     meta.Summary,
		Description: meta.Description,
		Tags:        meta.Tags,
		Deprecated:  meta.Deprecated,
		OperationID: buildOperationID(re.Method, oaPath),
		Responses:   make(map[string]*Response),
	}

	// Extract path parameters from the original chi path.
	matches := pathParamRe.FindAllStringSubmatch(re.FullPath, -1)
	for _, m := range matches {
		op.Parameters = append(op.Parameters, &Parameter{
			Name:     m[1],
			In:       "path",
			Required: true,
			Schema:   &Schema{Type: "string"},
		})
	}

	// Request body schema.
	if meta.RequestBody != nil {
		schema := sb.fromValue(meta.RequestBody)
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]*MediaType{
				"application/json": {Schema: schema},
			},
		}
	}

	// Response schemas from explicit .Response() calls.
	for _, rd := range meta.Responses {
		resp := &Response{Description: statusText(rd.Status)}
		if rd.Body != nil {
			schema := sb.fromValue(rd.Body)
			resp.Content = map[string]*MediaType{
				"application/json": {Schema: schema},
			}
		}
		op.Responses[strconv.Itoa(rd.Status)] = resp
	}

	// Infer a default success response when none are explicitly declared.
	if len(op.Responses) == 0 {
		status := defaultStatus(re.Method)
		op.Responses[strconv.Itoa(status)] = &Response{Description: statusText(status)}
	}

	// Per-route security override: empty array means "no auth required".
	if meta.NoSecurity {
		empty := []map[string][]string{}
		op.Security = &empty
	}

	// Automatically inject standard error responses unless opted out.
	if autoErrorResponsesEnabled(cfg) {
		injectAutoErrorResponses(sb, cfg, op, meta)
	}

	return op
}

// injectAutoErrorResponses adds standard HTTP error responses to op based on
// the route's metadata. Existing explicit responses are never overwritten.
//
//   - 500 Internal Server Error — always (framework panic recovery or handler errors)
//   - 400 Bad Request          — routes with a request body (JSON decode failure)
//   - 422 Unprocessable Entity — routes with a request body (validation failure)
//   - 404 Not Found            — routes with path parameters
func injectAutoErrorResponses(sb *schemaBuilder, cfg Config, op *Operation, meta core.RouteDocMeta) {
	errSchema := autoErrorSchema(sb, cfg)

	// addIfMissing inserts code → description unless already documented.
	addIfMissing := func(code int) {
		key := strconv.Itoa(code)
		if _, exists := op.Responses[key]; exists {
			return
		}
		op.Responses[key] = &Response{
			Description: statusText(code),
			Content: map[string]*MediaType{
				"application/json": {Schema: errSchema},
			},
		}
	}

	// 500 is always possible (panic recovery, unhandled handler errors).
	addIfMissing(500)

	// Body-bearing routes can produce 400 (bad JSON) and 422 (validation).
	if meta.RequestBody != nil {
		addIfMissing(400)
		addIfMissing(422)
	}

	// Routes with path parameters can produce 404 (resource not found).
	if len(op.Parameters) > 0 {
		for _, p := range op.Parameters {
			if p.In == "path" {
				addIfMissing(404)
				break
			}
		}
	}
}

// autoErrorSchema returns the appropriate error schema for auto-injected
// responses. ProblemDetail is used when cfg.UseRFC9457 is true; otherwise
// ErrorResponse is used. Both schemas are registered in the shared component
// registry so they appear under #/components/schemas.
func autoErrorSchema(sb *schemaBuilder, cfg Config) *Schema {
	if cfg.UseRFC9457 {
		return sb.fromValue(core.ProblemDetail{})
	}
	return sb.fromValue(core.ErrorResponse{})
}

func buildOperationID(method, oaPath string) string {
	method = strings.ToLower(method)
	parts := strings.Split(oaPath, "/")
	var b strings.Builder
	b.WriteString(method)
	for _, p := range parts {
		if p == "" {
			continue
		}
		// Strip braces from path params: {id} → id
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			p = p[1 : len(p)-1]
		}
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return b.String()
}

func statusText(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return strconv.Itoa(code)
	}
	return text
}

func defaultStatus(method string) int {
	switch method {
	case "POST":
		return 201
	case "DELETE":
		return 204
	default:
		return 200
	}
}
