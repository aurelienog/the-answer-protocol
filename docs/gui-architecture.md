# TAP Desktop & Web GUI Architecture

This document describes the architecture, directory structure, assets organization, and runtime data flow of the TAP graphical client (`cmd/desktop`, `internal/desktop`, and `frontend/gui`).

---

## 1. Overview

The GUI client is built as a lightweight **Go HTTP/SSE bridge** paired with a **zero-dependency modular Web UI** (vanilla HTML5, CSS3, ES6 JavaScript).

- On desktop (`make run-client-gui`), `cmd/desktop` launches an embedded local HTTP server (`127.0.0.1:0`) and opens a dedicated browser window in `--app` mode (Chrome / Chromium / Brave / Firefox fallback).
- Closing the window or pressing `Ctrl+C` cleanly sends `QUIT` to the TCP server and terminates the process.
- The GUI communicates with the TAP game server strictly via standard TCP line-based protocol messages, keeping it decoupled from server internals and interchangeable across RFC 42TAP-compliant servers.

---

## 2. Go Bridge Layer

| Path | Responsibility |
|---|---|
| `cmd/desktop/main.go` | Parses CLI flags (`-host`, `-port`, `-world`), starts the local HTTP server, serves static GUI assets, exposes `/api/world` and `/api/config`, and manages the native app-mode browser lifecycle and `SIGINT`/`SIGTERM` cleanup. |
| `internal/desktop/app.go` | Manages the TCP socket to the TAP server with serialized writes (`writeMu`), performs the `CONNECT` handshake, streams asynchronous server lines (`OK`, `ERR`, `EVT`) over Server-Sent Events (`/api/stream`), accepts commands via `/api/command`, and handles session restoration (`/api/session`) across browser refreshes. |

---

## 3. Frontend Directory Structure

