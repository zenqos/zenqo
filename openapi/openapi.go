// Package openapi generates an OpenAPI 3.1 spec from Zenqo route definitions
// and serves it alongside a Swagger UI explorer.
//
// Usage:
//
//	app := core.NewApp()
//	app.UseController(userController)
//
//	openapi.Mount(app, openapi.Config{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	})
//
//	app.Start(":3000")
//	// → GET /openapi.json  — machine-readable spec
//	// → GET /docs          — Swagger UI
package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zenqos/zenqo/core"
	enc "github.com/zenqos/zenqo/internal/encoding"
)

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

// Config holds metadata for the generated OpenAPI spec.
type Config struct {
	// Title is the API name shown in Swagger UI (required).
	Title string
	// Version is the API version string (default: "1.0.0").
	Version string
	// Description is an optional Markdown description shown in Swagger UI.
	Description string
	// SpecPath is the URL path that serves the JSON spec (default: "/openapi.json").
	SpecPath string
	// DocsPath is the URL path that serves the Swagger UI (default: "/docs").
	DocsPath string
}

// Mount registers the OpenAPI JSON spec and Swagger UI endpoints on the app.
// Call this after registering all controllers and modules, but before app.Start().
//
// Endpoints registered:
//   - GET {SpecPath}  → application/json OpenAPI 3.1 spec
//   - GET {DocsPath}  → Swagger UI HTML
func Mount(app *core.App, cfg Config) {
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.SpecPath == "" {
		cfg.SpecPath = "/openapi.json"
	}
	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}

	// Collect routes NOW — before adding the openAPI controller itself.
	routes := app.CollectRoutes()
	sb := &schemaBuilder{
		schemas:  make(map[string]*Schema),
		building: make(map[string]bool),
	}
	spec := buildSpec(sb, cfg, routes)
	if len(sb.schemas) > 0 {
		spec.Components = &Components{Schemas: sb.schemas}
	}
	specJSON, _ := json.MarshalIndent(spec, "", "  ")

	specPath := cfg.SpecPath
	docsPath := cfg.DocsPath
	title := cfg.Title

	// The Swagger UI must fetch the spec using the full URL path (including any
	// global prefix set via app.SetGlobalPrefix), so the browser request resolves
	// correctly regardless of which path the docs page is served from.
	fullSpecURL := app.Prefix() + specPath

	ctrl := &openAPICtrl{}
	ctrl.SetBasePath("/")
	ctrl.Handle("GET", specPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(specJSON) //nolint:errcheck
	})
	ctrl.Handle("GET", docsPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, swaggerUIHTML, title, fullSpecURL) //nolint:errcheck
	})
	app.UseController(ctrl)
}

// ─────────────────────────────────────────────────────────────────────────────
// OpenAPI 3.1 spec structs
// ─────────────────────────────────────────────────────────────────────────────

// Spec is the root OpenAPI 3.1 document object.
type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// PathItem groups all operations for a single URL path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation.
type Operation struct {
	Summary     string               `json:"summary,omitempty"`
	Description string               `json:"description,omitempty"`
	OperationID string               `json:"operationId,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Deprecated  bool                 `json:"deprecated,omitempty"`
	Parameters  []*Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*Response `json:"responses"`
}

// Parameter describes a path, query, header, or cookie parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes the request body for a POST/PUT/PATCH operation.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required"`
	Content     map[string]*MediaType `json:"content"`
}

// MediaType wraps a schema for a specific content type.
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Response describes a single HTTP response.
type Response struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// Components holds reusable schemas referenced by $ref.
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// Schema is a JSON Schema (subset) used to describe request/response bodies.
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Spec builder
// ─────────────────────────────────────────────────────────────────────────────

