# TAP Server Architecture

This document describes the server architecture, package
responsibilities, TCP session lifecycle, command dispatch, shared world
model, event distribution, and operational controls of the TAP server
(`cmd/server` and `internal`).

---

## 1. Overview

The TAP server is a **Go TCP server** that implements the line-oriented
TAP protocol and manages the shared game world.

Its main responsibilities are:

- Accept and manage persistent TCP client connections.
- Send the initial protocol greeting and authenticate players with `CONNECT`.
- Parse and validate incoming TAP commands.
- Dispatch commands to dedicated command handlers.
- Maintain authoritative world, player, inventory, quest, group, NPC, and combat state.
- Return synchronous `OK` / `ERR` responses.
- Publish asynchronous world events and deliver them to relevant connected clients.
- Enforce connection and command-rate limits.
- Record structured operational and gameplay logs.

The server is the single source of truth for gameplay. CLI and GUI
clients communicate with it through the TAP protocol and do not access
its internal packages.

---

## 2. Entry Point and Initialization

| Path | Responsibility  |
|---|---|
| `cmd/server/main.go` | Initializes logging and configuration, creates abuse protection, loads and validates world data, wires the event bus and subscribers, registers commands, opens the TCP listener, admits connections, and starts one handler goroutine per admitted connection. |
| `internal/config/config.go` | Loads server configuration such as listen address, world file, timeouts, combat turn timeout, and abuse limits. |
| `internal/logger/logger.go` | Creates the structured logger and the audit logger used for server and world-event logging. |

Initialization order in `main.go`:

1. Create the structured logger.
2. Load server configuration.
3. Create the `AbuseGuard` and start its periodic cleanup routine.
4. Load the game world from the configured world file. Startup stops if loading or validation fails.
5. Create the session `Registry` and internal event `Bus`.
6. Subscribe the audit logger and notification dispatcher to the event bus.
7. Create and subscribe the combat-turn timeout manager; start its lifecycle routine.
8. Create, subscribe and start the `NPCRespawnManager`.
9. Create the command `Dispatcher` and register supported commands.
10. Open the TCP listener.
11. Accept connections, reserve capacity through the abuse guard, reject over-limit connections, and start a goroutine for each admitted connection.

The server loads world data once during startup. Runtime world changes
are held in memory; the project does not require persistent storage
across server restarts.

---

## 3. Directory Structure