```text
frontend/gui/
├── index.html                        # Semantic shell (Login screen, Top HUD, World stage, Docks, 10 Modals)
│
├── styles/
│   ├── main.css                      # Entry point importing all modular stylesheets
│   │
│   ├── base/
│   │   ├── variables.css             # Color palette (--gold-*, --sapphire-*, HP/MP/Stamina gradients)
│   │   └── reset.css                 # Box-sizing, typography, custom scrollbars, shared image helpers
│   │
│   ├── frames/                       # Pure-CSS ornate frame systems (::before / ::after, zero DOM spans)
│   │   ├── window-frame.css          # Main sapphire & gold window frame (.gold-panel)
│   │   ├── inner-panel-frame.css     # Split inner panels (.inner-gold-box & .flush-portrait)
│   │   ├── slot-input-frame.css      # Cards, slots, inputs, previews, and small square buttons
│   │   ├── hp-bar-frame.css          # Ornate HP/stat bar borders
│   │   └── right-menu-frame.css      # Vertical arched frame for the right action sidebar
│   │
│   ├── components/
│   │   ├── buttons.css               # Textured RPG buttons & tabs (Base, Blue, Gold, Red cross-fade)
│   │   ├── cards.css                 # Selectable list cards (.select-card) & checkmark states
│   │   └── dividers.css              # Sapphire dividers (.sapphire-divider) & badges (.badge)
│   │
│   ├── layout/
│   │   ├── login.css                 # Login screen & responsive form layout
│   │   ├── top-hud.css               # Top bar (Hero avatar/HP, TAP crest, Room/Server counters, WHO/STATUS/QUIT)
│   │   ├── world-stage.css           # Fullscreen room background & location title banner
│   │   ├── left-npcs.css             # Left quick-action NPC dock (TALK / ATTACK cards)
│   │   ├── right-sidebar.css         # Right action menu (8 protocol command buttons)
│   │   ├── compass.css               # Center compass rose with dynamic N/S/E/W exit pins
│   │   ├── bottom-dock.css           # Bottom dock (Chat/Log tabs, Compass, Location/Party tables)
│   │   └── responsive.css            # Responsive breakpoints (2K/ultrawide, compact split-screen, mobile)
│   │
│   └── modals/
│       ├── modal-base.css            # Shared modal geometry, headers, tabs, split columns & footers
│       ├── modal-look.css            # LOOK modal (5 tabs: Description, Exits, Items, NPCs, Players)
│       ├── modal-inventory.css       # INVENTORY 4x4 grid & item inspector
│       ├── modal-status.css          # STATUS portrait & combat/world stats stack
│       ├── modal-quit.css            # QUIT & FLEE confirmation dialogs with top crest badge
│       └── modal-combat.css          # Fullscreen combat arena, fighter HUDs, turn banner & animations
│
├── src/
│   ├── core/
│   │   ├── assets.js                 # Visual asset catalog (rooms, items, NPCs, avatars, combat backgrounds)
│   │   ├── state.js                  # Runtime GameState & world.json metadata synchronization
│   │   ├── audio.js                  # AudioManager (world playlist, combat BGM switching, mute, SFX hook)
│   │   └── network.js                # HTTP/SSE bridge client, TAP protocol parser & event router
│   │
│   ├── hud/
│   │   ├── topHudView.js             # Updates Top HUD HP bar and server population counter
│   │   ├── chatView.js               # Manages Room/World/Party/Combat/System tabs & safe message rendering
│   │   ├── playersDockView.js        # Updates Location & Party tables and room/server player counters
│   │   └── stageView.js              # Updates room backdrop, banner, left NPC quick dock & compass exits
│   │
│   ├── modals/
│   │   ├── lookModal.js              # LOOK window controller (5 tabs)
│   │   ├── selectorModal.js          # Shared selector modal for MOVE, TAKE, TALK, ATTACK (with respawn status)
│   │   ├── dialogueModal.js          # NPC conversation & quest acceptance / auto-completion display
│   │   ├── inventoryModal.js         # 4x4 bag grid, item inspector, USE & DROP actions
│   │   ├── statusModal.js            # Character HP, equipment bonuses calculation & world standing
│   │   ├── questsModal.js            # Active / Completed quest journal with dynamic world.json rewards
│   │   ├── groupModal.js             # Party creation, joining, invitations & member list (XSS-escaped)
│   │   ├── combatAnimator.js         # Frame-by-frame sprite animator & step timing calculator
│   │   └── combatModal.js            # Combat arena state, turn timer, action queue & outcome flow
│   │
│   └── ui.js                         # Master UI orchestrator, HTML escaping & global event delegation
│
└── assets/                           # Visual & audio assets (see Section 4)
```

---

## 4. Assets Organization (`frontend/gui/assets/`)

- `assets/avatars/` & `assets/avatars/icons/` — Full portraits and circular HUD icons for player avatars (`hero_male`, `hero_female`) and all NPCs.
- `assets/rooms/`, `assets/rooms/cards/`, `assets/rooms/icons/` — Fullscreen room scenes (`.webp`), modal preview cards, and compass/list icons (`.png`).
- `assets/items/card/` & `assets/items/icons/` — Detailed item cards and 4×4 inventory slot icons (`.webp`).
- `assets/combat/` & `assets/combat/anims/<actor>/<state>/` — Combat arena backgrounds (`bg_*.webp`), static sprites, and frame sequences (`idle`, `attack`, `defend`, `heal`, `hit`, `death`).
- `assets/frame/`, `assets/buttons/`, `assets/ui/` — Ornate 9-slice/corner PNG textures, textured button states (`base`, `blue`, `gold`, `red`), dividers, and crests.
- `assets/icons/command/` & `assets/icons/ui/` — Command toolbar icons and inline UI badges.
- `assets/audio/music/` & `assets/audio/sfx/` — World background playlist tracks, looping combat BGM, and combat sound effects.

---

## 5. Data & Event Flow

