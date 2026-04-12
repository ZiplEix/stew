# 🍲 STEW

**Stew** is a high-performance, opinionated **Isomorphic Go Framework** designed for the **Go + Wasm + HTMX** stack. It transforms Go's standard library into a modern fullstack experience with **File-Based Routing**, **Isomorphic Reactivity**, and built-in **Hot Morphing**.

Inspired by SvelteKit but powered by the performance and type-safety of Go, Stew handles the "plumbing" of your application, from Wasm compilation (TinyGo) to server-side rendering, so you can focus on building features.

---

## ✨ Key Features

- 📂 **File-Based Routing**: Your directory structure in `pages/` defines your UI and API routes automatically.
- 🏗️ **Recursive Nesting**: Automatically wrap pages in hierarchical layouts (`@layout.stew`) and protect them with cascading Go middlewares (`stew.middleware.go`).
- ⚡ **Isomorphic Wasm**: Write Go logic in `<goscript client>` blocks. Stew compiles it to Wasm and handles all bindings (`bind:`, `on:`) automatically.
- 🔄 **Hot Morphing (SSE)**: Built-in dev server. Updates the browser via **Idiomorph** without losing state (focus, scroll position, and inputs are preserved).
- 🛡️ **No Dependencies (Runtime)**: Stew compiles your entire project into a single, dependency-free Go binary for production.

---

## 🚀 Quick Start

1. **Install Stew CLI**
   ```bash
   go install github.com/ZiplEix/stew@latest
   ```

2. **Initialize a Project**
   ```bash
   # Initialize in current directory with a module name
   stew init github.com/username/my-app
   ```

3. **Run Development Server**
   ```bash
   stew run dev
   ```

---

## 📂 Router Architecture

Stew uses a strict file-naming convention to avoid conflicts with Go's standard build tools. All special files in `pages/` are prefixed with `@` or `stew.`.

| File | Purpose | Logic Context |
| --- | --- | --- |
| **`@page.stew`** | UI of the route (HTML + Go expressions). | Server & Client (Wasm) |
| **`@layout.stew`** | Wraps all child pages/layouts via `<slot />`. | Server & Client (Wasm) |
| **`stew.server.go`** | Server-only Handlers (GET, POST, API). | Server |
| **`stew.middleware.go`**| Cascading Go middlewares. | Server |

### Example Mapping:
- `pages/@page.stew` → `GET /`
- `pages/api/login/stew.server.go` (func `Post`) → `POST /api/login`
- `pages/users/__id__/@page.stew` → `GET /users/{id}` (Go 1.22+ wildcard support)

---

## 🐹 The `.stew` Syntax

Stew files combine the simplicity of HTML with the power of Go.

```html
<goscript>
    // Server-side logic
    var name = data.URL
    type User struct { Name string }
</goscript>

<goscript client>
    // Isomorphic Wasm logic (TinyGo)
    import github.com/ZiplEix/stew/sdk/wasm"

    func HandleClick() {
        wasm.Alert("Clicked from Go/Wasm!")
    }
</goscript>

<div class="card">
    <h1>Hello, {{ name }}</h1>
    
    {{ if name == "/secret" }}
        <p>This is a secret page!</p>
    {{ end }}
    
    <button on:click="HandleClick">Click Me</button>
</div>
```

---

## 🛠️ CLI Commands

| Command | Description |
| --- | --- |
| `stew init [module]` | Scaffolds a complete Stew-Lang project. |
| `stew compile` | Compiles each `.stew` into Go code and Wasm binaries. |
| `stew generate` | Scans `pages/` and writes the optimized router `stew_router_gen.go`. |
| `stew run dev` | Runs the full dev stack (Watchers + Hot Morphing). |
| `stew run build` | Production build (Compile → Generate → Go Build). |
| `stew clean` | Recursively removes all generated artifacts and tracking files. |

---

## 🧪 Requirements

- **Go 1.22+**: Uses the new `http.ServeMux` features.
- **TinyGo**: Required for Wasm compilation (isomorphic features).
- **HTMX & Idiomorph**: For Hot Morphing (injected automatically).

---

📝 **License**

Distributed under the MIT License. See `LICENSE` for more information.
