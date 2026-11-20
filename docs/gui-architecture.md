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
4. **Combat Choreography:** Combat events (`STARTED` JSON participants, `TURN` with round and remaining actions, `DAMAGE`, `PLAYER_FLED`, `NPC_DIED`, `PLAYER_DIED`, `ENDED` with reward items) feed into `CombatModal.animQueue` so each sprite animation, impact shake, and floating damage number plays sequentially with timing derived from `CombatAnimator.getStepTiming()`.
5. **Security & Rendering Safety:** Chat lines use `textContent`, while user-controlled party/player labels rendered inside modal templates pass through `UI.escapeHtml()`.
