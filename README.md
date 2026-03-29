# 🍲 Stew

**Stew** (stands for **S**imple **T**ask **E**xecution **W**orkflow) is a minimalist, high-performance task runner and orchestrator designed specifically for Go developers. It provides a modern developer experience (DX) similar to `npm` or `bun`, but without the need for a Node.js runtime.

Built with **Go**, **Cobra**, and **Love**, Stew is the perfect companion for the **Go + Templ + HTMX** stack.

---

## ✨ Features
- **🚀 Parallel & Serial Execution**: Run your dev tools (Air, Templ, Tailwind) simultaneously or in sequence.
- **🔄 Integrated Live Reload**: Built-in file watcher and SSE-based browser reloading (No proxy needed!).
- **🎨 Real-time Colored Logs**: Distinguish logs with customizable prefixes, streamed instantly without buffering.
- **🌱 Native .env Support**: Automatically loads environment variables into your sub-processes.
- **📦 Dependency Management**: Define and install required Go binaries (stew install).
- **🧹 Recursive Cleaning**: Easily remove build artifacts (like *_templ.go) project-wide.

---

## 🚀 Installation & Setup

### Install Stew

```bash
go install github.com/ZiplEix/stew@latest
```

### Setup your project

1. Initialize the configuration:

    ```bash
    stew init
    ```

2. Install the Live Reload SDK in your Go app:

    ```bash
    go get github.com/ZiplEix/stew/sdk/live
    ```

### 🛠 Live Reload Integration
Stew uses a lightweight SDK to communicate with your browser.

#### 1. The Middleware

Add the Stew middleware to your router (only in development):

```go
if os.Getenv("STEW_DEV") == "true" {
    handler = live.Middleware(mux)
}
```

#### 2. The Script

Inject the reload script in your base HTML/Templ layout:

```go
// In a Templ file
@templ.Raw(live.InjectScript())
```

#### 3. How it works

When you run ``stew run dev``, Stew starts a file watcher. Upon saving a file, Stew waits for a 1s debounce delay (to allow ``air`` or ``templ`` to finish their work) before signaling the browser to refresh.

The SDK automatically handles reconnections if the server restarts.

## 🛠 Configuration (`.stew.yaml`)

Stew is driven by a simple YAML file:

```yaml
requires:
  - name: templ
    package: github.com/a-h/templ/cmd/templ@latest
  - name: air
    package: github.com/air-verse/air@latest

env_files:
  - .env

colors:
  - '\033[32m' # Green
  - '\033[34m' # Blue

commands:
  dev:
    parallel: true
    scripts:
      - name: templ
        run: templ generate --watch
        watch: true # Tells Stew to trigger reload on file changes
      - name: api
        run: air
```

## 📖 Commands

| Command | Description |
| --- | --- |
| ``stew init`` | Initializes the configuration in a ``.stew.yaml`` file. |
| ``stew run [command]`` | Runs a defined recipe from your config. If no command is provided, it lists all available recipes. - Parallel mode: Best for dev (running compilers and servers). - Serial mode: Best for CI/CD or production builds. |
| ``stew install`` | Installs all Go packages listed in the ``requires`` section using ``go install``. |
| ``stew check`` | Verifies if all binaries used in your scripts are currently available in your $PATH. |
| ``stew clean`` | Removes files and directories defined in the clean section. Use ``**/*suffix`` for recursive deletion (e.g., ``**/*_templ.go``). |
| ``stew exec "command"`` | Runs a one-shot command with ``.env`` variables loaded and Stew's signature colored logging. |
| ``stew env`` | Debugs and lists all environment variables currently loaded from your configured ``.env`` files. |
| ``stew version`` | Displays the current version of Stew. |

## 🤝 Contributing

Contributions are welcome! Feel free to open issues or submit pull requests to help make Stew even better.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

Handcrafted with ❤️ by [ZiplEix](https://github.com/ZiplEix) for the Go community.
