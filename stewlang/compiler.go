package stewlang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var attrRegex = regexp.MustCompile(`([a-zA-Z0-9_-]+)(?:=(?:"([^"]*)"|'([^']*)'|{{([^}]+)}}))?`)

type WasmOptions struct {
	UsesData    bool
	UsesIO      bool
	UsesNav     bool
	UsesStorage bool
}

func buildWasm(name string, nodes []Node, bindings string, clientImports []string, opts WasmOptions, clientVars map[string]bool) (string, error) {
	if _, err := exec.LookPath("tinygo"); err != nil {
		return "", fmt.Errorf("tinygo is not installed. Please install TinyGo to compile WebAssembly bindings: https://tinygo.org/getting-started/")
	}

	var wasmBuf bytes.Buffer
	wasmBuf.WriteString("package main\n\n")

	// Robust scan for reactivity and required packages
	scan := scanReactiveBlocks(nodes, true, clientVars)
	hasAnyReactivity := scan.hasExpressions || scan.hasRanges
	exprCounter := 0
	if !hasAnyReactivity {
		for _, n := range nodes {
			if ifNode, ok := n.(NodeIf); ok && ifNode.BlockID != "" {
				hasAnyReactivity = true
				break
			}
			if eachNode, ok := n.(NodeEach); ok && eachNode.BlockID != "" {
				hasAnyReactivity = true
				break
			}
		}
	}

	importMap := make(map[string]string)
	importMap["github.com/ZiplEix/stew/sdk/wasm"] = ""

	if hasAnyReactivity {
		importMap["strings"] = ""
		if scan.hasExpressions || scan.hasRanges {
			importMap["fmt"] = ""
		}
		if scan.hasExpressions {
			importMap["html"] = ""
		}
		if scan.hasRanges {
			importMap["github.com/ZiplEix/stew/sdk/stew"] = ""
		}
	}

	// Double check if fmt or html are used in bindings even if not caught by scanner
	if strings.Contains(bindings, "fmt.") {
		importMap["fmt"] = ""
	}
	if strings.Contains(bindings, "html.") {
		importMap["html"] = ""
	}

	for _, imp := range clientImports {
		trimmedImp := strings.Trim(imp, "\"")
		if trimmedImp == "stew/data" {
			importMap["github.com/ZiplEix/stew/sdk/wasm/data"] = "stewdata"
			continue
		}
		if trimmedImp == "stew/io" {
			importMap["github.com/ZiplEix/stew/sdk/wasm/io"] = ""
			continue
		}
		if trimmedImp == "stew/nav" {
			importMap["github.com/ZiplEix/stew/sdk/wasm/nav"] = ""
			continue
		}
		if trimmedImp == "stew/storage" {
			importMap["github.com/ZiplEix/stew/sdk/wasm/storage"] = ""
			continue
		}
		if trimmedImp == "stew/state" {
			importMap["github.com/ZiplEix/stew/sdk/wasm/state"] = ""
			continue
		}
		importMap[trimmedImp] = ""
	}

	wasmBuf.WriteString("import (\n")
	for path, alias := range importMap {
		if alias != "" {
			wasmBuf.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, path))
		} else {
			wasmBuf.WriteString(fmt.Sprintf("\t\"%s\"\n", path))
		}
	}
	wasmBuf.WriteString(")\n\n")

	var structDefinitions typesBuilder
	nodes = extractTypes(nodes, &structDefinitions, "client")
	if structDefinitions.Len() > 0 {
		wasmBuf.WriteString(structDefinitions.String() + "\n")
	}

	wasmBuf.WriteString("\nfunc main() {\n")
	var allDeclaredNames []string

	// 1. Initialize Standard PageData (always provided as 'data' variable)
	wasmBuf.WriteString("\n\t// Initialize PageData\n")
	wasmBuf.WriteString("\tdata := wasm.GetPageData()\n")
	wasmBuf.WriteString("\t_ = data\n")
	allDeclaredNames = append(allDeclaredNames, "data")

	// 2. Initialize SDK Helpers
	if opts.UsesIO {
		wasmBuf.WriteString("\tConsole := io.Console\n")
		wasmBuf.WriteString("\tAlert := io.Alert\n")
		wasmBuf.WriteString("\tPrompt := io.Prompt\n")
		wasmBuf.WriteString("\tConfirm := io.Confirm\n")
		allDeclaredNames = append(allDeclaredNames, "Console", "Alert", "Prompt", "Confirm")
	}
	if opts.UsesNav {
		wasmBuf.WriteString("\tnav := nav.Instance\n")
		allDeclaredNames = append(allDeclaredNames, "nav")
	}
	if opts.UsesStorage {
		wasmBuf.WriteString("\tstorage := storage.Instance\n")
		allDeclaredNames = append(allDeclaredNames, "storage")
	}

	// 3. Touch package aliases to avoid "imported and not used"
	for _, alias := range importMap {
		if alias != "" {
			wasmBuf.WriteString(fmt.Sprintf("\t_ = %s.Data\n", alias))
		}
	}

	// 4. Emit User Scripts
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok && gs.Context == "client" {
			lines := strings.Split(gs.Content, "\n")
			cleaned := ""
			for _, line := range lines {
				if !strings.HasPrefix(strings.TrimSpace(line), "import ") {
					wasmBuf.WriteString("\t" + line + "\n")
					cleaned += line + "\n"
				}
			}
			allDeclaredNames = append(allDeclaredNames, extractDeclaredNames(cleaned)...)
		}
	}

	// 5. Automatically "touch" EVERYTHING to avoid "unused" errors
	if len(allDeclaredNames) > 0 {
		wasmBuf.WriteString("\n\t// Automatically touch variables to avoid unused errors\n")
		seen := make(map[string]bool)
		for _, name := range allDeclaredNames {
			if !seen[name] {
				wasmBuf.WriteString(fmt.Sprintf("\t_ = %s\n", name))
				seen[name] = true
			}
		}
	}

	if hasAnyReactivity || exprCounter > 0 {
		wasmBuf.WriteString("\t// Ensure packages are used\n")
		// Prevent "imported and not used" errors
		if _, ok := importMap["fmt"]; ok {
			wasmBuf.WriteString("\t_ = fmt.Sprint\n")
		}
		if _, ok := importMap["html"]; ok {
			wasmBuf.WriteString("\t_ = html.EscapeString\n")
		}
		if _, ok := importMap["strings"]; ok {
			wasmBuf.WriteString("\t_ = strings.Contains\n")
		}
		if _, ok := importMap["github.com/ZiplEix/stew/sdk/stew"]; ok {
			wasmBuf.WriteString("\t_ = stew.Range(0, 0)\n")
		}
		wasmBuf.WriteString("\tanonBuilder := strings.Builder{}\n")
		wasmBuf.WriteString("\t_ = anonBuilder\n")
	}

	// Determine if we have actual code or reactive blocks
	hasClientCode := false
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok && gs.Context == "client" {
			hasClientCode = true
			break
		}
	}

	// If there's no actual client code and no bindings, just skip
	if !hasClientCode && bindings == "" && !hasAnyReactivity {
		return "", nil
	}

	wasmBuf.WriteString("\n\t// DOM Bindings generated by Stew\n")
	wasmBuf.WriteString(bindings)

	// Universal Reactivity: IDs for expressions
	var regBuf bytes.Buffer

	// First pass: Emit nodes and collect registrations
	emitWasmNodes(&wasmBuf, &regBuf, nodes, nil, false, &exprCounter, clientVars)

	// Append registrations to main
	wasmBuf.Write(regBuf.Bytes())

	if hasAnyReactivity || exprCounter > 0 {
		wasmBuf.WriteString("\n\t// Start the reactivity loop\n")
		wasmBuf.WriteString("\twasm.StartReactivityLoop()\n")
	}
	wasmBuf.WriteString("}\n\n")

	tmpDir := filepath.Join(os.TempDir(), "stew_wasm")
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, 0755)
	}
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("stew_main_wasm_%s.go", name))

	wasmCode := wasmBuf.String()
	// fmt.Println("----- GENERATED WASM -----")
	// fmt.Println(wasmCode)
	// fmt.Println("--------------------------")

	err := os.WriteFile(tmpFile, []byte(wasmCode), 0644)
	if err != nil {
		return "", err
	}

	outWasm := filepath.Join(".", "static", "wasm", name+".wasm")
	os.MkdirAll(filepath.Dir(outWasm), 0755)

	cmd := exec.Command("tinygo", "build", "-o", outWasm, "-target", "wasm", tmpFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  TinyGo build warning: %v.\nTinyGo Stderr: %s\n", err, stderr.String())
		return "", nil // Non-fatal, we still track
	}

	return outWasm, nil
}

