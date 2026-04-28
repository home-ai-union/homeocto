# HomeOcto: Smart Home Agent, Import PicoClaw as Go Module

## Context

HomeClaw (`g:\code\homeclaw`) is currently a **fork** of PicoClaw (`module github.com/sipeed/picoclaw` in go.mod). Smart-home functionality (Xiaomi/Tuya/HomeKit device control, ws-tool WebSocket, intent system) was added directly into PicoClaw's core codebase, creating ongoing merge conflicts when syncing with upstream.

**Goal:** Create a **brand new project** called `homeocto` that imports upstream PicoClaw (`github.com/sipeed/picoclaw`) as a `go mod` dependency. No modifications to the existing homeclaw codebase. HomeOcto wraps PicoClaw's AgentLoop with a thin wrapper pattern.

**Constraints:**
- Cannot modify upstream PicoClaw
- Do NOT modify the current homeclaw codebase (`g:\code\homeclaw`)
- HomeOcto spawns its own custom gateway binary (includes ws-tool WebSocket)
- Frontend: single SPA that combines picoclaw pages (vendored copy) with homeocto smart-home pages (owned code). NOT git submodule — picoclaw frontend is a tightly-coupled SPA with shared sidebar, stores, and routing.

---

## Source: Reference HomeClaw Fork Code

The existing homeclaw fork at `g:\code\homeclaw` serves as the reference for extracting HomeOcto-specific code. Key code locations to copy from:

| Component | Source Path in homeclaw | Nature |
|-----------|------------------------|--------|
| HomeClaw core | `pkg/homeclaw/homeclaw.go` | NewHomeClaw, RegisterTools, SetClients, RunIntent |
| ToolWSHandler | `pkg/homeclaw/tool_ws_handler.go` | WebSocket handler for ws-tool |
| HomeClaw config | `pkg/homeclaw/config/` | homeclaw.json config types |
| Data stores | `pkg/homeclaw/data/` | JSON stores for workflows, devices, spaces |
| Intent system | `pkg/homeclaw/intent/` | Intent classification & routing |
| IoC factory | `pkg/homeclaw/ioc/` | Singleton object factory |
| LLM helpers | `pkg/homeclaw/llm/` | LLM utilities |
| Third-party clients | `pkg/homeclaw/third/` | Xiaomi (miio), Tuya, HomeKit |
| Device tools | `pkg/homeclaw/tool/` | cli_tool, llm_tool, video_tool, common_tool, workflow_tool |
| Workflow engine | `pkg/homeclaw/workflow/` | Workflow execution |
| Web backend API | `web/backend/homeclaw/` | Tuya/Xiaomi/HomeKit/Go2RTC/DeviceOps managers |
| Web frontend | `web/frontend/src/homeclaw/` | ~30 TS/TSX components/stores/hooks |
| Smart-home routes | `web/frontend/src/routes/smart-home/` | 5 route pages |

## Upstream Extension Points (PicoClaw Public API)

These are public APIs HomeOcto can use **without modifying upstream**:

| API | Package | Purpose |
|-----|---------|---------|
| `agent.NewAgentLoop(cfg, msgBus, provider)` | `pkg/agent` | Create agent loop |
| `agentLoop.RegisterTool(tool)` | `pkg/agent` | Register tool on all agents |
| `agentLoop.GetRegistry().GetDefaultAgent().Tools` | `pkg/agent` | Access tool registry |
| `agentLoop.SetChannelManager(cm)` | `pkg/agent` | Inject channel manager |
| `agentLoop.SetMediaStore(s)` | `pkg/agent` | Inject media store |
| `agentLoop.MountHook(NamedHook(...))` | `pkg/agent` | In-process hooks |
| `channels.NewManager(cfg, msgBus, media)` | `pkg/channels` | Create channel manager |
| `channels.ToolCallHandler` interface | `pkg/channels/base.go` | ws-tool handler interface |
| `channels.ToolHandlerSetter` interface | `pkg/channels/base.go` | PicoChannel accepts handler |
| `cm.ChannelByName("pico")` | `pkg/channels` | Get PicoChannel instance |

---

## Implementation Plan

### Step 1: Create New Project `homeocto`

Create a new directory/project alongside `g:\code\homeclaw`:

