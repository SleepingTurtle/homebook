# Semantic Versioning Implementation

## Overview

Add semantic versioning (starting at v1.0.0) with:
- Version injected at build time via ldflags
- Displayed in UI footer
- Logged on server startup
- Available via `--version` flag and `/api/version` endpoint

## Implementation Steps

### 1. Create Version Package

**New file: `internal/version/version.go`**

```go
package version

// Set via ldflags at build time
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)
```

This allows build-time injection while having sensible defaults for development.

---

### 2. Update Main Entry Point

**File: `cmd/server/main.go`**

- Import the version package
- Add `--version` flag handling
- Log version on startup

```go
import "homebooks/internal/version"

func main() {
    // Handle --version flag
    if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
        fmt.Printf("HomeBooks %s (built %s, commit %s)\n",
            version.Version, version.BuildTime, version.GitCommit)
        os.Exit(0)
    }

    // ... existing init code ...

    log.Info("server_starting",
        "port", port,
        "version", version.Version,
        "build_time", version.BuildTime)
}
```

---

### 3. Update Handlers to Pass Version to Templates

**File: `internal/handlers/handlers.go`**

Update the `render` helper to automatically include version:

```go
import "homebooks/internal/version"

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
    data["Version"] = version.Version
    // ... existing render code ...
}
```

---

### 4. Display Version in UI Footer

**File: `web/templates/layout.html`**

Update the footer template:

```html
{{define "footer"}}
    </main>
    <footer class="app-footer">
        <span class="version">HomeBooks {{.Version}}</span>
    </footer>
</body>
</html>
{{end}}
```

Add minimal CSS in `web/static/style.css`:

```css
.app-footer {
    text-align: center;
    padding: 1rem;
    color: #9ca3af;
    font-size: 0.75rem;
}
```

---

### 5. Add Version API Endpoint (Optional)

**File: `internal/handlers/handlers.go`**

```go
func (h *Handler) APIVersion(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "version":    version.Version,
        "build_time": version.BuildTime,
        "commit":     version.GitCommit,
    })
}
```

**File: `cmd/server/main.go`** (add route)

```go
mux.HandleFunc("GET /api/version", h.APIVersion)
```

---

### 6. Update Build Configuration

**File: `Makefile`**

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X homebooks/internal/version.Version=$(VERSION) \
           -X homebooks/internal/version.BuildTime=$(BUILD_TIME) \
           -X homebooks/internal/version.GitCommit=$(GIT_COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o homebooks ./cmd/server

release:
	@echo "Building release $(VERSION)..."
	go build -ldflags "$(LDFLAGS) -s -w" -o homebooks ./cmd/server
```

**File: `Dockerfile`**

```dockerfile
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X homebooks/internal/version.Version=${VERSION} \
              -X homebooks/internal/version.BuildTime=${BUILD_TIME} \
              -X homebooks/internal/version.GitCommit=${GIT_COMMIT}" \
    -o homebooks ./cmd/server
```

---

### 7. Create VERSION File

**New file: `VERSION`**

```
1.0.0
```

Update Makefile to read from it:

```makefile
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
```

---

### 8. Git Tagging Workflow

After merge to main:

```bash
# Update VERSION file
echo "1.0.0" > VERSION

# Commit and tag
git add VERSION
git commit -m "Bump version to 1.0.0"
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin main --tags
```

---

## Files Summary

| File | Action |
|------|--------|
| `internal/version/version.go` | **Create** |
| `VERSION` | **Create** |
| `cmd/server/main.go` | Modify (add version flag + logging) |
| `internal/handlers/handlers.go` | Modify (pass version to templates) |
| `web/templates/layout.html` | Modify (add footer) |
| `web/static/style.css` | Modify (footer styles) |
| `Makefile` | Modify (add ldflags) |
| `Dockerfile` | Modify (add build args) |

---

## Usage

```bash
# Development
make dev                    # Shows "dev" version

# Build with auto version from git
make build                  # Uses git describe

# Build specific version
VERSION=1.0.0 make build

# Check version
./homebooks --version       # HomeBooks 1.0.0 (built 2026-01-18T..., commit abc123)

# Docker build with version
docker build --build-arg VERSION=1.0.0 -t homebooks:1.0.0 .
```

---

## Version Bumping Guidelines

Follow semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR** (1.x.x → 2.0.0): Breaking changes, major rewrites
- **MINOR** (1.0.x → 1.1.0): New features, backward compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, small improvements

Examples:
- Adding bank reconciliation module → 1.1.0
- Fixing a parsing bug → 1.1.1
- Complete UI overhaul with breaking API changes → 2.0.0