func Compile(name string, pkgName string, moduleBase string, relFilePath string, input string) (string, []string, error) {
	lexer := NewLexer(input, moduleBase, relFilePath)
	tokens := lexer.Lex()

	parser := NewParser(tokens)
	nodes, err := parser.Parse()
	if err != nil {
		return "", nil, err
	}

	// First pass: extract imports from <goscript>
	var userImports []string
	var stewImports []string
	var clientImports []string
	wasmOpts := extractImports(nodes, &userImports, &stewImports, &clientImports, moduleBase, relFilePath)

	isComponent := false
	if regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`).MatchString(name) && !strings.HasSuffix(name, "Page") && name != "Layout" {
		isComponent = true
	}

	// Isomorphic processing: detect bindings & inject IDs
	var modifiedNodes []Node
	bindCounter := 0
	blockCounter := 0
	hasClientCode := false
	var clientBindings bytes.Buffer

	for i := 0; i < len(nodes); i++ {
		n := nodes[i]
		// Assign BlockIDs to If/Each blocks for reactivity
		if ifNode, ok := n.(NodeIf); ok {
			ifNode.BlockID = fmt.Sprintf("stew-block-%s-%d", strings.ToLower(name), blockCounter)
			blockCounter++
			n = ifNode
		} else if eachNode, ok := n.(NodeEach); ok {
			eachNode.BlockID = fmt.Sprintf("stew-block-%s-%d", strings.ToLower(name), blockCounter)
			blockCounter++
			n = eachNode
		}

		if bindNode, ok := n.(NodeBind); ok {
			// Orphan bindings (without a client script) will be ignored later by buildWasm check
			if i > 0 && len(modifiedNodes) > 0 {
				if prevHTML, isHTML := modifiedNodes[len(modifiedNodes)-1].(NodeHTML); isHTML {
					// Detect if we already injected an ID for this element (multiple bindings)
					var bindID string
					idMatch := regexp.MustCompile(`id="(stew-bind-[^"]+)"`).FindStringSubmatch(prevHTML.Content)
					if len(idMatch) > 1 {
						bindID = idMatch[1]
					} else {
						bindID = fmt.Sprintf("stew-bind-%s-%d", strings.ToLower(name), bindCounter)
						bindCounter++
						// Defense check: ensure we are not injecting into a closed tag or text node
						trimmed := strings.TrimSpace(prevHTML.Content)
						if !strings.HasSuffix(trimmed, ">") && strings.Contains(trimmed, "<") {
							// Naïve injection at the end of the HTML string pre-parsing (assumes tag is open)
							prevHTML.Content += fmt.Sprintf(` id="%s" `, bindID)
						}
						modifiedNodes[len(modifiedNodes)-1] = prevHTML
					}

					// Process binding variable: strip braces if they were passed (though Lexer handles it usually)
					cleanVar := strings.TrimSpace(bindNode.BindVar)
					isLiteral := strings.HasPrefix(cleanVar, "\"") && strings.HasSuffix(cleanVar, "\"")

					if bindNode.IsEvent {
						// Smarter event handling heuristic:
						// 1. If it's a function literal: func(...) { ... }, pass it directly.
						// 2. If it's a simple identifier (no parens), assume it's a func name and pass it directly.
						// 3. Otherwise (a call or expression), wrap it in a func() { ... }.
						expr := cleanVar
						if isLiteral {
							expr = strings.Trim(expr, "\"")
						}

						shouldWrap := true
						trimmedExpr := strings.TrimSpace(expr)
						if strings.HasPrefix(trimmedExpr, "func") {
							shouldWrap = false
						} else if !strings.Contains(trimmedExpr, "(") {
							// Identifier or variable reference
							shouldWrap = false
						}

						if shouldWrap {
							prefix := ""
							if strings.Contains(expr, "this") {
								prefix = fmt.Sprintf("this := wasm.GetElement(\"%s\"); _ = this; ", bindID)
							}
							clientBindings.WriteString(fmt.Sprintf("\twasm.OnEvent(\"%s\", \"%s\", func() { %s%s })\n", bindID, bindNode.BindType, prefix, expr))
						} else {
							clientBindings.WriteString(fmt.Sprintf("\twasm.OnEvent(\"%s\", \"%s\", %s)\n", bindID, bindNode.BindType, expr))
						}
					} else {
						if bindNode.BindType == "value" {
							// Literals for values don't make sense for BindInput (&pointer)
							if !isLiteral {
								if strings.Contains(cleanVar, ".Get()") {
									// Signal-based value binding (experimental/manual via on:input usually)
									// But for now, we just emit a warning or ignore to avoid pointer-to-Get()
								} else {
									clientBindings.WriteString(fmt.Sprintf("\twasm.BindInput(\"%s\", &%s)\n", bindID, cleanVar))
								}
							}
						} else {
							if isLiteral {
								// Literals: ignore or just set once?
							} else {
								if strings.Contains(cleanVar, ".Get()") {
									// IMPORTANT: Reactive expression (Signal). 
									// Use BindBlock instead of BindContent to avoid "&userName.Get()" error.
									clientBindings.WriteString(fmt.Sprintf("\twasm.BindBlock(\"%s\", func() string { return fmt.Sprint(%s) })\n", bindID, cleanVar))
								} else {
									clientBindings.WriteString(fmt.Sprintf("\twasm.BindContent(\"%s\", &%s)\n", bindID, cleanVar))
								}
							}
						}
					}
				}
			}
		} else if gs, ok := n.(NodeGoScript); ok {
			if gs.Context == "client" {
				hasClientCode = true
			}
			modifiedNodes = append(modifiedNodes, n)
		} else {
			modifiedNodes = append(modifiedNodes, n)
		}
	}
	nodes = modifiedNodes

	// Check if there is actual client script code to determine Wasm build
	// Reactive bindings alone don't trigger build if there's no script to define vars
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok && gs.Context == "client" {
			hasClientCode = true
			break
		}
	}

	// Generate unique Wasm name to avoid clashes between nested pages
	wasmName := strings.ReplaceAll(strings.TrimSuffix(relFilePath, ".stew"), "/", "_")
	wasmName = strings.ReplaceAll(wasmName, "@", "")
	wasmName = strings.ToLower(wasmName)

	// Determine client names for reactivity
	clientVars := make(map[string]bool)
	clientVars["wasm"] = true
	clientVars["stew"] = true
	// "data" is provided in Wasm, but shouldn't trigger reactivity on its own
	// to avoid server-side functions like getLinkClass(data.URL) failing in Wasm.
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok && gs.Context == "client" {
			names := extractDeclaredNames(gs.Content)
			for _, name := range names {
				clientVars[name] = true
			}
		}
	}

	// Build WebAssembly if client code is present
	var generatedArtifacts []string
	if hasClientCode {
		wasmPath, err := buildWasm(wasmName, nodes, clientBindings.String(), clientImports, wasmOpts, clientVars)
		if err != nil {
			fmt.Printf("⚠️  TinyGo build warning: %v.\n", err)
		} else if wasmPath != "" {
			generatedArtifacts = append(generatedArtifacts, wasmPath)
		}
	}

	var pkgLevel typesBuilder
	nodes = extractTypes(nodes, &pkgLevel, "server")

	pageName := name
	if !isComponent && pageName != "Layout" {
		if len(pageName) > 0 {
			pageName = strings.ToUpper(pageName[0:1]) + pageName[1:]
		}
		if !strings.HasSuffix(pageName, "Page") {
			pageName += "Page"
		}
	}

	var bodyBuf bytes.Buffer
	if isComponent {
		bodyBuf.WriteString(fmt.Sprintf("func %s(w io.Writer, data stew.PageData, props %sProps, slot func()) {\n\n", pageName, pageName))
	} else if pageName == "Layout" {
		bodyBuf.WriteString("func Layout(w io.Writer, data stew.PageData, slot func()) {\n\n")
	} else {
		bodyBuf.WriteString(fmt.Sprintf("func %s(w io.Writer, data stew.PageData) {\n\n", pageName))
	}

	// Server emit
	var serverNodes []Node
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok {
			if gs.Context == "" || gs.Context == "server" {
				serverNodes = append(serverNodes, gs)
			}
		} else {
			serverNodes = append(serverNodes, n)
		}
	}
	bodyExprCounter := 0
	if err := emitNodes(&bodyBuf, serverNodes, lexer.ValidComponents, &bodyExprCounter, false, clientVars); err != nil {
		return "", nil, err
	}

	// Inject Wasm Bootstrap if client code
	if hasClientCode {
		bodyBuf.WriteString("\tdataJSON, _ := json.Marshal(data)\n")
		bodyBuf.WriteString("\tw.Write([]byte(`<script type=\"application/json\" id=\"stew-pagedata\">`))\n")
		bodyBuf.WriteString("\tw.Write(dataJSON)\n")
		bodyBuf.WriteString("\tw.Write([]byte(`</script>`))\n")

		wasmPath := fmt.Sprintf("/static/wasm/%s.wasm", wasmName)
		bootstrap := fmt.Sprintf(`<script src="/static/wasm/wasm_exec.js"></script><script>const go = new Go(); WebAssembly.instantiateStreaming(fetch("%s"), go.importObject).then((result) => { go.run(result.instance); });</script>`, wasmPath)
		bodyBuf.WriteString(fmt.Sprintf("\tw.Write([]byte(`%s`))\n", bootstrap))
	}

	bodyBuf.WriteString("}\n")
	bodyStr := bodyBuf.String()

	// Final Import block generation
	var buf bytes.Buffer
	buf.WriteString("// Code generated by Stew-Lang. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	// Collect unique imports and verify usage
	importPaths := make(map[string]bool)
	importPaths["\"io\""] = true
	importPaths["\"github.com/ZiplEix/stew/sdk/stew\""] = true
	if hasClientCode {
		importPaths["\"encoding/json\""] = true
	}

	// codeOnlyStr strips the raw HTML string content (inside backtick literals)
	codeOnlyStr := stripBacktickContent(bodyStr)

	// Dynamic detection of fmt and html through codeOnlyStr
	if strings.Contains(codeOnlyStr, "fmt.") {
		importPaths["\"fmt\""] = true
	}
	if strings.Contains(codeOnlyStr, "html.EscapeString") || strings.Contains(codeOnlyStr, "html.") {
		importPaths["\"html\""] = true
	}
	if strings.Contains(codeOnlyStr, "state.") || strings.Contains(pkgLevel.String(), "state.") {
		importPaths["\"github.com/ZiplEix/stew/sdk/wasm/state\""] = true
	}

	// Check user imports usage
	for _, imp := range userImports {
		cleanImp := strings.Trim(imp, "\"")
		alias := ""
		parts := strings.Fields(cleanImp)
		if len(parts) == 2 {
			alias = parts[0]
		} else {
			alias = path.Base(strings.Trim(cleanImp, "\"'"))
		}
		if strings.Contains(codeOnlyStr, alias+".") || strings.Contains(pkgLevel.String(), alias+".") {
			importPaths[imp] = true
		}
	}

	// Check stew component imports usage
	for _, imp := range stewImports {
		cleanImp := strings.Trim(imp, "\"")
		alias := path.Base(cleanImp)
		if strings.Contains(codeOnlyStr, alias+".") || strings.Contains(pkgLevel.String(), alias+".") {
			importPaths[imp] = true
		}
	}

	buf.WriteString("import (\n")
	for imp := range importPaths {
		buf.WriteString("\t" + imp + "\n")
	}
	buf.WriteString(")\n\n")

	if pkgLevel.Len() > 0 {
		buf.WriteString(pkgLevel.String() + "\n")
	}

	if isComponent {
		matched, _ := regexp.MatchString(fmt.Sprintf(`type\s+%sProps\s+struct`, pageName), pkgLevel.String())
		if !matched {
			buf.WriteString(fmt.Sprintf("type %sProps struct {}\n\n", pageName))
		}
	}

	buf.WriteString(bodyStr)

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Output unformatted for debugging if formatting fails
		return buf.String(), generatedArtifacts, fmt.Errorf("error formatting generated code: %v\nCode:\n%s", err, buf.String())
	}

	return string(formatted), generatedArtifacts, nil
}