```
homeocto/
├── go.mod                          # module github.com/<user>/homeocto
│                                   # require github.com/sipeed/picoclaw v<tag>
├── cmd/
│   ├── homeocto-gateway/           # Custom gateway binary (wraps picoclaw)
│   │   └── main.go
│   └── homeocto-launcher/          # Web launcher binary
│       └── main.go
├── pkg/
│   └── homeocto/                   # Copied from homeclaw's pkg/homeclaw/
│       ├── homeocto.go             # Core: NewHomeOcto, RegisterTools, SetClients
│       ├── tool_ws_handler.go      # ToolWSHandler (implements channels.ToolCallHandler)
│       ├── config/                 # homeocto.json config
│       ├── data/                   # JSON stores
│       ├── event/                  # Event center
│       ├── intent/                 # Intent classification
│       ├── ioc/                    # IoC factory
│       ├── llm/                    # LLM helpers
│       ├── third/                  # Xiaomi/Tuya/HomeKit clients
│       │   ├── ioc/
│       │   ├── miio/
│       │   ├── tuya/
│       │   └── homekit/
│       ├── tool/                   # cli_tool, llm_tool, video_tool, etc.
│       └── workflow/               # Workflow engine
├── web/
│   ├── backend/                    # HomeOcto launcher web backend
│   │   ├── main.go                 # Adapted from homeclaw's web/backend/main.go
│   │   ├── api/
│   │   │   ├── router.go          # Merges picoclaw routes + homeocto routes
│   │   │   └── homeocto/          # Tuya/Xiaomi/HomeKit/Go2RTC managers
│   │   └── ...
│   └── frontend/                   # Single SPA (vendored picoclaw + homeocto code)
│       ├── package.json
│       ├── vite.config.ts
│       ├── tsconfig.json
│       └── src/
│           ├── routes/             # Picoclaw pages (chat, config, models, etc.)
│           │   └── smart-home/     # HomeOcto smart-home routes (owned)
│           ├── components/         # Picoclaw UI components + app-sidebar
│           ├── homeocto/           # HomeOcto API/stores/context/components
│           ├── store/              # Shared stores (chat, gateway)
│           ├── api/                # Shared API clients
│           └── i18n/
│               └── locales/
│                   └── homeocto/   # HomeOcto i18n namespace
```

**Import changes:**
- Picoclaw packages (bus, config, tools, channels, logger, media, providers, agents) -> `github.com/sipeed/picoclaw/pkg/...` (real go mod dependency)
- HomeOcto internal packages -> `github.com/<user>/homeocto/pkg/homeocto/...`

**Code to copy from homeclaw fork (reference only, homeclaw NOT modified):**

Backend (Go):
- `g:\code\homeclaw\pkg\homeclaw/**` -> `pkg/homeocto/**` (rename package `homeclaw` -> `homeocto`)
- `g:\code\homeclaw\web\backend\homeclaw/**` -> `web/backend/api/homeocto/**`

Frontend (TypeScript/React):
- `g:\code\homeclaw\web\frontend\src/homeclaw/**` -> `web/frontend/src/homeocto/**` (rename imports, i18n namespace)
- `g:\code\homeclaw\web\frontend\src/routes/smart-home/**` -> `web/frontend/src/routes/smart-home/**`
- `g:\code\homeclaw\web\frontend\src/i18n/locales/homeclaw/**` -> `web/frontend/src/i18n/locales/homeocto/**`

Frontend vendored from picoclaw (copy as starting point, maintain in homeocto):
- `g:\code\homeclaw\web\frontend\src/routes/` (except smart-home/) -> all picoclaw page routes
- `g:\code\homeclaw\web\frontend\src/components/` -> all UI components
- `g:\code\homeclaw\web\frontend\src/store/` -> shared stores
- `g:\code\homeclaw\web\frontend\src/api/` -> shared API clients
- `g:\code\homeclaw\web\frontend\src/features/` -> chat features
- `g:\code\homeclaw\web\frontend\src/i18n/locales/{en,zh}.json` -> picoclaw translations
- `g:\code\homeclaw\web\frontend\package.json, vite.config.ts, tsconfig.json` -> build config

### Step 2: Custom Gateway Binary (Key)

This is the **linchpin**. HomeOcto cannot call upstream's `gateway.Run()` because it's monolithic with no extension points. HomeOcto must replicate the orchestration sequence.

**File: `cmd/homeocto-gateway/main.go`**

The custom gateway follows this sequence:

```
 1. Load config, setup logging                    [same as upstream]
 2. Open listeners, write PID                     [same as upstream]
 3. Create provider                               [same as upstream]
 4. Create MessageBus                             [same as upstream]
 5. agentLoop = agent.NewAgentLoop(cfg, msgBus, provider)   [upstream call]
     │
     ├─── HOMEOCTO POST-INIT ───┐
     │                          │
 6.  ho = homeocto.NewHomeOcto(workspace, cfg, msgBus)
 7.  ho.SetClients()                              [init Xiaomi/Tuya/HomeKit]
 8.  ho.RegisterTools(agentLoop.GetRegistry().GetDefaultAgent().Tools)
     │                          │
     └──────────────────────────┘
     │
 9.  Setup services (cron, heartbeat, media)      [same as upstream]
10.  channelManager = channels.NewManager(...)     [same as upstream]
11.  agentLoop.SetChannelManager(channelManager)
12.  agentLoop.SetMediaStore(mediaStore)
     │
     ├─── WS-TOOL INJECTION ────┐
     │                          │
13.  handler = homeocto.NewToolWSHandler(ho, defaultAgent.Tools, cfg)
14.  picoChannel = channelManager.ChannelByName("pico")
15.  picoChannel.(channels.ToolHandlerSetter).SetToolHandler(handler)
     │                          │
     └──────────────────────────┘
     │
16.  Start channels, health server                [same as upstream]
17.  go agentLoop.Run(ctx)                        [upstream call]
18.  Signal handling loop                          [same as upstream]
```

**What this replicates from upstream `pkg/gateway/gateway.go`:**
- `Run()` function (lines 116-297)
- `setupAndStartServices()` function (lines 338-417)
- Config reload logic (`executeReload`, `handleConfigReload`)

**What HomeOcto adds (steps 6-8, 13-15):**
- HomeOcto initialization after AgentLoop creation
- Tool registration into picoclaw's agent
- ToolWSHandler injection into PicoChannel

**Maintenance trade-off:** HomeOcto must track upstream changes to `gateway.Run()`. Mitigate by:
- Calling upstream helper functions wherever possible (e.g., `providers.CreateProviderFromConfig`, `channels.NewManager`)
- Only replicating orchestration, not internal logic
- Keeping a `// SYNC: picoclaw@v1.x.x` comment noting which version this was derived from

### Step 3: Tool Registration

Straightforward - uses existing upstream API:

```go
// After agent.NewAgentLoop() returns:
ho, err := homeocto.NewHomeOcto(workspace, cfg, msgBus)
ho.SetClients()  // init third-party clients
ho.RegisterTools(agentLoop.GetRegistry().GetDefaultAgent().Tools)
```

**Tools registered** (copied from homeclaw, already implement `tools.Tool` interface):
- Workflow tools: list, get, save, delete, enable, disable
- Video/RTSP tool
- LLM tool (for intent reasoning)
- Common tool
- CLI tool (device control commands)

### Step 4: ws-tool WebSocket (Standalone Handler via PicoChannel Routing)

**Architecture analysis:** ws-tool is a direct WebSocket->tool->response loop that **never touches MessageBus**. Creating a separate ChannelManager would cause message-stealing (two consumers competing for the same `OutboundChan()`), and is unnecessary since ws-tool doesn't need MessageBus at all.

**Recommended approach: Inject ToolWSHandler into PicoChannel** (3 lines, uses upstream public API):

```go
// In HomeOcto's custom gateway, after ChannelManager is created:
handler := homeocto.NewToolWSHandler(ho, toolRegistry, cfg)
ch := channelManager.ChannelByName("pico")
ch.(channels.ToolHandlerSetter).SetToolHandler(handler)
```

**Why this IS "standalone":**
- ToolWSHandler is 100% HomeOcto code
- It never uses MessageBus - direct WebSocket <-> tool execution
- PicoChannel is only the HTTP routing layer (delegates `/pico/ws-tool` to the handler)
- The handler holds `*HomeOcto` + `*tools.ToolRegistry`, no picoclaw internal state
- `ToolCallHandler` and `ToolHandlerSetter` are stable upstream interfaces in `channels/base.go`

**Fallback option (if upstream removes ToolHandlerSetter):**
HomeOcto's gateway creates a separate HTTP server on a dedicated port for `/pico/ws-tool`, and the launcher proxies to it.

**NEW: Standalone WebSocket Server (Recommended)**

Since we cannot modify PicoClaw's code directly, HomeOcto now provides a **standalone WebSocket server** that runs on a separate port. This is the recommended approach when PicoClaw's channel injection is not available.

**Configuration:**

Add `tool_ws_port` to your `homeocto.json`:

```json
{
  "enabled": true,
  "intent_enabled": false,
  "tool_ws_port": 8765
}
```