``` text
cmd/
└── server/
    └── main.go                         # Server composition root and TCP accept loop

internal/
├── abuse/                              # Connection admission and request-rate policies
│   ├── guard.go                        # AbuseGuard state and policy coordination
│   ├── policies.go                     # Configurable limits and defaults
│   ├── connections.go                  # Connection leases and connection accounting
│   ├── rate_limit.go                   # Per-player command and CHAT rate checks
│   └── cleanup.go                      # Periodic removal of expired tracking records
│
├── commands/                           # TAP command handlers and dispatcher
│   ├── command.go                      # Command interface
│   ├── context.go                      # Execution context passed to handlers
│   ├── dispatcher.go                   # Parses requests and selects a handler
│   ├── register.go                     # Registers supported command handlers
│   ├── attack.go
│   ├── chat.go
│   ├── combat.go
│   ├── combat_errors.go
│   ├── drop.go
│   ├── group.go
│   ├── inventory.go
│   ├── look.go
│   ├── move.go
│   ├── quest.go
│   ├── quests.go
│   ├── take.go
│   ├── talk.go
│   ├── use.go
│   ├── status.go
│   ├── who.go
│   ├── quit.go
│   └── complete.go
│
├── config/
│   └── config.go                       # Shared runtime configuration
│
├── events/
│   └── bus.go                          # Internal publish/subscribe event bus
│
├── logger/
│   └── logger.go                       # Structured logger and event audit subscriber
│
├── protocol/                           # TAP message representation and wire parsing
│   ├── message.go                      # Protocol message types
│   ├── request.go                      # Parsed client request
│   ├── parser.go                       # Incoming request parsing
│   ├── encode.go                       # Outgoing message encoding
│   ├── codes.go                        # Protocol response/error codes
│   ├── errors.go                       # Protocol parsing and validation errors
│   └── username.go                     # Username validation
│
├── server/                             # TCP sessions and asynchronous delivery
│   ├── connection.go                   # Per-connection lifecycle and read loop
│   ├── connection_auth.go              # CONNECT parsing and player authentication
│   ├── connection_commands.go          # Rate limiting and command dispatch
│   ├── connection_cleanup.go           # Player/session cleanup on disconnect
│   ├── reader.go                       # Bounded line reader pump
│   ├── writer.go                       # Session outbox writer pump
│   ├── session.go                      # Active session and outgoing queue
│   ├── registry.go                     # Active session registry and message routing
│   ├── notification_dispatcher.go      # Converts world events into TAP EVT messages
│   ├── npc_respawn_manager.go
│   ├── npc_respawn_handler.go
│   └── combat_turn_timeout_manager.go  # Player combat-turn timeout lifecycle
│
└── world/                              # Authoritative game domain and rules
    ├── world.go                        # World aggregate and synchronized state
    ├── loader.go                       # World data loading
    ├── json_validation.go              # JSON structure validation
    ├── room.go
    ├── room_validation.go
    ├── npc.go
    ├── npc_queries.go
    ├── npc_validation.go
    ├── npc_respawn.go
    ├── players.go
    ├── movement.go
    ├── item.go
    ├── item_validation.go
    ├── inventory.go
    ├── items_use.go
    ├── group.go
    ├── quest.go
    ├── quest_actions.go
    ├── quest_queries.go
    ├── quest_validation.go
    ├── validate_relations.go
    ├── chat.go
    ├── errors.go
    ├── events.go
    └── combat/
        # Combat implementation is split across focused files in world/:
        # combat.go, combat_ai.go, combat_attack.go, combat_defend.go,
        # combat_effects.go, combat_end.go, combat_flee.go, combat_help.go,
        # combat_helpers.go, combat_participants.go, combat_progression.go,
        # combat_rewards.go, combat_stats.go, combat_turns.go,
        # combat_lifecycle.go
```

Note: combat source files are located directly in `internal/world/`; the
`combat/` label above is a conceptual grouping only, not a separate Go
package or directory.

---

## 4. Main Architectural Layers

The server is organized into cooperating layers. The boundaries keep TCP
handling, protocol representation, command behavior, game rules, and
event delivery separate.

| Layer | Package | Responsibility |
|---|---|---|
| Application startup | `cmd/server` | Wires dependencies and starts the listener. |
| Configuration | `internal/config` | Supplies runtime settings. |
| Admission control | `internal/abuse` | Applies connection and request-rate policies. |
| Transport/session | `internal/server` | Owns TCP lifecycle, sessions, reader/writer pumps, and recipient registry. |
| Protocol | `internal/protocol` | Parses and encodes TAP messages and validates protocol-level data. |
| Command dispatch | `internal/commands` | Maps a parsed request to the appropriate command handler. |
| Domain | `internal/world` | Owns authoritative game state and game rules. |
| Event distribution  | `internal/events` + `internal/server` | Publishes domain events and turns them into client notifications. |
| Observability | `internal/logger` | Writes structured logs and audits domain events. |

---

## 5. TCP Connection and Session Lifecycle

Each accepted connection is admitted by the abuse guard before a
connection handler is started.

### 5.1 Connection setup

`HandleAdmittedConnection` creates a `connectionHandler` and starts its
lifecycle:

1. Create a `Session` with a unique session ID, remote IP, TCP connection, and buffered `Outbox`.
2. Start the session `WritePump`.
3. Send the greeting `OK hello proto=1`.
4. Start the `ReadPump`.
5. Enter the connection read loop.
6. Require a valid `CONNECT <username>` command before normal gameplay commands are accepted.