func extractImports(nodes []Node, userImports *[]string, stewImports *[]string, clientImports *[]string, moduleBase string, relFilePath string) WasmOptions {
	var opts WasmOptions
	for _, n := range nodes {
		if gs, ok := n.(NodeGoScript); ok {
			lines := strings.Split(gs.Content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "import ") {
					importStr := strings.TrimSpace(strings.TrimPrefix(line, "import "))
					importStr = strings.Trim(importStr, "\"'")

					if importStr == "stew/data" {
						opts.UsesData = true
						*clientImports = append(*clientImports, "\""+importStr+"\"")
						continue
					}
					if importStr == "stew/io" {
						opts.UsesIO = true
						*clientImports = append(*clientImports, "\""+importStr+"\"")
						continue
					}
					if importStr == "stew/nav" {
						opts.UsesNav = true
						*clientImports = append(*clientImports, "\""+importStr+"\"")
						continue
					}
					if importStr == "stew/storage" {
						opts.UsesStorage = true
						*clientImports = append(*clientImports, "\""+importStr+"\"")
						continue
					}
					if importStr == "stew/state" {
						if gs.Context == "client" {
							*clientImports = append(*clientImports, "\""+importStr+"\"")
						} else {
							*userImports = append(*userImports, "\"github.com/ZiplEix/stew/sdk/wasm/state\"")
						}
						continue
					}

					if gs.Context == "client" {
						*clientImports = append(*clientImports, "\""+importStr+"\"")
						continue
					}

					if strings.HasSuffix(importStr, ".stew") {
						dir := importStr
						lastSlash := strings.LastIndex(dir, "/")
						if lastSlash != -1 {
							dir = dir[:lastSlash]
						} else {
							dir = "."
						}

						currentDir := "."
						lastFileSlash := strings.LastIndex(relFilePath, "/")
						if lastFileSlash != -1 {
							currentDir = relFilePath[:lastFileSlash]
						}

						if dir == "." {
							continue
						}

						fullPkgPath := path.Join(moduleBase, currentDir, dir)
						*stewImports = append(*stewImports, "\""+fullPkgPath+"\"")
						continue
					}
					*userImports = append(*userImports, "\""+importStr+"\"")
				}
			}
		} else if b, ok := n.(NodeIf); ok {
			res1 := extractImports(b.Body, userImports, stewImports, clientImports, moduleBase, relFilePath)
			res2 := extractImports(b.ElseBody, userImports, stewImports, clientImports, moduleBase, relFilePath)
			opts.merge(res1)
			opts.merge(res2)
		} else if b, ok := n.(NodeEach); ok {
			res := extractImports(b.Body, userImports, stewImports, clientImports, moduleBase, relFilePath)
			opts.merge(res)
		} else if b, ok := n.(NodeComponent); ok {
			res := extractImports(b.Body, userImports, stewImports, clientImports, moduleBase, relFilePath)
			opts.merge(res)
		}
	}
	return opts
}