1. **Initialization:** On page load, `UI.initEventListeners()` mounts listeners and `App.restoreSessionIfActive()` fetches `/api/config` (CLI host/port defaults), `/api/world` (to populate `ASSETS` descriptions, combat stats, gear bonuses, and quest rewards without hardcoding game balance), and opens the SSE stream `/api/stream`.
2. **Command Execution:** User actions call `App.sendCommand(cmd)` → `POST /api/command` → mutex-serialized TCP write (`writeMu`) → TAP server.
3. **Asynchronous Updates & Interoperability:** Server `OK`, `ERR`, and `EVT` lines are pushed via SSE to `App.handleServerPacket()`. The parser supports both TAP's enriched JSON payloads and standard RFC 42TAP fallback formats (plain string arrays for `LOOK`/`INVENTORY`, plain text `TALK`, space-delimited `CHAT` events), preserving cross-group client compatibility.
4. **Combat Choreography:** Combat events (`STARTED` JSON participants, `TURN` with round and remaining actions, `DAMAGE`, `PLAYER FLED`, `NPC DIED`, `PLAYER DIED`, `ENDED` with reward items) feed into `CombatModal.animQueue` so each sprite animation, impact shake, and floating damage number plays sequentially with timing derived from `CombatAnimator.getStepTiming()`.
5. **Security & Rendering Safety:** Chat lines use `textContent`, while user-controlled party/player labels rendered inside modal templates pass through `UI.escapeHtml()`.

## 6. Local HTTP API

The Go bridge exposes a small local HTTP API. It is the boundary between
browser JavaScript and the TCP connection to the game server.

| Endpoint | Method | Responsibility |
|---|---|---|
| `/api/session` | GET | Reports whether the bridge currently has an authenticated TCP connection and returns the active username. Used to restore the frontend after a browser refresh. |
| `/api/connect` | POST | Accepts the server host, port, and username; opens TCP, validates `OK hello proto=1`, sends `CONNECT`, and stores the authenticated connection. |
| `/api/command` | POST | Accepts a TAP command and writes it to the active TCP connection. Writes are serialized with `writeMu`. |
| `/api/stream` | GET | Opens an SSE stream and forwards server messages to the browser. |
| `/api/world` | GET | Serves the configured world JSON file with caching disabled, allowing the frontend to load current world metadata. |
| `/api/config` | GET | Returns the configured server host and port used as login defaults.
| `/` and static paths | GET | Serves `frontend/gui` files (HTML, CSS, JavaScript, and assets). |

---

The `/api/command` response acknowledges the local HTTP request and TCP
write. The actual TAP command result (`OK` or `ERR`) is received
separately through `/api/stream`. This distinction is important: the
HTTP acknowledgement is not the game server's command result.

---

## 7. Bridge Lifecycle and Concurrency

### 7.1 Desktop startup

`cmd/desktop/main.go` acts as the composition root:

1. Loads shared configuration through `internal/config`.
2. Creates the desktop bridge with `desktop.NewApp()`.
3. Registers the API handlers and static file server on an HTTP mux.
4. Binds the local HTTP listener to `127.0.0.1:0`, allowing the operating system to select an available port.
5. Starts a browser window in application mode where a supported browser is available; otherwise it falls back to opening the local URL.
6. Handles `SIGINT` and `SIGTERM` and delegates cleanup to `App.Shutdown()`.

The desktop bridge is a separate process-side component. It does not
import or call `internal/world` or `internal/commands`; all gameplay
interaction goes through the TCP protocol.

### 7.2 TCP connection lifecycle

`internal/desktop.App` owns the active TCP connection and the
authenticated player name.

- `/api/connect` first closes any existing connection, then dials the requested server with a timeout.
- The bridge reads and validates the server greeting before sending `CONNECT <username>`.
- After `OK connected`, it stores the connection and starts `readLoop` in a goroutine.
- `readLoop` continuously reads complete TAP lines and classifies them as `EVT`, `ERR`, or `OK` before publishing them to SSE subscribers.
- `QUIT`, connection errors, or bridge shutdown close the connection and clear the active player/session fields.

The bridge uses `mu` to protect connection and player state, `writeMu`
to serialize TCP writes, and `subMu` to protect the set of active SSE
subscribers.

### 7.3 SSE fan-out

Each `/api/stream` request registers its own buffered subscriber
channel. The bridge forwards messages from the TCP reader to active
subscribers; each browser stream then writes them as SSE `data:` frames.