**Usage in Gateway:**

```go
// In your custom gateway, after creating HomeOcto and registering tools:
ho, err := homeocto.NewHomeOcto(workspace, cfg, msgBus)
if err != nil {
    // handle error
}

// Register tools to the toolRegistry
ho.RegisterTools(toolRegistry)

// Start the standalone WebSocket server (will be skipped if tool_ws_port <= 0)
if err := ho.StartToolWSServer(toolRegistry, cfg); err != nil {
    logger.ErrorCF("gateway", "Failed to start tool WS server", map[string]any{"error": err.Error()})
}

// ... rest of gateway startup ...

// On shutdown:
ho.StopToolWSServer()
```

**Endpoint:**
- The server listens on `:{tool_ws_port}/ws-tool`
- Example: `ws://localhost:8765/ws-tool`
- Uses the same protocol as the PicoClaw-injected handler (token authentication, same message format)

**Benefits:**
- No dependency on PicoClaw's internal interfaces (`ToolHandlerSetter`)
- Can run independently alongside PicoClaw
- Easy to configure via `homeocto.json`
- Clean shutdown support with `StopToolWSServer()`

### Step 5: Intent System (Deferred)

The intent system (`RunIntent`) is **not called** from the current `processMessage`. It's a standalone capability of HomeOcto.

**Future integration path** (when needed):
- Use `agentLoop.MountHook(NamedHook("homeocto-intent", &intentHook{}))` with a `LLMInterceptor`
- In `BeforeLLM()`: classify intent, if handled, publish response via MessageBus

**Recommendation:** Defer intent integration. ws-tool is the primary device control channel.

### Step 6: Web Launcher

Based on homeclaw's `web/backend/main.go` with these changes:

| Component | homeclaw (fork) | homeocto (new) |
|-----------|----------------|-----------------|
| Gateway binary | Spawns `picoclaw gateway -E` | Spawns `homeocto-gateway -E` |
| API routes - picoclaw | Built-in | Import from picoclaw or replicate |
| API routes - homeclaw | In `web/backend/homeclaw/` | In `web/backend/api/homeocto/` |
| Frontend | Single SPA | Unified SPA (vendored picoclaw + homeocto pages) |
| Binary name | `picoclaw-web` | `homeocto-launcher` |

**Key changes:**
- Find `homeocto-gateway` binary instead of `picoclaw`
- API router merges picoclaw's config/session/tool routes with homeocto's device routes

### Step 7: Frontend Integration

**Approach: Single SPA, owned by homeocto, copies picoclaw pages as base**

The current homeclaw fork already has a **unified frontend** — picoclaw pages (chat, config, models) and homeclaw smart-home pages live in the same `web/frontend/` directory, share the same sidebar, and use the same routing. HomeOcto follows the same pattern.