func (o *WasmOptions) merge(other WasmOptions) {
	if other.UsesData {
		o.UsesData = true
	}
	if other.UsesIO {
		o.UsesIO = true
	}
	if other.UsesNav {
		o.UsesNav = true
	}
	if other.UsesStorage {
		o.UsesStorage = true
	}
}

type typesBuilder struct {
	strings.Builder
}

func extractTypes(nodes []Node, pkgLevel *typesBuilder, targetContext string) []Node {
	var out []Node
	for _, n := range nodes {
		switch node := n.(type) {
		case NodeGoScript:
			isMatch := node.Context == targetContext
			// Special case: empty context is treated as "server" if target is "server"
			if targetContext == "server" && node.Context == "" {
				isMatch = true
			}

			if !isMatch {
				out = append(out, node)
				continue
			}

			lines := strings.Split(node.Content, "\n")
			var remainingLines []string

			inTypeBlock := false
			inImportBlock := false
			braceCount := 0
			parenCount := 0
			var typeBlock strings.Builder

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if !inTypeBlock && !inImportBlock {
					if strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, " struct") {
						inTypeBlock = true
						typeBlock.WriteString(line + "\n")
						braceCount += strings.Count(line, "{") - strings.Count(line, "}")
						if braceCount <= 0 && strings.Contains(line, "{") {
							pkgLevel.WriteString(typeBlock.String())
							typeBlock.Reset()
							inTypeBlock = false
						}
					} else if strings.HasPrefix(trimmed, "import (") {
						inImportBlock = true
						// We don't append to pkgLevel here as extractImports already handled it
						parenCount += strings.Count(line, "(") - strings.Count(line, ")")
						if parenCount <= 0 {
							inImportBlock = false
						}
					} else if strings.HasPrefix(trimmed, "import ") {
						// Strip single line import
						continue
					} else {
						remainingLines = append(remainingLines, line)
					}
				} else if inTypeBlock {
					typeBlock.WriteString(line + "\n")
					braceCount += strings.Count(line, "{") - strings.Count(line, "}")
					if braceCount <= 0 {
						pkgLevel.WriteString(typeBlock.String())
						typeBlock.Reset()
						inTypeBlock = false
					}
				} else if inImportBlock {
					parenCount += strings.Count(line, "(") - strings.Count(line, ")")
					if parenCount <= 0 {
						inImportBlock = false
					}
				}
			}
			node.Content = strings.Join(remainingLines, "\n")
			out = append(out, node)

		case NodeIf:
			node.Body = extractTypes(node.Body, pkgLevel, targetContext)
			node.ElseBody = extractTypes(node.ElseBody, pkgLevel, targetContext)
			out = append(out, node)
		case NodeEach:
			node.Body = extractTypes(node.Body, pkgLevel, targetContext)
			out = append(out, node)
		case NodeComponent:
			node.Body = extractTypes(node.Body, pkgLevel, targetContext)
			out = append(out, node)
		default:
			out = append(out, node)
		}
	}
	return out
}