// pathParamRe matches chi-style path parameters: {id}, {id:\\d+}, {name:[a-z]+}
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

	for _, re := range routes {
		oaPath := toOpenAPIPath(re.FullPath)
		item, ok := spec.Paths[oaPath]
		if !ok {
			item = &PathItem{}
			spec.Paths[oaPath] = item
		}
		op := buildOperation(sb, re, oaPath)
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
// e.g. /users/{id:\\d+} → /users/{id}
func toOpenAPIPath(chiPath string) string {
	result := pathParamRe.ReplaceAllString(chiPath, "{$1}")
	if result != "/" && strings.HasSuffix(result, "/") {
		result = strings.TrimSuffix(result, "/")
	}
	return result
}

func buildOperation(sb *schemaBuilder, re core.RouteEntry, oaPath string) *Operation {
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

	return op
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

// ─────────────────────────────────────────────────────────────────────────────
// Schema builder
// ─────────────────────────────────────────────────────────────────────────────

var timeType = reflect.TypeOf(time.Time{})

type schemaBuilder struct {
	schemas  map[string]*Schema
	building map[string]bool // recursion guard for self-referential structs
}

// fromValue infers a Schema from a Go value (typically a zero-value struct).
func (sb *schemaBuilder) fromValue(v any) *Schema {
	if v == nil {
		return &Schema{Type: "object"}
	}
	return sb.fromType(reflect.TypeOf(v))
}

// fromType converts a reflect.Type to an OpenAPI Schema.
func (sb *schemaBuilder) fromType(t reflect.Type) *Schema {
	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Slices.
	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.Uint8 { // []byte → base64 string
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: sb.fromType(t.Elem())}
	}

	// time.Time → date-time string.
	if t == timeType {
		return &Schema{Type: "string", Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s := &Schema{Type: "integer"}
		if t.Kind() == reflect.Int32 {
			s.Format = "int32"
		} else if t.Kind() == reflect.Int64 {
			s.Format = "int64"
		}
		return s
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: sb.fromType(t.Elem())}
	case reflect.Struct:
		return sb.fromStruct(t)
	default:
		return &Schema{Type: "object"}
	}
}

// fromStruct converts a struct type to an OpenAPI Schema.
// Named structs are added to components/schemas and referenced via $ref.
func (sb *schemaBuilder) fromStruct(t reflect.Type) *Schema {
	name := t.Name()

	if name != "" {
		if _, exists := sb.schemas[name]; exists {
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		if sb.building[name] {
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		sb.building[name] = true
		defer func() { sb.building[name] = false }()
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fieldName, _ := enc.ResolveFieldTag(f)
		if fieldName == "-" {
			continue
		}

		fieldSchema := sb.fromType(f.Type)
		applyValidateTags(f, f.Type, fieldSchema, &schema.Required, fieldName)
		schema.Properties[fieldName] = fieldSchema
	}

	if len(schema.Properties) == 0 {
		schema.Properties = nil
	}

	if name != "" {
		sb.schemas[name] = schema
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	return schema
}

// applyValidateTags reads validate:"..." struct tags and sets OpenAPI constraints
// on the field schema, and appends to the parent object's required list.
func applyValidateTags(f reflect.StructField, ft reflect.Type, schema *Schema, required *[]string, fieldName string) {
	tag := f.Tag.Get("validate")
	if tag == "" {
		return
	}
	// Dereference pointer for kind checks.
	for ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	k := ft.Kind()

	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "required":
			*required = append(*required, fieldName)
		case strings.HasPrefix(rule, "min="):
			n, err := strconv.Atoi(rule[4:])
			if err != nil {
				break
			}
			if k == reflect.String {
				schema.MinLength = &n
			} else if isNumericKind(k) {
				f64 := float64(n)
				schema.Minimum = &f64
			}
		case strings.HasPrefix(rule, "max="):
			n, err := strconv.Atoi(rule[4:])
			if err != nil {
				break
			}
			if k == reflect.String {
				schema.MaxLength = &n
			} else if isNumericKind(k) {
				f64 := float64(n)
				schema.Maximum = &f64
			}
		case rule == "email":
			schema.Format = "email"
		case rule == "url":
			schema.Format = "uri"
		case rule == "uuid":
			schema.Format = "uuid"
		case strings.HasPrefix(rule, "oneof="):
			for _, v := range strings.Split(rule[6:], "|") {
				schema.Enum = append(schema.Enum, v)
			}
		}
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal controller (serves spec + Swagger UI)
// ─────────────────────────────────────────────────────────────────────────────

type openAPICtrl struct {
	core.BaseController
}

// ─────────────────────────────────────────────────────────────────────────────
// Swagger UI HTML template
// ─────────────────────────────────────────────────────────────────────────────

// swaggerUIHTML is a minimal Swagger UI page that loads from jsDelivr CDN.
// Format args: %s = API title, %s = spec URL path.
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
      url: '%s',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout',
      tryItOutEnabled: true,
    });
  </script>
</body>
</html>`