**Directory structure:**
```
web/frontend/
├── package.json               # React 19, TanStack Router, Vite
├── vite.config.ts             # Build config + dev proxy
├── tsconfig.json              # Path alias: @/* → ./src/*
├── src/
│   ├── main.tsx               # Entry point (from picoclaw)
│   ├── routes/
│   │   ├── __root.tsx         # Root layout + auth guard (from picoclaw)
│   │   ├── index.tsx          # Chat page (from picoclaw)
│   │   ├── models.tsx         # Models management (from picoclaw)
│   │   ├── credentials.tsx    # Credentials (from picoclaw)
│   │   ├── config.tsx         # Config page (from picoclaw)
│   │   ├── config/raw.tsx     # Raw config (from picoclaw)
│   │   ├── logs.tsx           # Logs viewer (from picoclaw)
│   │   ├── channels/$name.tsx # Channel pages (from picoclaw)
│   │   ├── agent/             # Agent pages (from picoclaw)
│   │   └── smart-home/        # HomeOcto smart-home pages (OWNED)
│   │       ├── device-control.tsx
│   │       ├── xiaomi.tsx
│   │       ├── tuya.tsx
│   │       ├── apple.tsx
│   │       └── go2rtc.tsx
│   ├── components/
│   │   ├── app-sidebar.tsx    # Navigation (MODIFIED - adds homeocto nav items)
│   │   ├── app-layout.tsx     # Main wrapper (from picoclaw)
│   │   ├── chat/              # Chat components (from picoclaw)
│   │   ├── ui/                # Shadcn/ui (from picoclaw)
│   │   └── ...                # Other picoclaw components
│   ├── homeocto/              # HomeOcto-specific code (OWNED)
│   │   ├── api/
│   │   │   ├── device-control-websocket.ts  # Device WS manager
│   │   │   ├── device-command-executor.ts   # Tool call executor
│   │   │   ├── xiaomi.ts
│   │   │   ├── tuya.ts
│   │   │   ├── apple.ts
│   │   │   ├── go2rtc.ts
│   │   │   └── home-sync.ts
│   │   ├── store/
│   │   │   ├── xiaomi.ts
│   │   │   ├── tuya.ts
│   │   │   ├── apple.ts
│   │   │   ├── go2rtc.ts
│   │   │   └── device-ops.ts
│   │   ├── context/
│   │   │   └── device-control-context.tsx   # Global device context
│   │   └── components/
│   │       ├── smart-home-layout.tsx
│   │       ├── home-section.tsx
│   │       ├── device-list-section.tsx
│   │       ├── device-control-panel.tsx
│   │       ├── xiaomi-page.tsx
│   │       ├── tuya-page.tsx
│   │       ├── apple-page.tsx
│   │       └── go2rtc-page.tsx
│   ├── store/                 # Shared stores (from picoclaw)
│   │   ├── chat.ts
│   │   ├── gateway.ts
│   │   └── tour.ts
│   ├── api/                   # Shared API clients (from picoclaw)
│   │   ├── gateway.ts
│   │   ├── channels.ts
│   │   ├── sessions.ts
│   │   └── models.ts
│   ├── features/              # Feature modules (from picoclaw)
│   │   └── chat/
│   ├── i18n/
│   │   └── locales/
│   │       ├── en.json        # Picoclaw translations (from picoclaw)
│   │       ├── zh.json
│   │       └── homeocto/      # HomeOcto translations (OWNED)
│   │           ├── en.json
│   │           └── zh.json
│   └── hooks/                 # Shared hooks (from picoclaw)
```

**Why NOT git submodule:**
1. Picoclaw's frontend is a **single SPA** — pages are tightly coupled through shared sidebar, layout, stores, and routing. A submodule would create import path conflicts.
2. The TanStack Router file-based routing auto-generates `routeTree.gen.ts` — mixing submodule routes with owned routes is fragile.
3. HomeOcto owns the sidebar navigation (`app-sidebar.tsx`) which references both picoclaw and homeocto routes — splitting them across repos breaks this.

**Recommended approach: Copy + maintain**
1. Copy picoclaw's `web/frontend/` source into `homeocto/web/frontend/` as starting point
2. HomeOcto owns all files; picoclaw files are vendored copies
3. To sync with upstream picoclaw frontend changes: diff upstream `web/frontend/` and apply relevant changes to homeocto's copy
4. Smart-home pages (`src/routes/smart-home/`, `src/homeocto/`) are purely homeocto code

**Key frontend patterns to preserve:**
- **Two WebSocket connections**: Chat WS (`/pico/ws?session_id=chat`) + Device Control WS (`/pico/ws` without session_id). Both connect to the same gateway.
- **Jotai state management**: Picoclaw uses `chatAtom`, `gatewayAtom`; HomeOcto uses `xiaomiAtom`, `tuyaAtom`, etc. No conflict.
- **i18n namespacing**: Picoclaw uses default namespace `t()`; HomeOcto uses `useTranslation("homeclaw")` → `th()`. Namespaces stay separate.
- **DeviceControlProvider**: Wraps entire app in `app-layout.tsx`, provides `useDeviceControl()` hook globally.
- **Sidebar groups**: 6 nav groups — Chat, Model, Channels, Smart Home, Agent, Services. HomeOcto adds/extends Smart Home group items.

**Vite proxy config (unchanged from homeclaw):**
```ts
// vite.config.ts
server: {
  proxy: {
    "/api": "http://localhost:18800",
    "/pico/ws": { target: "ws://localhost:18800", ws: true },
    "/pico/media": "http://localhost:18800",
  }
}
```

**Build command (unchanged):**
```bash
pnpm build:backend  # vite build --outDir ../backend/dist
```
Output embedded into Go binary via `//go:embed`.