func emitNodes(buf *bytes.Buffer, nodes []Node, validComps map[string]string, exprCounter *int, inReactiveBlock bool, clientVars map[string]bool) error {
	inTag := false
	for _, n := range nodes {
		switch node := n.(type) {
		case NodeHTML:
			emitHTML(buf, node.Content)
			lastOpen := strings.LastIndex(node.Content, "<")
			lastClose := strings.LastIndex(node.Content, ">")
			if lastOpen > lastClose {
				inTag = true
			} else if lastClose > lastOpen {
				inTag = false
			}
		case NodeExpression:
			// If we are already inside a reactive parent, we don't need a separate ID/span
			// because the parent re-renders its entire body using the client's loop.
			isReactive := inReactiveBlock || isReactiveExpression(node.Content, clientVars)

			if isReactive && !inReactiveBlock && !inTag {
				id := fmt.Sprintf("stew-expr-%d", *exprCounter)
				*exprCounter++
				buf.WriteString(fmt.Sprintf("\tw.Write([]byte(`<span id=\"%s\">`))\n", id))
				emitExpression(buf, node.Content)
				buf.WriteString("\tw.Write([]byte(`</span>`))\n")
			} else {
				emitExpression(buf, node.Content)
			}
		case NodeGoScript:
			// Emit both default and explicit server scripts
			if node.Context == "" || node.Context == "server" {
				buf.WriteString(node.Content + "\n")
			}
		case NodeIf:
			bodyExprCounter := *exprCounter
			isStructural := !inTag
			if isStructural && node.BlockID != "" {
				buf.WriteString(fmt.Sprintf("\tw.Write([]byte(`<div id=\"%s\" style=\"display: contents;\">`))\n", node.BlockID))
			}
			// SSR Logic
			buf.WriteString(fmt.Sprintf("\tif %s {\n", node.Condition))
			emitNodes(buf, node.Body, validComps, exprCounter, node.BlockID != "", clientVars)
			if len(node.ElseBody) > 0 {
				buf.WriteString("\t} else {\n")
				emitNodes(buf, node.ElseBody, validComps, exprCounter, node.BlockID != "", clientVars)
			}
			buf.WriteString("\t}\n")
			if isStructural && node.BlockID != "" {
				buf.WriteString("\tw.Write([]byte(`</div>`))\n")
			}
			_ = bodyExprCounter
		case NodeEach:
			bodyExprCounter := *exprCounter
			isStructural := !inTag
			parts := strings.Split(node.Iterator, " as ")
			if len(parts) == 2 {
				slice := strings.TrimSpace(parts[0])
				vars := strings.Split(parts[1], ",")
				varName := "item"
				idxName := "_"
				if len(vars) == 1 {
					varName = strings.TrimSpace(vars[0])
				} else if len(vars) == 2 {
					varName = strings.TrimSpace(vars[0])
					idxName = strings.TrimSpace(vars[1])
				}
				if isStructural && node.BlockID != "" {
					buf.WriteString(fmt.Sprintf("\tw.Write([]byte(`<div id=\"%s\" style=\"display: contents;\">`))\n", node.BlockID))
				}
				buf.WriteString(fmt.Sprintf("\tfor %s, %s := range %s {\n", idxName, varName, resolveIterator(slice)))
				emitNodes(buf, node.Body, validComps, exprCounter, node.BlockID != "", clientVars)
				buf.WriteString("\t}\n")
				if isStructural && node.BlockID != "" {
					buf.WriteString("\tw.Write([]byte(`</div>`))\n")
				}
			}
			_ = bodyExprCounter
		case NodeComponent:
			// parse symbols
			props := parseAttributes(node.TagContent)
			var structValues []string
			for k, v := range props {
				fieldName := k
				if len(fieldName) > 0 {
					fieldName = strings.ToUpper(fieldName[0:1]) + fieldName[1:]
				}
				structValues = append(structValues, fmt.Sprintf("%s: %s", fieldName, v))
			}
			structInit := fmt.Sprintf("%sProps{%s}", node.TagName, strings.Join(structValues, ", "))
			alias := validComps[node.TagName]

			if node.SelfClosing || len(node.Body) == 0 {
				buf.WriteString(fmt.Sprintf("\t%s%s(w, data, %s%s, nil)\n", alias, node.TagName, alias, structInit))
			} else {
				buf.WriteString(fmt.Sprintf("\t%s%s(w, data, %s%s, func() {\n", alias, node.TagName, alias, structInit))
				if err := emitNodes(buf, node.Body, validComps, exprCounter, inReactiveBlock, clientVars); err != nil {
					return err
				}
				buf.WriteString("\t})\n")
			}

		case NodeSlot:
			buf.WriteString("\tif slot != nil {\n\t\tslot()\n\t}\n")
		}
	}
	return nil
}