The stream is independent of individual HTTP command requests, so
asynchronous server events can reach the UI even when the player is not
sending a command.

Subscriber delivery is non-blocking. If a subscriber's buffer is full,
that particular event is dropped rather than blocking the TCP reader.
The subscriber buffer currently holds 32 messages.

When the last SSE subscriber disconnects, the bridge waits briefly
before shutting down. This grace period allows a browser refresh to
reconnect without immediately terminating the desktop client.

---

## 8. Frontend Runtime Responsibilities

The frontend is organized around a small set of cooperating modules:

| Area | Main files | Responsibility |
|---|---|---|
| Core state and | `core/state.js`, `core/assets.js` | Maintains the integration browser-side `GameState` and loads visual/game metadata from the world JSON. |
| Network and protocol | `core/network.js` | Calls the local HTTP API, opens the SSE stream, parses TAP lines, and routes responses/events to  the relevant UI modules. |
| Audio | `core/audio.js` | Controls exploration music, combat music, mute state, and sound-effect playback. |
| HUD and stage | `hud/*.js` | Updates the top HUD, room stage, NPC quick actions, chat, player tables, and compass. |
| Modal controllers | `modals/*.js` | Owns the behavior of the LOOK, selector, dialogue, inventory, status, quests, group, and combat windows. |
| Combat animation | `modals/combatAnimator.js` | Calculates animation step timing and coordinates sprite-frame sequences used by the combat UI. |
| UI orchestration | `ui.js` | Initializes event listeners, delegates user interactions, coordinates modules, and provides shared rendering helpers such as HTML escaping. |

---

`index.html` defines the static interface shell and loads the JavaScript
modules in dependency order: core modules first, then HUD modules, modal
controllers, and finally `ui.js`.

The ten HTML dialogs are LOOK, selector, dialogue, inventory, status,
quests, group, quit, combat, and flee. The quit and flee dialogs share
the `modal-quit.css` stylesheet; `combatAnimator.js` is an animation
helper, not a dialog controller.

---

## 9. State Ownership and Data Boundaries

The server remains authoritative for player and world state. The GUI
keeps a browser-side projection in `GameState` so it can render the
current room, player status, inventory, quests, group, chat, and combat
interface.

The frontend obtains different kinds of data through different paths:

- **Live game state:** TAP command responses and asynchronous events
    received over TCP and forwarded through SSE.
- **World metadata:** `/api/world`, used by the asset catalog and UI
    to resolve room descriptions, NPC/item metadata, combat values,
    equipment bonuses, and quest rewards.
- **Connection defaults:** `/api/config`.
- **Current bridge session:** `/api/session`.

The GUI may calculate presentation values (for example, displaying
equipment bonuses or animating an HP bar), but gameplay validation and
state transitions belong to the server.

---

## 10. Error Handling and Security

- Connection establishment and authentication use timeouts. A failed
    greeting, dial, or `CONNECT` response closes the attempted TCP
    connection.
- A command submitted while the bridge is disconnected receives a
    local HTTP error response.
- TCP read failures clear the active connection and publish a
    `DISCONNECTED` event to the browser.
- TCP writes are serialized to prevent concurrent handlers from
    writing to the same connection.
- Chat content is inserted using safe text APIs, and user-controlled
    labels used in HTML templates are escaped through `UI.escapeHtml()`.
- The local HTTP bridge binds to loopback (`127.0.0.1`), rather than
    exposing its control API on all network interfaces.

The bridge is intended as a local desktop companion for the GUI. It is
not a public web API and does not replace server-side validation.

---

## 11. Related Documentation

- [`cli-architecture.md`](cli-architecture.md) - terminal client
    architecture.
- [`server-architecture.md`](server-architecture.md) - TCP server,
    world domain, concurrency, and event distribution.
- [`../GUI_USER_GUIDE.md`](../GUI_USER_GUIDE.md) - player-facing GUI
    instructions.
- [`../CLI_USER_GUIDE.md`](../CLI_USER_GUIDE.md) - player-facing CLI
    instructions.
- [`../protocol-rfc.html`](../protocol-rfc.html) - TAP wire protocol
    specification.
