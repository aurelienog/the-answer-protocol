# GUI User Guide

## 1. Introduction

Welcome to the Desktop GUI client of **The Answer Protocol (TAP)**, a medieval multiplayer text-based adventure game set in the Kingdom of Arvendale.

The GUI client connects to any RFC 42TAP-compatible TCP server through a lightweight Go HTTP/SSE bridge and renders the shared world in a dedicated native browser window (`--app` mode), complete with illustrated locations, interactive NPC portraits, a directional compass, inventory management, real-time scoped chat, animated turn-based combat, and adaptive background music.

## 2. Getting Started

### Requirements

- **Go 1.27+**
- A running TAP server instance (e.g., `make run-server`)
- A Chromium-based browser (`google-chrome`, `chromium`, `chromium-browser`, `brave-browser`) or `firefox` installed for automatic window launch
- Run commands from the **repository root** so the client can serve `frontend/gui` and `data/world.json`

### Starting the Client

Build and launch the Desktop GUI client with the Makefile:

```bash
make run-client-gui
```

To target a custom server address or port:

```bash
make run-client-gui HOST=127.0.0.1 PORT=8090
```

Or run the compiled binary directly:

```bash
./bin/tap-desktop -host 127.0.0.1 -port 8090
```

Closing the GUI window (`X`) or pressing `Ctrl+C` in the terminal gracefully sends `QUIT` to the server and shuts down the client process.

### Authentication & Session Recovery

On the **Login Screen**:

1. Enter your **Character Name** (`1–12` characters, no spaces; letters, numbers, and underscores).
2. Choose your **Gender (Avatar)** (`Male` or `Female`). Your preference is saved in `localStorage`.
3. Verify the **Server Address** and **Port** (pre-filled from CLI flags or defaults `127.0.0.1:8090`).
4. Click **`✦ CONNECT ✦`** to enter Arvendale.

If you refresh the window (`F5`) while connected, the client automatically restores your active session via `/api/session` and resynchronizes room, status, inventory, quests, group, and ongoing combat state.

## 3. Interface Overview

The game screen is organized into five interactive zones:

### Top HUD Bar

- **Left (Hero Panel):** Displays your avatar, character name, and live **HP bar**.
- **Center:** The Answer Protocol crest.
- **Right (Server & Quick Actions):**
  - Live counters for players in the current **Room** and total players on the **Server**.
  - Quick system buttons: **`WHO`** (refresh online count), **`STATUS`** (open character sheet), and **`QUIT`** (disconnect confirmation).

### World Stage & Left NPC Dock

- Displays the full-screen illustrated backdrop and region banner of your current room.
- **Left NPC Quick Dock:** Lists every NPC currently in the room as an interactive card:
  - **Green (`TALK`):** Peaceful inhabitants and quest givers — click to open dialogue immediately.
  - **Red (`ATTACK`):** Hostile enemies — click to start combat (or join an active combat when help is requested).

### Right Action Bar

Provides one-click access to all core TAP protocol windows:

| Button | Protocol Command(s) | Purpose |
|---|---|---|
| **LOOK** | `LOOK` | Inspect room description, exits, ground items, NPCs, and players. |
| **MOVE** | `MOVE <direction>` | Choose an adjacent destination from available exits. |
| **TALK** | `TALK <npc>` | Select an NPC in the room and start a conversation. |
| **TAKE** | `TAKE <item>` | Inspect and pick up items located in the current room. |
| **ATTACK** | `ATTACK <npc>` / `COMBAT JOIN` | Inspect target combat stats and initiate or join combat. |
| **INVENTORY** | `INVENTORY`, `USE`, `DROP` | Manage carried gear, quest items, consumables, and common items. |
| **QUESTS** | `QUESTS` | Track active objectives and completed quests. |
| **GROUP** | `GROUP CREATE/JOIN/INVITE/LEAVE/INFO/INVITATIONS` | Create or manage a multiplayer party. |

### Interactive Compass Rose (Bottom Center)

- Automatically displays directional buttons (**North**, **East**, **South**, **West**) for the exits available in your current room, labeled with destination icons and short names.
- Click any direction pin to travel immediately (`MOVE <direction>`).

### Bottom Left: Chat & System Console

Separates player communication from raw protocol logs using five tabs:

| Tab | Scope / Behavior |
|---|---|
| **Room** | Messages visible to players in the current room (`CHAT ROOM`), plus presence and item updates. |
| **World** | Server-wide messages (`CHAT GLOBAL`). |
| **Party** | Private messages shared with your current group (`CHAT GROUP`). |
| **Combat** | Real-time battle log (strikes, damage, heals, retreats, loot, and calls for help). |
| **System** | Raw TAP protocol traffic (`Client →`, `Server ←`, `Event ←`). **Tip:** Typing in this tab sends raw TAP commands directly to the server (e.g., `WHO`, `TAKE Hempen Rope`, `STATUS`). |

Window controls in the top-right of the chat panel let you **expand chat height (+30%)** (`⤢`) or **minimize/restore** (`▾` / `▴`) the panel independently.

### Bottom Right: Location & Party Dock

- **Location Tab:** Lists all players currently standing in your room.
  - When you belong to a party, an **`Invite`** button appears next to ungrouped players in the room.
  - When a player calls for help during combat, an **`Aid`** button (`COMBAT JOIN`) appears next to their name.