// stripBacktickContent removes the content of Go raw string literals (between backticks)
// from generated code, so that import-usage detection doesn't false-positive on
// HTML display text that happens to contain "fmt." or "html." etc.
func stripBacktickContent(s string) string {
	// Temporarily mask the backtick escape sequence used in emitHTML
	// logic: w.Write([]byte(`...` + "`" + `...`))
	s = strings.ReplaceAll(s, "` + \"`\" + `", "__B_ESC__")

	var result strings.Builder
	inBacktick := false
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			inBacktick = !inBacktick
			result.WriteByte('`')
			continue
		}
		if !inBacktick {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

func emitHTML(buf *bytes.Buffer, content string) {
	if len(content) == 0 {
		return
	}
	// escape ` and write
	escaped := strings.ReplaceAll(content, "`", "`+\"`\"+`")
	buf.WriteString(fmt.Sprintf("\tw.Write([]byte(`%s`))\n", escaped))
}

func emitExpression(buf *bytes.Buffer, expr string) {
	expr = strings.TrimSpace(expr)

	// handle raw()
	if strings.HasPrefix(expr, "raw(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		buf.WriteString(fmt.Sprintf("\tw.Write([]byte(fmt.Sprint(%s)))\n", inner))
		return
	}

	buf.WriteString(fmt.Sprintf("\tw.Write([]byte(html.EscapeString(fmt.Sprint(%s))))\n", expr))
}

func emitWasmNodes(buf *bytes.Buffer, regBuf *bytes.Buffer, nodes []Node, validComps map[string]string, skipReg bool, exprCounter *int, clientVars map[string]bool) {
	inTag := false
	for _, n := range nodes {
		switch node := n.(type) {
		case NodeHTML:
			if skipReg {
				emitWasmHTML(buf, node.Content, true)
			}
			lastOpen := strings.LastIndex(node.Content, "<")
			lastClose := strings.LastIndex(node.Content, ">")
			if lastOpen > lastClose {
				inTag = true
			} else if lastClose > lastOpen {
				inTag = false
			}
		case NodeExpression:
			isReactive := skipReg || isReactiveExpression(node.Content, clientVars)
			// If we're at the top level (not inside a structural block or tag attribute), register a reactive span
			if isReactive && !skipReg && !inTag {
				id := fmt.Sprintf("stew-expr-%d", *exprCounter)
				*exprCounter++
				// Register reactivity in regBuf (main)
				regBuf.WriteString(fmt.Sprintf("\twasm.BindBlock(\"%s\", func() string {\n", id))
				regBuf.WriteString("\t\tvar buf strings.Builder\n")
				emitWasmExpression(regBuf, node.Content, true)
				regBuf.WriteString("\t\treturn buf.String()\n")
				regBuf.WriteString("\t})\n")
			} else if skipReg {
				// We are inside a callback, render expression logic
				emitWasmExpression(buf, node.Content, true)
			}
		case NodeGoScript:
			// Scripts are handled in buildWasm's pre-pass to be at the top of main()
			continue
		case NodeIf:
			if node.BlockID != "" && !skipReg && !inTag {
				// Register structure reactivity in main
				regBuf.WriteString(fmt.Sprintf("\twasm.BindBlock(\"%s\", func() string {\n", node.BlockID))
				regBuf.WriteString("\t\tvar buf strings.Builder\n")
				// Inside callback, render body but SKIP further registrations
				emitWasmNodes(regBuf, regBuf, []Node{node}, validComps, true, exprCounter, clientVars)
				regBuf.WriteString("\t\treturn buf.String()\n")
				regBuf.WriteString("\t})\n")
			}

			if skipReg {
				// Emit Go IF logic within a parent callback
				buf.WriteString(fmt.Sprintf("\tif %s {\n", node.Condition))
				emitWasmNodes(buf, regBuf, node.Body, validComps, true, exprCounter, clientVars)
				if len(node.ElseBody) > 0 {
					buf.WriteString("\t} else {\n")
					emitWasmNodes(buf, regBuf, node.ElseBody, validComps, true, exprCounter, clientVars)
				}
				buf.WriteString("\t}\n")
			}
		case NodeEach:
			if node.BlockID != "" && !skipReg && !inTag {
				// Register structure reactivity in main
				regBuf.WriteString(fmt.Sprintf("\twasm.BindBlock(\"%s\", func() string {\n", node.BlockID))
				regBuf.WriteString("\t\tvar buf strings.Builder\n")
				emitWasmNodes(regBuf, regBuf, []Node{node}, validComps, true, exprCounter, clientVars)
				regBuf.WriteString("\t\treturn buf.String()\n")
				regBuf.WriteString("\t})\n")
			}

			if skipReg {
				parts := strings.Split(node.Iterator, " as ")
				if len(parts) == 2 {
					slice := strings.TrimSpace(parts[0])
					vars := strings.Split(parts[1], ",")
					varName := "item"
					idxName := "_"
					if len(vars) == 1 {
						varName = strings.TrimSpace(vars[0])
					} else if len(vars) == 2 {
						varName = strings.TrimSpace(vars[0])
						idxName = strings.TrimSpace(vars[1])
					}
					buf.WriteString(fmt.Sprintf("\tfor %s, %s := range %s {\n", idxName, varName, resolveIterator(slice)))
					emitWasmNodes(buf, regBuf, node.Body, validComps, true, exprCounter, clientVars)
					buf.WriteString("\t}\n")
				}
			}
		case NodeComponent:
			if skipReg {
				// Inside a reactive block, we just render the component's output
				// Since we don't know the component's props logic here easily, we just emit a placeholder or the SSR call
				// Actually, components should probably not be nested inside reactive blocks for now if they are not client-ready
				// For now, let's just skip to avoid complex recursion issues in Wasm closures
			}
		}
	}
}

type ScanResult struct {
	hasExpressions bool
	hasRanges      bool
}

func scanReactiveBlocks(nodes []Node, onlyReactive bool, clientVars map[string]bool) ScanResult {
	res := ScanResult{}
	for _, n := range nodes {
		switch node := n.(type) {
		case NodeIf:
			if !onlyReactive || node.BlockID != "" {
				r := scanReactiveBlocks(node.Body, false, clientVars)
				res.hasExpressions = res.hasExpressions || r.hasExpressions
				res.hasRanges = res.hasRanges || r.hasRanges
				r2 := scanReactiveBlocks(node.ElseBody, false, clientVars)
				res.hasExpressions = res.hasExpressions || r2.hasExpressions
				res.hasRanges = res.hasRanges || r2.hasRanges
			}
		case NodeEach:
			if !onlyReactive || node.BlockID != "" {
				res.hasRanges = true
				r := scanReactiveBlocks(node.Body, false, clientVars)
				res.hasExpressions = res.hasExpressions || r.hasExpressions
				res.hasRanges = res.hasRanges || r.hasRanges
			}
		case NodeExpression:
			isReactive := !onlyReactive || isReactiveExpression(node.Content, clientVars)
			if isReactive {
				res.hasExpressions = true
			}
		case NodeComponent:
			r := scanReactiveBlocks(node.Body, false, clientVars)
			res.hasExpressions = res.hasExpressions || r.hasExpressions
			res.hasRanges = res.hasRanges || r.hasRanges
		case NodeBind:
			if !node.IsEvent && strings.Contains(node.BindVar, ".Get()") {
				res.hasExpressions = true
			}
		}
	}
	return res
}

func getUsedIdentifiers(expr string) map[string]bool {
	identifiers := make(map[string]bool)
	// Try parsing it as a full expression.
	// We might need to wrap it if it's just a variable name? parseExpr usually handles that.
	e, err := parser.ParseExpr(expr)
	if err != nil {
		// Fallback: simple split if it's not a valid Go expression (unlikely here)
		return identifiers
	}
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			identifiers[id.Name] = true
		}
		return true
	})
	return identifiers
}

