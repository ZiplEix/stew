# 🍲 STEW 2.0

**Stew** is an opinionated, high-performance **Meta-Framework** and orchestrator built for the **Go + Templ + HTMX** stack. It transforms Go's standard library into a modern fullstack experience with **File-Based Routing**, automatic **Middleware/Layout nesting**, and built-in **Hot Morphing**.

Inspired by the developer experience of SvelteKit but powered by the type-safety of Go, Stew handles the "plumbing" of your application so you can focus on building features.

---

## ✨ Key Features

- 📂 **File-Based Routing**: Your directory structure in ``pages/`` defines your API and UI routes.
- 🏗️ **Recursive Nesting**: Automatically wrap pages in hierarchical layouts and protect them with cascading middlewares.
- 🔄 **Hot Morphing (DX)**: Built-in SSE-based watcher. Updates the browser via Idiomorph without losing state (inputs, scroll position, and modals remain intact).
- 🛡️ **Type-Safe API**: Catch routing errors at compile-time. If a page or handler is missing, Go won't compile.
- 📦 **Orchestration**: Built-in task runner to manage ``templ``, ``air``, and ``stew`` generation in parallel.

---

## 🚀 Quick Start

1. Install Stew

```Bash
go install github.com/ZiplEix/stew@latest
```

2. Initialize a Project

```Bash
# Initialize in current directory with a module name
stew init github.com/username/my-app
```

This command performs a full "simmering" process:
1. Creates ``go.mod`` and ``.stew.yaml``.
2. Scaffolds the ``pages/`` directory with a root layout and page.
3. Generates a pre-configured ``main.go``.
4. Runs ``stew install``, ``templ generate``, and ``stew generate`` to make the project compilable immediately.

3. Run Development Server

```Bash
stew run dev
```

---

## 📂 File-Based Routing Convention

Routes are defined by the folder structure inside the ``pages/`` directory. Each folder is a separate Go package. To avoid conflicts with Go's build tool, all special Stew files use the ``stew.`` prefix.

### Special Files

| File | Purpose | Function Signature |
| --- | --- | --- |
| ``stew.page.templ`` | Defines the UI for the route. | ``templ Page()`` |
| ``stew.server.go`` | Defines API handlers (GET, POST, etc.). | ``func Method(w http.ResponseWriter, r *http.Request)`` |
| ``stew.layout.templ`` | Wraps all child pages/layouts. | ``templ Layout(contents templ.Component)`` |
| ``stew.middleware.go`` | Intercepts requests for the branch. | ``func Middleware(next http.Handler) http.Handler`` |

### Route Mapping Examples

- ``pages/stew.page.templ`` → ``GET /``
- ``pages/about/stew.page.templ`` → ``GET /about``
- ``pages/api/login/stew.server.go`` (with ``func Post``) → ``POST /api/login``
- ``pages/users/_id_/stew.page.templ`` → ``GET /users/{id}`` (Standard Go 1.22+ wildcards)

---

## 🏗️ The Cascade System

Stew 2.0 uses a recursive nesting logic for both UI and Logic. When a route is accessed, Stew crawls from the root pages/ directory down to the target folder.

1. Layout Nesting

    Layouts are emboîtés (nested) like Russian dolls. A page at ``/admin/settings`` will be rendered as:

    ``RootLayout( AdminLayout( SettingsPage() ) )``

2. Middleware Onion

    Middlewares are chained from the root downwards.

    ``RootMiddleware -> AdminMiddleware -> SettingsHandler``

---

## 🛠️ Detailed File Specifications

### ``stew.page.templ``

Defines the UI for the route.

```templ
package pages

templ Page() {
    <h1>Hello, World!</h1>
}
```

### ``stew.server.go``

Expose standard HTTP methods as functions. Stew automatically detects these via AST parsing:

```Go
package hello

import "net/http"

func Get(w http.ResponseWriter, r *http.Request) { /* ... */ }
func Post(w http.ResponseWriter, r *http.Request) { /* ... */ }
```

> Note: If a ``stew.page.templ`` exists in the same folder, it takes priority for GET requests.

### ``stew.middleware.go``

Must return a valid ``http.Handler``.

```Go
package admin

import "net/http"

func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Auth logic here
        next.ServeHTTP(w, r)
    })
}
```

### ``stew.layout.templ``

Must accept ``templ.Component`` to allow nesting.

```Go
package pages

templ Layout(contents templ.Component) {
    <html>
        <body>
            <nav>Navbar</nav>
            @contents
        </body>
    </html>
}
```

## CLI Reference

| Command | Description |
| --- | --- |
| ``stew init [module]`` | Scaffolds a complete Meta-Framework project. |
| ``stew generate`` | Scans ``pages/`` and writes ``stew_router_gen.go.`` |
| ``stew run dev`` | Runs the full dev stack (Router + Templ + Air) with Hot Morphing. |
| ``stew install`` | Installs required binaries (``templ``, ``air``) defined in ``.stew.yaml``. |
| ``stew clean`` | Recursively removes build artifacts (e.g., ``**/*.stew.*_templ.go``). |

## 🧪 Requirements

- Go: 1.22+ (Uses the new ``http.ServeMux`` routing features)
- Templ: Latest
- HTMX & Idiomorph: For Hot Morphing (Injected automatically via ``stew init``)

---

📝 License

Distributed under the MIT License. See ``LICENSE`` for more information.

---

Handcrafted with ❤️ by ZiplEix for the Go community.