Authentication has a timeout. Once authenticated, the handler registers
the session and player in the appropriate runtime structures and
publishes presence/player-count events.

### 5.2 Reader-handler-writer model

The connection uses separate read and write responsibilities:

``` text
                         TCP Connection
                         /             \
                        v               v
                   ReadPump          WritePump
                      |                  ^
                      v                  |
                ReadEvent channel        |
                      |                  |
                      v                  |
               connectionHandler         |
                      |                  |
                      v                  |
              Command Dispatcher         |
                      |                  |
                      v                  |
                SendMessage()            |
                      |                  |
                      v                  |
                Session.Outbox ----------+
```

- `ReadPump` reads bounded newline-terminated messages and reports lines or read/protocol errors.
- `connectionHandler` owns the connection state machine and decides whether a line is a `CONNECT` request or an authenticated command.
- `WritePump` is the only goroutine that writes to a session's TCP connection.
- `Session.Outbox` decouples message producers from socket writes.

This prevents multiple goroutines from writing concurrently to the same
TCP connection. A slow or disconnected client is handled at its own
session boundary rather than making every broadcast write directly to
the socket.

### 5.3 Session registry

`Registry` tracks active sessions and provides operations to register,
remove, find, and queue messages for connected players. Notification
delivery uses the registry to resolve recipients.

### 5.4 Disconnect and cleanup

Disconnect cleanup is centralized in `connection_cleanup.go`:

- Remove the player's runtime state from the world.
- Remove the session from the registry.
- Publish relevant departure, player-count, item-return, and combat events.
- Close the outbox and wait for the writer to finish.
- Close the TCP connection and wait for the reader to finish when it was started.

The cleanup path removes player/session state before publishing the
corresponding departure notifications, preventing broadcasts from
treating the disconnected player as still active.

---

## 6. Protocol Layer

`internal/protocol` defines the boundary between raw TCP lines and typed
TAP messages.

Its responsibilities include:

- Parse client request lines.
- Represent protocol messages and requests.
- Encode outgoing `OK`, `ERR`, and `EVT` messages.
- Define response/error codes.
- Validate protocol-level values such as usernames.

The transport uses UTF-8, line-oriented messages terminated by LF
(`\n`). The server sends the initial greeting and then processes the
client's `CONNECT` command before accepting game commands.

The protocol package does not implement game rules. It deals with
syntax, message structure, encoding, and protocol-level validation.

---

## 7. Command Pattern and Dispatcher

The `internal/commands` package separates command behavior from the TCP
connection loop.

Every command implements the `Command` interface:

``` go
type Command interface {
    Name() string
    Execute(ctx *Context) (protocol.Message, error)
}
```

The dispatcher parses an incoming command, finds the registered handler,
builds or uses the execution context, and invokes `Execute`.

The context provides the handler with the information it needs,
including the player/session identity, the shared world, the event
publisher, and the logger.

Conceptual flow:

``` text
Client command line
       |
       v
connectionHandler
       |
       v
Rate-limit check
       |
       v
commands.Dispatcher
       |
       v
Registered Command.Execute(context)
       |
       +------> world methods
       |
       +------> publish domain events
       |
       v
protocol.Message (OK / ERR)
       |
       v
Session.Outbox -> WritePump -> Client
```

Command handlers are grouped by behavior, for example `move.go`,
`look.go`, `inventory.go`, `quest.go`, `group.go`, `attack.go`, and
`combat.go`.

Handlers translate domain errors into protocol error responses. The
world package remains responsible for validating and applying game-state
transitions.

---

## 8. World Domain and Shared State

`internal/world` contains the authoritative game model.

The `World` struct owns maps for:

- Rooms and their exits.
- Players and their current state.
- Item definitions and unique item instances.
- Room item instances and player inventories.
- NPCs and quests.
- Player quest progress.
- Groups and player-to-group membership.
- Active combats and combat identifiers.

