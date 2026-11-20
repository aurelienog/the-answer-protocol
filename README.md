*This project has been created as part of the 42 curriculum by oshtohri, aunoguei.*

# The Answer Protocol (TAP)

## Table of Contents

- [Description](#description)
- [Instructions](#instructions)
- [Architecture](#architecture)
- [Protocol Implementation](#protocol-implementation)
- [Combat System](#combat-system)
- [Quest System](#quest-system)
- [World Design](#world-design)
- [Server Logging & Abuse Controls](#server-logging--abuse-controls)
- [Group Contributions](#group-contributions)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Resources](#resources)

## Description

**The Answer Protocol (TAP)** is a TCP-based multiplayer text adventure protocol and game server written in Go, accompanied by an interactive terminal client (TUI) and a desktop GUI client.

Built upon RFC 42TAP, the project provides a shared-world environment where multiple players can connect concurrently, authenticate, explore interconnected rooms, interact with NPCs, manage item instances, complete quest chains, form groups, engage in turn-based combat, and communicate via real-time chat channels.

## Instructions

### Requirements

- **Go 1.27+**
- The world data file available at `data/world.json`.

### Quick Start

1. **Install dependencies and compile all binaries** (`bin/tap-server`, `bin/tap-cli`, `bin/tap-desktop`):
   ```bash
   make install
   ```
2. **Start the TCP server:**
   ```bash
   make run-server
   ```
3. **Connect using the CLI or GUI client:**
   ```bash
   make run-client      # Terminal TUI client
   make run-client-gui  # Desktop GUI client
   ```
4. **Remove compiled binaries:**
   ```bash
   make clean
   ```

Once connected, authenticate with a username (`CONNECT <username>`) and interact with the world (`LOOK`, `MOVE <dir>`, `TAKE <item>`, `TALK <npc>`, `ATTACK <npc>`, `CHAT GLOBAL <msg>`, `QUIT`). For detailed CLI controls and local commands, see [CLI_USER_GUIDE.md](CLI_USER_GUIDE.md).

## Architecture

TAP decouples network transport, protocol parsing, command execution, sessions and game state management.

```
+--------------------+      +--------------------+
|  TUI Client (CLI)  |      | Desktop GUI Client |
+---------+----------+      +---------+----------+
          ^                           ^
          |   TCP (UTF-8 lines)       |
          +-------------+-------------+
                        |  ^
       Commands (in)    |  |  OK / ERR / EVT (out)
                        v  |
          +---------------------------+
          |   Abuse & Rate Limiter    |
          +-------------+-------------+
                        |  ^
                        v  |
          +---------------------------+      +--------------------+
          |    Session Management     |<-----|  Session Registry  |
          |  (ReadPump / WritePump)   |      | (Active Sessions)  |
          +-------------+-------------+      +---------+----------+
                        |  ^                           ^
                        v  |                           | Resolve
          +---------------------------+                | recipients
          |      Protocol Parser      |                |
          +-------------+-------------+                |
                        |  ^                           |
                        v  |                           |
          +---------------------------+                |
          |    Command Dispatcher     |                |
          +-------------+-------------+                |
                        |  ^                           |
                        v  |                           |
          +---------------------------+                |
          |  World State (RWMutex)    |                |
          +-------------+-------------+                |
                        |                              |
                        v                              |
          +---------------------------+                |
          |  Event Bus & Notifier     |----------------+
          +---------------------------+
```


### Repository Layout

| Component | Responsibility |
|---|---|
| `cmd/server` `cmd/cli` `cmd/desktop` | Entry points for the TCP server, TUI client, and desktop GUI client.|
| `internal/protocol` | UTF-8 line framing, message parsing, response encoding, and error codes.|
| `internal/commands` | Command registry and isolated gameplay handlers (Command Pattern).|
| `internal/server` | TCP session lifecycle, read/write pumps, session registry, and notification dispatch.|
| `internal/world` | Thread-safe authoritative game state (rooms, items, NPCs, quests, combat, groups).|
| `internal/events` | Internal event bus. |
| `internal/abuse` | Connection admission limits and rolling-window rate limiting. |
| `internal/config` | Handles application configuration. |
| `frontend/cli` | Interactive Bubble Tea terminal client (networking, local projection, commands, and TUI views). |
| `frontend/gui` | Desktop graphical client interface and state integration. |
| `data/` | Declarative world configuration (`world.json`). |

### Server Concurrency & Design Patterns

- **Command Pattern (`Dispatcher`):** Instead of executing game logic inside the socket read loop, incoming commands are parsed and routed through a central `Dispatcher` (Command Pattern). This isolates command validation from connection management and simplifies unit testing.
```
Parsed Message
      |
      v
  Dispatcher
      |
      +------ LOOK ------> LookHandler
      |
      +------ MOVE ------> MoveHandler
      |
      +------ TALK ------> TalkHandler

...
```

- **Reader–Writer–Outbox Model:** Each session spawns dedicated `ReadPump` and `WritePump` goroutines connected via a buffered `Outbox` channel. A single goroutine owns TCP writes per connection, preventing race conditions and ensuring slow clients do not block server broadcasts.

```text
                    TCP Connection
                          |
              +-----------+-----------+
              |                       |
              v                       v
         ReadPump                 WritePump
              |                       ^
              |                       |
              v                       |
         Command Loop                 |
              |                       |
              v                       |
         Command Handler              |
              |                       |
              v                       |
         Session.Outbox --------------+
              ^
              |
       Registry.Broadcast
```

- **Shared State & Event Bus:** The `World` struct is the single source of truth, guarded by a `sync.RWMutex`. State mutations emit typed domain events (`PlayerMoved`, `CombatDamage`, `ChatMessage`, etc.) to the internal event bus. A notification dispatcher resolves recipient sessions via the `Registry` and pushes asynchronous `EVT` frames to their outboxes.

```text
TCP Client ──► ReadPump ──► Protocol Parser ──► Dispatcher ──► World (RWMutex)
                                                                     │
TCP Client ◄── WritePump ◄── Session.Outbox ◄── Notifier ◄── Event Bus
```

### Client Architecture (TUI & GUI)

#### TUI

The CLI client uses **Bubble Tea**  (Elm Architecture: `Model`, `Update`, `View`) over a four-layer pipeline (`network`, `game`, `commands`, `ui`). 

To synchronize request/response cycles without request IDs, `RequestManager` enforces one in-flight request at a time, while `ReadPump` routes `OK/ERR` replies to the active caller and pushes `EVT` frames to a buffered channel (capacity `128`) for asynchronous UI updates.

| Layer | Responsibility |
|---|---|
| `network` | Manages TCP transport, authentication, read/write pumps, request serialization, and asynchronous event routing. |
| `game` | Maintains a local read-only projection of the player, current room, inventory, quests, party, and chat logs. The server remains the sole authority. |
| `commands` | Parses user input and distinguishes between local UI commands and remote TAP protocol requests. |
| `ui` | Implements the Bubble Tea state loop, responsive layout management, and modular views (world, room, inventory, character, quests, party, combat). |

#### GUI

## Protocol Implementation

Communication uses UTF-8, `\n`-terminated lines over TCP with three message prefixes: `OK [payload]`, `ERR <code> <NAME>`, and `EVT <TYPE> [data]`.

### Supported Commands

| Command | Syntax | Behaviour | Successful response |
|---|---|---|---|
| `LOOK` | `LOOK`| Displays the current room, players, item definitions, and NPC names as JSON.| `OK <JSON>`|
| `QUIT` | `QUIT`| Terminates the player's session.| `OK bye` |
| `MOVE` | `MOVE <direction>`| Moves the player through an available exit.| `OK room=<room_id>`|
| `TAKE` | `TAKE <item>` | Takes an item from the current room. | `OK taken=<item_id>` |
| `DROP` | `DROP <item>` | Drops an item from the player's inventory. | `OK dropped=<item_id>` |
| `INVENTORY` | `INVENTORY` | Lists inventory quantities grouped by item ID as JSON. | `OK [{"id":"item_id","quantity":1}]` |
| `TALK` | `TALK <npc>` | Starts a conversation with an NPC and retrieves dialogue and quest information. | `OK <JSON>` |
| `QUEST` | `QUEST <npc>` | Accepts an available quest offered by the specified NPC. | `OK <JSON>` |
| `QUESTS` | `QUESTS` | Lists the player's discovered quests and their statuses. | `OK <JSON>` |
| `WHO` | `WHO` | Retrieves information about connected players. | `OK players=<total player count>` |
| `GROUP` | `GROUP <subcommand>` | Manages player groups and invitations. | Depends on subcommand |
| `CHAT` | `CHAT <scope> <message>` | Sends a message to a global, room or group audience. | `OK` |
| `ATTACK` | `ATTACK <npc>` | Starts a combat encounter with a hostile NPC. | `OK <JSON>` |
| `STATUS` | `STATUS` | Retrieves the player's health and status as JSON. | `OK <JSON>` |
| `COMBAT` | `COMBAT <action> [arguments]` | Performs an action in an active combat. | Depends on action |

### RFC 42TAP Deviations & Extensions

The following response shapes and command semantics intentionally extend or differ from the RFC examples.

| Feature | RFC 42TAP Specification | TAP Implementation & Justification | 
|---|---|---|
|`LOOK` & `INVENTORY`|Lists raw item/NPC objects or individual item instances.|`LOOK` returns item definitions, room item counts ({"item.herbs": 2}), and NPC details. `INVENTORY` aggregates quantities by item ID ([{"id":"item.herbs","quantity":2}]) while tracking unique instances internally, reducing payload bloat for UI rendering.|
|`TALK` & `QUEST`|`TALK` returns plain text; quest acceptance/completion flow is open.|`TALK` returns structured JSON (NPC stats, dialogue, quest status). `QUEST <npc>` accepts available quests immediately, and talking to the giver again auto-completes the quest once requirements are met—eliminating redundant turn-in commands.|
|`GROUP CREATE`|Syntax written as `GROUP CREATE` without arguments.|Requires `GROUP CREATE <name>` (spaces allowed) so parties have human-readable identifiers in invites and UI views.|
|Error Codes|Representative `4xx` (non-fatal) and `9xx` (fatal) codes.|Extended with granular `4xx` codes (`401`–`422`, e.g., `412 ITEM_CANNOT_BE_DROPPED`, `415 NPC_RESPAWNING`) and `429 RATE_LIMITED` so clients can handle gameplay restrictions cleanly without dropping the session.|

## Combat System

Combat is turn-based, deterministic, and server-authoritative so encounters progress smoothly without real-time client synchronization:

- Initiative & Action Economy: Effective speed determines turn order at the start of each round and grants speed points; every combat action costs 10 speed points. NPC turns execute automatically, and player turns have an expiration timer so inactive players cannot stall an encounter.

- Damage Formula: $\text{Damage} = \max(1, \text{Effective Attack} - \text{Effective Defense})$.

- Combat Actions (`COMBAT <action>`):
     - `ATTACK`: Deals standard damage to the target.
     - `DEFEND`: Grants a +50% defense bonus until the player's next turn.
     - `USE <item>`: Consumes an eligible healing/buff item from inventory.
     - `HELP` / `JOIN`: Allows a player to request assistance and allies in the room to join an active encounter.
     - `FLEE`: Exits combat immediately at the cost of a health penalty (cannot reduce the player below 1 HP).

## Quest System

Quest progress is tracked per player and driven declaratively by data/world.json (validated against cycles on startup).

- States: `locked` $\rightarrow$ `available` $\rightarrow$ `active` $\rightarrow$ `completed`.

- Progression & Validation: Talking to an NPC (`TALK <npc>`) reveals available quests and triggers state-specific dialogue (`default`, `available`, `active`, `completed`, `locked`). Running `QUEST <npc>` moves an `available` quest to `active`. When the player holds the required item quantities and speaks to the giver again, the server validates inventory counts, optionally consumes the required items, grants item rewards, and unlocks the next quest in the chain. `QUESTS` lists all active and completed quests.

## World Design

## World Design

The world of **Arvendale** is loaded from `data/world.json`, which undergoes strict pre-start validation (rejecting malformed JSON, duplicate keys, missing fields, or broken entity references).

### World Layout & Room Connections

The map consists of **9 interconnected rooms** spanning a coastal city, a harbor district, and a northern forest edge:

```text
                    [Whispering Forest Edge] (Dire Wolf)
                               |
                        [North Town Gate] (North Gate Guard)
                               |
[Royal Blacksmith Forge] -- [Market Street] -- [Apothecary Shop]
   (Master Borin)        (Guide & Advisor)      (Apprentice Elisa)
                               |                        |
                    [Central Harbor Square] -- [The Salty Siren Tavern]
                       (Captain Eliza)                  (Mara)
                               |
                        [The Great Pier] ----- [Abandoned Warehouse]
                         (Dock Warden)           (Shadow Smuggler)
```

#### NPC Roles & Progression

The world features 10 NPCs divided into friendly town inhabitants and hostile encounters with automatic respawn timers:

- Quest Givers & Social NPCs:

      - **Quest Advisor** (`Market Street`): Offers the introductory scouting quest (quest.explore_arvendale).
      - **Apprentice Elisa** (`Apothecary Shop`): Offers the forest herb quest (quest.gather_herbs) and a repeatable potion-brewing bounty.
      - **Captain Eliza** (`Central Harbor Square`): Leads the main two-part harbor conspiracy arc (`quest.expose_smugglers` $\rightarrow$ `quest.investigate_missing_ships`) plus a repeatable smuggler bounty.
      - **Mara** (`The Salty Siren Tavern`): Tavern keeper offering a repeatable bounty for defeating the Dock Warden.
      - **Town Guide**, **Master Borin** & **North Gate Guard**: Provide world navigation tips, combat gear hints, and perimeter lore.
      
- Hostile Encounters (with loot drops & respawn):
   - **Dire Wolf** (`Whispering Forest Edge`): Entry-level beast (45 HP, 45s respawn); drops Healing Herbs.
   - **Shadow Smuggler** (`Abandoned Warehouse`): Mid-tier enemy (75 HP, 60s respawn); drops the Incriminating Letter.
   - **Dock Warden** (`The Great Pier`): End-game boss (140 HP, 24 ATK, 14 DEF, 90s respawn); drops the `Missing Ships Report` and requires full armor/weapons to defeat.

### Items, Instances & Distribution
The engine separates immutable **Item Definitions** (17 total) from runtime **Item Instances** to track individual ownership and ensure `unique` items return to the world if a player disconnects:

| Category | Examples & Distribution | Gameplay Effect |
| --- | --- | --- |
| **Unique** | `Iron Sword` (+6 ATK) & `Oak Shield` (+4 DEF) in `Royal Blacksmith Forge`; `Purse of Gold Coins`. | Passive combat stat bonuses; return to spawn room on disconnect. |
| **Equipment** | `Pathfinder Boots` (+2 DEF, +5 SPD), `Alchemist's Iron Gauntlets` (+3 ATK, +3 DEF), `Royal Guard Breastplate` (+6 DEF), `Helm of Arvendale` (+2 ATK, +5 DEF). | Earned exclusively as quest rewards to scale player stats for boss encounters. |
| **Consumable** | `Healing Elixir` (+40 HP) in `Apothecary Shop`; `Fresh Bread` (+15 HP) & `Harbor Vintage Wine` (+25 HP) in `The Salty Siren Tavern`. | Restores HP during combat (`COMBAT USE <item>`) or exploration. |
| **Quest** | `Northern Gate Seal` (`North Town Gate`), `Healing Herbs` (`Whispering Forest Edge`), `Incriminating Letter` (`Abandoned Warehouse`), `Missing Ships Report` (`The Great Pier`). | Required objectives for quest completion (can be gathered or looted from enemies). |
| **Common** | `Hempen Rope` (`Harbor Square`), `Leather Pouch` (`Market Street`), `Brass Lantern` (`The Great Pier`). | Atmospheric world props that can be picked up, dropped, and traded between rooms. |

### Groups

Players may belong to one group at a time (`GROUP CREATE <name>`), enabling private `GROUP` chat, coordinated room exploration, and cooperative combat (`COMBAT HELP` / `COMBAT JOIN`).

## Server Logging & Abuse Controls

### Server Logging

The server uses Go's standard log package to record startup validation, connection lifecycle events, and asynchronous delivery failures (e.g., logging the target player name and error whenever a Session.Outbox enqueue fails or overflows). Operators can redirect stdout/stderr to a log aggregator or file to monitor connection spikes, session disconnects, and dropped notifications.

### Abuse Detection & Rate Limiting

The server's `internal/abuse` package applies connection admission and request-rate policies independently of game rules. By default, it allows up to 100 active connections globally, 5 simultaneous connections per IP address, 10 connection attempts per IP per minute, and 20 unauthenticated connections globally. Authenticated connections no longer count against the unauthenticated limit.

The limits can be configured with server flags or environment variables:

| Flag | Environment variable | Default |
|---|---|---:|
| `-max-connections` | `TAP_MAX_CONNECTIONS` | 100 |
| `-max-connections-per-ip` | `TAP_MAX_CONNECTIONS_PER_IP` | 5 |
| `-max-connection-attempts-per-ip` | `TAP_MAX_CONNECTION_ATTEMPTS_PER_IP` | 10 per minute |
| `-max-unauthenticated-connections` | `TAP_MAX_UNAUTHENTICATED_CONNECTIONS` | 20 |

Per authenticated player, the server allows up to 30 commands and 5 `CHAT` commands in each rolling 10-second window. A rate-limited command receives `ERR 429 RATE_LIMITED`.

This is an implementation extension because RFC 42TAP does not define a rate-limit error code. 
Rejected connections are closed, and expired per-IP and rate-limit records are periodically removed.

## Group Contributions

- **`oshtohri`:**
  - **Desktop GUI Client (`frontend/gui`):** Designed and implemented the graphical desktop client interface and state integration.
  - **World Design & Data (`data/world.json`):** Created the game world layout, room interconnections, NPC definitions and dialogue trees, item distributions, and quest chains.
- **`aunoguei`:**
  - **TCP Server & Core Engine (`cmd/server`, `internal/`):** Implemented the protocol parser, command dispatcher, session management, concurrency model, combat/quest logic, event bus, and abuse/rate-limiting controls.
  - **Terminal UI Client (`frontend/cli`):** Built the interactive Bubble Tea CLI client, networking pipeline, and local game state synchronization.

## Building and Running

The project uses a Makefile to manage builds and execution across all three binaries:

```bash
# Build all binaries into bin/ (tap-server, tap-cli, tap-desktop)
make build        # or `make install` to tidy modules and build

# Run the server (accepts optional flags such as -addr or abuse limits)
make run-server   # or `./bin/tap-server -addr :4242`

# Run the Terminal UI client
make run-client   # or `./bin/tap-cli`

# Run the Desktop GUI client
make run-client-gui # or `./bin/tap-desktop`

# Remove compiled binaries
make clean
```

## Testing

You can verify the server and client mechanics end-to-end by running a local server (`make run-server`) and connecting two or more clients simultaneously (`make run-client` or `make run-client-gui`).

### 1. Multiplayer Functionality (Presence, Chat & Groups)

1. **Concurrent Connections & Presence:** Connect two clients in separate terminals (`CONNECT alice` and `CONNECT bob`) in the same starting room. Verify that `bob` triggers an asynchronous `EVT ROOM_PRESENCE_ENTER bob` notification on `alice`'s screen, and that `WHO` reflects the updated player count.

2. **Scoped Chat:**
   - Run `CHAT ROOM Hello` — verify only players in the same room receive `EVT ROOM CHAT`.
   - Move one player away (`MOVE <dir>`) and run `CHAT GLOBAL Hi all` — verify all connected players receive `EVT GLOBAL CHAT`.
   
3. **Party Coordination:** Have `alice` run `GROUP CREATE Raiders` and `GROUP INVITE bob`. Have `bob` check `GROUP INVITATIONS`, accept with `GROUP JOIN Raiders`, and exchange messages via `CHAT GROUP <msg>`.

### 2. Combat System (Turns, Actions & Coop)
1. **Initiating Combat & Turn Order:** Locate a hostile NPC (`LOOK`) and start an encounter with `ATTACK <npc>`. 
2. **Combat Actions:**
   - Test `COMBAT ATTACK` to verify damage calculation ($\max(1, \text{Attack} - \text{Defense})$).
   - Test `COMBAT DEFEND` to confirm the +50% defense mitigation on the NPC's counter-attack.
   - Test `COMBAT USE <consumable>` to restore HP during a turn.
3. **Cooperative Combat & Fleeing:** Have `alice` call `COMBAT HELP` while `bob` is in the same room, then have `bob` enter the encounter with `COMBAT JOIN`. Finally, test `COMBAT FLEE` to confirm immediate disengagement with a non-lethal HP penalty (minimum 1 HP).

### 3. Quest System (Progression, Validation & Chains)
1. **Discovery & Acceptance:** Talk to a quest-giving NPC (`TALK <npc>`) to view its available quest dialogue, then accept it with `QUEST <npc>`. Verify via `QUESTS` that its state transitions from `available` to `active`.
2. **Requirement Validation & Completion:** Attempt to talk to the giver before gathering the required items (triggers active/in-progress dialogue). Collect the required items (`MOVE`, `TAKE <item>`), return to the NPC, and run `TALK <npc>` again to verify automatic validation, item consumption (if configured), reward delivery, and progression to the next quest in the chain (`completed` $\rightarrow$ next `available`).

### Automated Unit & Concurrency Tests
To run the automated test suite covering protocol parsing, state transitions, and race-condition checks:
```bash
go test -race tests/...
```

## Resources

### Go & Concurrency
- [A Tour of Go](https://go.dev/tour/list)
- [Go's Declaration Syntax](https://go.dev/blog/declaration-syntax)
- [Go Playground](https://go.dev/play/)
- [Go by Example](https://gobyexample.com/)
- [Gophercises](https://github.com/gophercises)
- [W3Schools Go Tutorial](https://www.w3schools.com/go/)
- [JSON and Go](https://go.dev/blog/json)
- [YAML vs JSON vs XML in Go](https://medium.com/geekculture/yaml-vs-json-vs-xml-in-go-bf4ebd1066f2)
- [Go Concurrency Patterns (Rob Pike)](https://www.youtube.com/watch?v=f6kdp27TYZs)
- [Advanced Go Concurrency Patterns](https://www.youtube.com/watch?v=QDDwwePbDtw)
- [Unpacking Go Channels: A Peek Under the Hood](https://dev.to/shrsv/unpacking-go-channels-a-peek-under-the-hood-2c42)

### Networking, Protocol & Security
- [Building a TCP Chatroom in Golang](https://medium.com/@pickleprat/building-a-tcp-chatroom-in-golang-6f6bd2b11bb7)
- [Why Many Backends Are Switching to Go](https://medium.com/@eikhapoetra/why-are-many-backend-switching-to-go-golang-abd3c5431997)
- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)

### Terminal UI (Charmbracelet Ecosystem)
- [Bubble Tea (TUI Framework — Elm Architecture)](https://github.com/charmbracelet/bubbletea) & [Go Reference](https://pkg.go.dev/github.com/charmbracelet/bubbletea)
- [Lip Gloss (Terminal Styling & Layout)](https://github.com/charmbracelet/lipgloss) & [Go Reference](https://pkg.go.dev/github.com/charmbracelet/lipgloss)
- [Bubbles (TUI Components & Animations — Spinners, Progress, Viewports)](https://github.com/charmbracelet/bubbles) & [Go Reference](https://pkg.go.dev/github.com/charmbracelet/bubbles)


### AI Usage

AI tools were used iteratively as a pair-programming collaborator throughout the entire lifecycle of the project, with all architectural decisions, generated code, and behaviors reviewed, tested, and validated by the authors:

- **Architecture & System Design:** Explored concurrency models (Reader–Writer–Outbox pattern, `sync.RWMutex` state synchronization, publish-subscribe event bus), protocol framing, and client-server decoupling before writing code.

- **Core Implementation (Server & Clients):** Assisted step-by-step in implementing the TCP server, RFC 42TAP parser, command dispatcher, combat and quest state machines, rate-limiting controls, as well as the Bubble Tea CLI client and desktop GUI client.

- **World Data & GUI Assets:** Helped structure and validate `data/world.json` (room connections, NPC dialogue trees, item definitions, and quest chains) and generated visual assets for the desktop GUI client.

- **Testing & Debugging:** Used to design unit and concurrency test suites (`-race`), identify protocol edge cases, and troubleshoot synchronization issues.

- **Documentation:** Assisted in organizing, condensing, and proofreading `README.md` and `CLI_USER_GUIDE.md` against the 42 subject requirements.