- **Party Tab:** Lists your active group name, members, and Leader/Member rank.
- **Controls:** Audio Mute/Unmute toggle, Next Music Track button, and Minimize/Restore (`▾` / `▴`) toggle.

## 4. Gameplay Windows

### LOOK Window

Organized into 5 tabs: **Description**, **Exits** (clickable to travel), **Items** (with one-click **Take** buttons), **NPCs**, and **Players**.

### NPC Dialogue & Quests

Opening a conversation automatically runs `TALK <npc>` and displays the character's portrait and dialogue:

- If the NPC offers an available quest, click **`Accept Quest (QUEST)`**.
- While a quest is active, return with the required items and click **`Check / Turn In Quest`** (or **`Speak again (TALK)`**) to complete it automatically, claim rewards, and unlock the next quest in the chain.

### Inventory

- Displays carried items in a **4×4 slot grid** with stack quantities and an inspector panel on the right.
- **Equipment** and **Unique** items provide passive combat stat bonuses automatically while in your inventory.
- **Consumables** (`Healing Elixir`, `Harbor Vintage Wine`, `Fresh Bread`) can be activated with **`Use`** (`USE <item>`).
- **Common** and **Consumable** items can be released into the current room with **`Drop`** (`DROP <item>`). Equipment, unique, and quest items are bound and cannot be dropped.

### Character Status (`STATUS`)

Displays your full character portrait, current HP bar, health condition, live **Combat Statistics** (Base + Equipment bonuses for Attack, Defense, and Speed), equipped gear count, completed quests, party status, and online population.

### Quests Journal (`QUESTS`)

Switches between **Active** and **Completed** quests, showing the quest giver, description, objective progress (`0/1`, `1/1`), and rewards synchronized from `data/world.json`.

### Group Management (`GROUP`)

- Enter a group name to **Create** (`GROUP CREATE <name>`) or **Join** (`GROUP JOIN <name>`), or click **Leave** (`GROUP LEAVE`).
- Enter another player's username to **Invite Player** (`GROUP INVITE <player>`).
- Click **Check Invites** (`GROUP INVITATIONS`) to view pending party invites and accept them with one click (**`Accept & Join`**).

## 5. Turn-Based Combat Arena

Initiating combat with a hostile NPC (`Dire Wolf`, `Shadow Smuggler`, or `Dock Warden`) opens the **Full-Screen Combat Arena** while keeping the bottom chat and party dock accessible:

- **Top Combat Bar:** Shows Hero HP (left), active Turn Banner with Round number and **30-second countdown timer** (center), and Enemy HP (right).
- **Animated Stage:** Plays frame-by-frame combat animations (`idle`, `attack`, `defend`, `heal`, `hit`, `death`) synchronized with server damage events and floating damage/heal numbers.
- **Combat Actions** (active during your turn):
  - **Attack (`COMBAT ATTACK`):** Strike the enemy using your effective Attack (Base + Equipment). With speed bonuses (e.g., *Pathfinder Boots*), you can earn multiple actions in a round.
  - **Defend (`COMBAT DEFEND`):** Brace for +50% Defense until your next turn.
  - **Heal (`COMBAT USE`):** Automatically consumes your strongest available healing item (`Healing Elixir` → `Harbor Vintage Wine` → `Fresh Bread`) without spending combat action points.
  - **Help (`COMBAT HELP`):** Broadcasts a call for assistance so other players in the room can join via the **Aid** button or by clicking the enemy.
  - **Flee (`COMBAT FLEE`):** Opens a confirmation dialog to escape combat immediately (incurs a non-lethal HP penalty; cannot drop below 1 HP).
- **Cooperative Combat:** Party members in the same room join automatically when combat starts. If you fall while an ally is still fighting, you enter **Spectator Mode** until the encounter concludes. Defeated players respawn at **Central Harbor Square** with 25% HP.
- **Enemy Respawn:** Recently defeated enemies appear as `0 / Max HP (Respawning)` in the `ATTACK` selector until their server respawn timer expires.

## 6. Audio & Responsive Layout

- **Music Engine:** Plays a 4-track Arvendale exploration playlist sequentially, automatically switches to dedicated combat BGM during battles, and remembers your mute preference in `localStorage`. Use the melody icon in the bottom-right dock to skip to the next track.
- **Multi-Client & Responsive Support:** The interface adapts automatically to **2K/ultrawide monitors**, standard desktop windows, **compact split-screen layouts** (ideal for testing 2–4 clients simultaneously on one screen), and mobile portrait/landscape screens.

## 7. Troubleshooting

### Cannot connect (`ERR 900 CONNECTION_FAILED`)

- Confirm the TCP server is running (`make run-server`) and listening on the matching port (`8090` by default).
- Verify the **Server Address** and **Port** fields on the login screen.

### Window does not open automatically

- The terminal prints the local URL (e.g., `TAP Desktop GUI started! Window URL: http://127.0.0.1:<port>`). Open that URL manually in any modern browser.

### Missing backgrounds or room metadata

- Always launch `./bin/tap-desktop` or `make run-client-gui` from the **project root directory** so `./frontend/gui` and `./data/world.json` resolve properly.

