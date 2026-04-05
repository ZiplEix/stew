package generator

// file specifier for stew 2.0
const (
	PageFile       = "stew.page.go"
	LayoutFile     = "stew.layout.go"
	ServerFile     = "stew.server.go"
	MiddlewareFile = "stew.middleware.go"
	ErrorFile      = "stew.error.go"
)

// RouteNode represents a URL segment and a Go package
type RouteNode struct {
	Name         string // folder name (ex: "__id__")
	RelativePath string // path from /pages (ex: "users/__id__")
	URLPath      string // final path for the router (ex: "/users/{id}")
	PackageAlias string // unique alias for import (ex: "stew_users_id")
	ImportPath   string // full Go import path

	Methods []string // HTTP methods supported by this route (GET, POST, PUT, DELETE, etc.)

	// presence of special files
	HasPage       bool
	HasServer     bool
	HasLayout     bool
	HasMiddleware bool
	HasError      bool

	Children []*RouteNode
	Parent   *RouteNode
}