A `sync.RWMutex` protects shared world state. Read operations use read
access where possible; state-changing operations take the write lock.
Methods return data or events to callers rather than exposing mutable
internal maps to network clients.

### 8.1 Loading and validation

`world.Load(...)` loads the configured world data. Validation is divided
across focused files and checks the structure and relationships of
rooms, exits, NPCs, items, quests, and their references.

The server logs a load failure and does not proceed to listen if world
loading fails.

### 8.2 Domain responsibilities

| Area |Representative files | Responsibility |
|---|---|---|
| Rooms and movement | `room.go`, `room_validation.go`, `movement.go` | Room data, exits, validation, and player movement. |
| Players | `players.go` | Player lifecycle and player state. |
| NPCs | `npc.go`, `npc_queries.go`, `npc_validation.go`, `npc_respawn.go` | NPC data, room queries, validation, and respawn behavior. |
| Items and inventory | `item.go`, `item_validation.go`, `inventory.go`, `items_use.go` | Item definitions/instances, ownership, quantities, and item use. |
| Quests | `quest.go`, `quest_actions.go`, `quest_queries.go`, `quest_validation.go` | Quest state, progression, queries, and validation. |
| Groups | `group.go` | Group membership and invitations. |
| Chat | `chat.go` | Chat domain events and recipient selection. |
| Combat | `combat*.go` | Encounter lifecycle, participants, turns, AI, actions, stats, effects, rewards, and resolution. |

---

### 8.3 Runtime state

The world state is held in memory. The configured world file supplies
the initial static definitions; connected players, item ownership, quest
progress, group membership, and active combats are runtime state.

---

## 9. Event Bus and Asynchronous Notifications

The server uses an internal publish/subscribe event bus
(`internal/events`) to decouple game-state changes from network
delivery.

Domain operations can return typed events. Command handlers or server
lifecycle code publish those events. Subscribers react independently.

The main subscribers wired in `cmd/server/main.go` are:

- **Audit logger:** records domain events as structured log entries.
- **Notification dispatcher:** translates supported domain events into
    TAP `EVT` messages and queues them for the relevant sessions.
- **Combat turn timeout manager:** reacts to combat turn changes and
    schedules or cancels player-turn timeouts.
- **NPC respawn manager:** responds to a scheduled NPC respawn and posts a message stating that the NPC has respawned.

Conceptual event path:

``` text
World operation
      |
      v
Typed domain event
      |
      v
   Event Bus
      |
      +--------------------+
      |                    |
      v                    v
 Audit Logger     Notification Dispatcher
                           |
                           v
                    Session Registry
                           |
                           v
                     Session Outbox
                           |
                           v
                       WritePump
                           |
                           v
                        TCP Client
```

The event bus carries internal Go event values. The notification
dispatcher is the boundary that converts those values into the
wire-level TAP event format.

---

## 10. Combat Turn Timeout Manager

`internal/server/combat_turn_timeout_manager.go` manages deadlines for
player turns.

- It listens for combat turn-change events.
- When a player becomes the active participant, it starts or replaces that combat's timer.
- When the turn changes or combat ends, the corresponding timer is stopped or removed.
- NPC respawn manager - that it operates using the same pattern as the CombatTurnTimeoutManager (reacting to `NPCRespawnScheduled` and publishing `NPCRespawned`).
- When a timer expires, the configured callback delegates timeout resolution to the command/world-side handler.
- The manager is stopped when the server cleanup context is cancelled.

The manager coordinates timing; combat rules and state transitions
remain in `internal/world` and the command/event integration.

---

## 11. Abuse Protection

`internal/abuse` is separate from gameplay rules and applies resource/admission policies.

The default connection limits defined in the current implementation are:

| Policy | Default |
|---|---|
| Total active connections | 100 |
| Simultaneous connections per IP | 5 |
| Connection attempts per IP | 10 per minute |
| Unauthenticated connections | 20 |
| Commands per authenticated player | 30 per rolling 10 seconds |
| `CHAT` commands per authenticated player | 5 per rolling 10 seconds |

Connection admission is checked before a connection handler is launched.
Command rate limits are checked before dispatch. The cleanup worker
periodically removes expired tracking records.

The configured connection limits can be supplied through server
configuration/flags and their corresponding environment variables. The
current implementation also returns `ERR 429 RATE_LIMITED` when a
player's command limit is exceeded; this is an implementation extension
beyond the base RFC examples.

---

## 12. Logging and Auditing

The server uses Go's `log/slog` through `internal/logger`.

Logging is structured and includes timestamps, levels, event names, and
contextual attributes such as session ID, remote IP, player, command,
error code, room, combat, or quest identifiers when available.

The audit logger subscribes to the event bus. Command dispatch and
connection/session code also record operational details. This separates
diagnostic logging from protocol responses sent to clients.

Typical areas logged include:

- Server startup and world loading.
- Accepted and rejected connections.
- Authentication, disconnection, and timeout events.
- Commands and their outcomes.
- Protocol and internal errors.
- World changes and gameplay events.
- Abuse/rate-limit decisions.

---

## 13. End-to-End Request and Event Flow

### 13.1 Synchronous command

``` text
CLI / GUI
   |
   | TCP line: "MOVE north\n"
   v
TCP listener -> connectionHandler
   |
   v
ReadPump -> request line
   |
   v
AbuseGuard rate check
   |
   v
Protocol parser / Dispatcher
   |
   v
MoveCommand.Execute()
   |
   v
World movement method (World lock)
   |
   +----> typed PlayerMoved event -> Event Bus -> EVT notifications
   |
   v
protocol.OK(...)
   |
   v
Session.Outbox -> WritePump -> TCP response
```

### 13.2 Asynchronous world event

``` text
World state transition
        |
        v
Typed event published
        |
        v
     Event Bus
        |
        v
Notification Dispatcher
        |
        v
Resolve recipients in Registry
        |
        v
Queue EVT message in each Session.Outbox
        |
        v
WritePump sends to each TCP client
```

A client can therefore receive a server event independently of its own
command/response cycle.

---

## 14. Design Decisions

| Decision | Reason |
|---|---|
| Dedicated command dispatcher | Keeps command-specific behavior out of the socket read loop and makes handlers independently understandable and testable. |
| Separate protocol package | Centralizes parsing, message representation, encoding, and protocol error definitions. |
| World as authoritative aggregate | Keeps gameplay rules and mutable shared state on the server. |
| `sync.RWMutex` around world state | Protects concurrent reads and mutations from multiple player sessions. |
| Reader and writer pumps per connection | Separates socket reading from session writing and ensures one writer per TCP connection. |
| Buffered session outbox | Decouples broadcasts and command responses from socket writes. |
| Internal event bus | Decouples domain changes from audit logging and client notification delivery. |
| Typed domain events | Keeps internal game events distinct from their serialized TAP `EVT` representation. |
| Separate abuse guard | Applies connection and request limits without mixing them into game rules. |
| In-memory runtime state | Meets the project requirement that persistence across server restarts is not required. |

---

## 15. Related Documentation

- [`gui-architecture.md`](gui-architecture.md) - desktop GUI and Go
    HTTP/SSE bridge architecture.
- [`cli-architecture.md`](cli-architecture.md) - terminal client
    architecture.
- [`CLI_USER_GUIDE.md`](../CLI_USER_GUIDE.md) - player-facing CLI
    instructions.
- [`GUI_USER_GUIDE.md`](../GUI_USER_GUIDE.md) - player-facing GUI
    instructions.
- [`protocol-rfc.html`](../protocol-rfc.html) - TAP wire protocol
    specification.