func isReactiveExpression(expr string, clientVars map[string]bool) bool {
	if strings.Contains(expr, ".Get()") {
		return true // New Signal-based reactivity
	}
	used := getUsedIdentifiers(expr)
	for id := range used {
		if clientVars[id] {
			return true
		}
	}
	return false
}

func parseAttributes(tagContent string) map[string]string {
	// e.g. <Button title="Hello" count={{ data.Count }} disabled>
	tagContent = strings.TrimSpace(tagContent)
	tagContent = strings.TrimPrefix(tagContent, "<")
	tagContent = strings.TrimSuffix(tagContent, "/>")
	tagContent = strings.TrimSuffix(tagContent, ">")

	// strip tag name
	idx := strings.IndexAny(tagContent, " \t\n\r")
	if idx == -1 {
		return nil
	}

	attrStr := tagContent[idx+1:]
	matches := attrRegex.FindAllStringSubmatch(attrStr, -1)

	props := make(map[string]string)

	for _, m := range matches {
		if len(m) == 0 {
			continue
		}
		key := m[1]
		if key == "" || strings.HasPrefix(key, "bind:") {
			continue
		}

		val := "true" // boolean prop like `disabled`

		if m[2] != "" {
			val = "\"" + m[2] + "\"" // string like "hello"
		} else if m[3] != "" {
			val = "\"" + m[3] + "\"" // string like 'hello'
		} else if m[4] != "" {
			val = m[4] // expression like data.Count
		}

		props[key] = val
	}

	return props
}

