package core

import "strings"

// RouteEntry describes a registered route with its fully resolved path.
// Used by the OpenAPI spec generator to enumerate all routes.
type RouteEntry struct {
	Method     string
	FullPath   string
	Definition *RouteDefinition
}

// CollectRoutes returns all registered routes with their fully resolved paths.
// Call after registering all modules and controllers but before app.Start().
// This is used by openapi.Mount to build the OpenAPI spec.
func (a *App) CollectRoutes() []RouteEntry {
	var entries []RouteEntry
	prefix := a.prefix

	add := func(c Controller, pathPrefix string) {
		rp, ok := c.(RouteProvider)
		if !ok {
			return
		}
		base := joinPath(pathPrefix, c.BasePath())
		for _, rd := range rp.Routes() {
			entries = append(entries, RouteEntry{
				Method:     rd.Method,
				FullPath:   joinPath(base, rd.Path),
				Definition: rd,
			})
		}
	}

	for _, m := range a.modules {
		for _, c := range m.Controllers() {
			add(c, prefix)
		}
	}
	for _, c := range a.controllers {
		add(c, prefix)
	}
	// Top-level routes registered directly on the app (app.GET/POST/etc.)
	for _, rd := range a.root.routes {
		entries = append(entries, RouteEntry{
			Method:     rd.Method,
			FullPath:   joinPath(prefix, rd.Path),
			Definition: rd,
		})
	}
	return entries
}

// joinPath concatenates two URL path segments, normalizing double slashes.
func joinPath(base, part string) string {
	if base == "/" {
		base = ""
	}
	if part == "" {
		part = "/"
	}
	p := base + part
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	if p == "" {
		return "/"
	}
	return p
}