**Frontend file copy list from homeclaw (reference only):**
| From homeclaw | To homeocto | Notes |
|---|---|---|
| `web/frontend/src/routes/` (except smart-home/) | `web/frontend/src/routes/` (except smart-home/) | All picoclaw pages |
| `web/frontend/src/components/` (all) | `web/frontend/src/components/` | Picoclaw UI components |
| `web/frontend/src/store/` | `web/frontend/src/store/` | Shared stores |
| `web/frontend/src/api/` | `web/frontend/src/api/` | Shared API clients |
| `web/frontend/src/features/` | `web/frontend/src/features/` | Chat features |
| `web/frontend/src/hooks/` | `web/frontend/src/hooks/` | Shared hooks |
| `web/frontend/src/i18n/locales/en.json, zh.json` | same | Picoclaw translations |
| `web/frontend/src/homeclaw/` | `web/frontend/src/homeocto/` | Rename package |
| `web/frontend/src/routes/smart-home/` | `web/frontend/src/routes/smart-home/` | Owned by homeocto |
| `web/frontend/src/i18n/locales/homeclaw/` | `web/frontend/src/i18n/locales/homeocto/` | Rename namespace |
| `web/frontend/package.json` | `web/frontend/package.json` | Update name |
| `web/frontend/vite.config.ts` | `web/frontend/vite.config.ts` | May need proxy tweaks |
| `web/frontend/tsconfig.json` | `web/frontend/tsconfig.json` | Unchanged |

---

## Critical Files to Create

### New files to create (from scratch in homeocto):
- `go.mod` - new module: `github.com/<user>/homeocto`
- `cmd/homeocto-gateway/main.go` - custom gateway entry point
- `pkg/hcgateway/run.go` - gateway orchestration (replicates upstream `pkg/gateway/gateway.go`)
- `cmd/homeocto-launcher/main.go` - web launcher entry point

### Files to copy from homeclaw backend (reference, homeclaw NOT modified):
- `g:\code\homeclaw\pkg\homeclaw/**` -> `pkg/homeocto/**` (rename `homeclaw` -> `homeocto`)
- `g:\code\homeclaw\web\backend\homeclaw/**` -> `web/backend/api/homeocto/**`

### Files to copy from homeclaw frontend (reference, homeclaw NOT modified):
- `g:\code\homeclaw\web\frontend\src/homeclaw/**` -> `web/frontend/src/homeocto/**`
- `g:\code\homeclaw\web\frontend\src/routes/smart-home/**` -> `web/frontend/src/routes/smart-home/**`
- `g:\code\homeclaw\web\frontend\src/i18n/locales/homeclaw/**` -> `web/frontend/src/i18n/locales/homeocto/**`

### Frontend files to vendor (copy from picoclaw, maintain in homeocto):
- All `web/frontend/` source files except `src/homeclaw/` and `src/routes/smart-home/`

### Files to replicate from upstream picoclaw (reference, upstream NOT modified):
- `pkg/gateway/gateway.go` (upstream) -> `pkg/hcgateway/run.go` (homeocto, adapted)
- `web/backend/main.go` (upstream) -> `cmd/homeocto-launcher/main.go` (homeocto, adapted)

---

## Verification

1. **Build check:** `go build ./cmd/homeocto-gateway/` and `go build ./cmd/homeocto-launcher/` compile without errors
2. **Gateway startup:** `homeocto-gateway -E` starts, logs show "HomeOcto initialised", tool count includes HomeOcto tools
3. **ws-tool WebSocket:** Frontend connects to `/pico/ws-tool`, sends `tool:hc_cli {...}`, receives device control response
4. **Agent chat:** Send message via `/pico/ws` chat, agent has HomeOcto tools available (hc_cli, hc_video, etc.)
5. **Device control:** Execute a device command (e.g., turn on a light) via ws-tool, verify it reaches the third-party API
6. **Frontend:** All pages load - picoclaw pages (chat, config, etc.) and homeocto pages (smart-home/*)
7. **Config reload:** Trigger `/reload`, verify HomeOcto tools are re-registered after reload
8. **Homeclaw unchanged:** Verify `g:\code\homeclaw` has no modifications

---

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Gateway orchestration drift | Medium | Pin picoclaw version. Keep SYNC comments. Review upstream gateway.go on upgrades. |
| Upstream `ToolHandlerSetter` removed | Low | Stable interface. Fallback: HomeOcto handles /ws-tool in its own HTTP handler. |
| Frontend sync with upstream picoclaw | Medium | Vendored copy approach. Diff upstream web/frontend/ on upgrades, apply relevant changes. No submodule import conflicts. |
| Upstream NewAgentLoop signature changes | Low | Major API - unlikely to break without major version bump. |
| Config reload path | Medium | HomeOcto gateway must handle reload same as upstream. Replicate `handleConfigReload` and add HomeOcto re-init. |