func emitWasmHTML(buf *bytes.Buffer, content string, isCallback bool) {
	if len(content) == 0 {
		return
	}
	escaped := strings.ReplaceAll(content, "`", "`+\"`\"+`")
	if isCallback {
		buf.WriteString(fmt.Sprintf("\t\tbuf.WriteString(`%s`)\n", escaped))
	}
}

func emitWasmExpression(buf *bytes.Buffer, expr string, isCallback bool) {
	expr = strings.TrimSpace(expr)
	if !isCallback {
		return // Expressions in main() don't emit HTML, they are already in SSR
	}

	if strings.HasPrefix(expr, "raw(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		buf.WriteString(fmt.Sprintf("\t\tbuf.WriteString(fmt.Sprint(%s))\n", inner))
		return
	}
	buf.WriteString(fmt.Sprintf("\t\tbuf.WriteString(html.EscapeString(fmt.Sprint(%s)))\n", expr))
}


func resolveIterator(it string) string {
	if strings.Contains(it, "..") {
		parts := strings.Split(it, "..")
		if len(parts) == 2 {
			start := strings.TrimSpace(parts[0])
			end := strings.TrimSpace(parts[1])
			return fmt.Sprintf("stew.Range(%s, %s)", start, end)
		}
	}
	return it
}

func extractDeclaredNames(content string) []string {
	// Strip imports before parsing, otherwise parser fails inside a func
	lines := strings.Split(content, "\n")
	cleaned := ""
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "import ") {
			cleaned += line + "\n"
		}
	}

	fset := token.NewFileSet()
	// Wrap in a function to allow statements and local declarations
	// We use a unique function name to find it easily
	wrapped := "package main\nfunc _stew_dummy_() {\n" + cleaned + "\n}"
	f, err := parser.ParseFile(fset, "", wrapped, 0)
	if err != nil {
		return nil
	}

	var names []string
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "_stew_dummy_" {
			for _, stmt := range fn.Body.List {
				switch s := stmt.(type) {
				case *ast.AssignStmt:
					if s.Tok == token.DEFINE {
						for _, lhs := range s.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
								names = append(names, id.Name)
							}
						}
					}
				case *ast.DeclStmt:
					if gd, ok := s.Decl.(*ast.GenDecl); ok {
						if gd.Tok == token.VAR || gd.Tok == token.CONST {
							for _, spec := range gd.Specs {
								if vs, ok := spec.(*ast.ValueSpec); ok {
									for _, id := range vs.Names {
										if id.Name != "_" {
											names = append(names, id.Name)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return names
}
