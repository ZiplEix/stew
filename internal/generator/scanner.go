package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Scanner struct {
	RootDir    string
	ModulePath string
}

func NewScanner(rootDir, modulePath string) *Scanner {
	return &Scanner{
		RootDir:    rootDir,
		ModulePath: modulePath,
	}
}

// Scan builds the route tree from the pages directory
func (s *Scanner) Scan() (*RouteNode, error) {
	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, err
	}

	rootNode := &RouteNode{
		Name:         "root",
		RelativePath: "",
		URLPath:      "/",
		PackageAlias: "stew_pages_root",
		ImportPath:   filepath.Join(s.ModulePath, s.RootDir),
	}

	nodes := make(map[string]*RouteNode)
	nodes[""] = rootNode

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("⚠️  Warning: could not access path %s: %v\n", path, err)
			return nil
		}

		if !d.IsDir() || path == absRoot {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		parentRel := filepath.Dir(rel)

		if parentRel == "." {
			parentRel = ""
		}

		node := s.createNode(rel, d.Name())

		s.fillFilesInfo(node, path)

		if parent, ok := nodes[parentRel]; ok {
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		}

		nodes[rel] = node
		return nil
	})

	s.fillFilesInfo(rootNode, absRoot)

	return rootNode, nil
}

// createNode calculate URLPath and PackageAlias
func (s *Scanner) createNode(relPath, name string) *RouteNode {
	urlPath := "/" + convertDynamicSegments(relPath)

	cleanedRel := strings.ReplaceAll(relPath, "__", "")
	alias := "stew_" + strings.ReplaceAll(strings.ReplaceAll(cleanedRel, "/", "_"), "-", "_")

	return &RouteNode{
		Name:         name,
		RelativePath: relPath,
		URLPath:      urlPath,
		PackageAlias: alias,
		ImportPath:   filepath.ToSlash(filepath.Join(s.ModulePath, s.RootDir, relPath)),
	}
}

// fillFilesInfo checks for the presence of special files
func (s *Scanner) fillFilesInfo(node *RouteNode, absPath string) {
	node.HasPage = fileExists(filepath.Join(absPath, PageFile))
	node.HasServer = fileExists(filepath.Join(absPath, ServerFile))
	node.HasLayout = fileExists(filepath.Join(absPath, LayoutFile))
	node.HasMiddleware = fileExists(filepath.Join(absPath, MiddlewareFile))
	node.HasError = fileExists(filepath.Join(absPath, ErrorFile))

	serverPath := filepath.Join(absPath, ServerFile)
	if fileExists(serverPath) {
		node.HasServer = true
		node.Methods = s.inspectServerMethods(serverPath)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// inspectServerMethods analyse the AST of stew.server.go and returns the HTTP methods
func (s *Scanner) inspectServerMethods(filePath string) []string {
	methods := []string{}
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		fmt.Printf("⚠️  Error parsing %s: %v\n", filePath, err)
		return methods
	}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		name := fn.Name.Name
		if isHTTPMethod(strings.ToUpper(name)) {
			methods = append(methods, name)
		}
	}
	return methods
}

func isHTTPMethod(name string) bool {
	valid := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true, "OPTIONS": true,
	}
	return valid[name]
}

// convertDynamicSegments transforms __slug__ folder names into {slug} URL parameters.
// Examples:
//
//	"users/__id__"       → "users/{id}"
//	"files/__path...__"  → "files/{path...}"
func convertDynamicSegments(relPath string) string {
	segments := strings.Split(relPath, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, "__") && strings.HasSuffix(seg, "__") && len(seg) > 4 {
			inner := seg[2 : len(seg)-2]
			segments[i] = "{" + inner + "}"
		}
	}
	return strings.Join(segments, "/")
}
