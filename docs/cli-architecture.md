# TAP CLI Client Architecture

This document describes the architecture, source structure, networking
pipeline, application state, and rendering flow of the TAP terminal
client (`cmd/cli` and `frontend/cli`).

---

## 1. Overview

The CLI is an interactive terminal client built with **Go and Bubble
Tea**. Its interface follows Bubble Tea's Model--Update--View (Elm
Architecture) and uses the Charm libraries for terminal rendering and
viewport management.

The client communicates with the TAP server through a persistent TCP
connection. It uses the public line-based TAP protocol rather than
calling server packages directly. This keeps the CLI independent of
server implementation details and allows it to connect to another RFC
42TAP-compatible server.

Main responsibilities:

- Establish a TCP connection and validate the server greeting.
- Authenticate the player with `CONNECT`.
- Send TAP commands and process their `OK` / `ERR` responses.
- Receive asynchronous `EVT` messages while the user interacts with the interface.
- Maintain a local projection of player and world data for display.
- Render the terminal views and handle keyboard input.

The server remains authoritative: the CLI's state is a client-side
representation, not the source of truth for game rules or world state.

---

## 2. Entry Point and Startup

| Path | Responsibility |
|---|---|
| `cmd/cli/main.go` | Loads shared configuration, dials the configured server address, creates the network client, constructs the Bubble Tea model, and starts the terminal program with the alternate screen enabled. |
| `internal/config/config.go` | Loads the server connection configuration shared by the command-line applications. |

---

Startup sequence:

1. `config.LoadConfig(...)` loads the server host and port.
2. `network.Dial(ctx, cfg.ServerAddr())` opens a TCP connection and validates the initial greeting (`OK hello proto=1`).
3. `network.NewClient(...)` creates the client networking components.
4. `ui.NewModel(client)` creates the initial UI model in authentication state.
5. `tea.NewProgram(...)` starts the Bubble Tea event loop.
6. The user enters a username; the client sends `CONNECT <username>`.
7. After successful authentication, the network reader and writer pumps start and the game interface becomes available.

The CLI does not start or embed a game server. It is a separate TCP
client process.

---

## 3. Directory Structure

``` text
cmd/
└── cli/
    └── main.go                         # CLI executable entry point

frontend/
└── cli/
    ├── network/                        # TCP connection and protocol I/O
    │   ├── client.go                   # Client lifecycle, authentication, Send, Events, Close
    │   ├── connection.go               # TCP dial, greeting validation, CONNECT handshake
    │   ├── transport.go                # Line-oriented TCP transport and validation
    │   ├── messages.go                 # IncomingMessage (parsed + raw message)
    │   ├── reader.go                   # Background reader pump
    │   ├── writer.go                   # Background writer pump
    │   ├── outbox.go                   # Bounded FIFO queue for outgoing messages
    │   └── requests.go                 # Serializes request/response operations
    │
    └── ui/                             # Bubble Tea application and presentation
        ├── authentication.go           # Username entry and authentication flow
        ├── model.go                    # Main Model and application/player/world/UI state
        ├── update.go                   # Bubble Tea Update loop and message handling
        ├── view.go                     # Main View and screen composition
        ├── message.go                  # UI message types (events, auth result, client shutdown)
        ├── events.go                   # Asynchronous server-event handling
        ├── command.go                  # Command input and local command routing
        ├── command_result.go           # Command response interpretation
        ├── command_handlers_combat.go  # Combat command result/state handling
        ├── command_handlers_player.go  # Player, inventory, status and quest result handling
        ├── command_handlers_world.go   # Room, movement and world result handling
        ├── data.go                     # Client-side data structures
        ├── room.go                     # Room data helpers
        ├── chat.go                     # Chat state and message rendering
        ├── available_commands.go       # Context-sensitive command hints
        ├── transition.go               # View transition state and behavior
        ├── view_world.go               # World/map view
        ├── view_character.go           # Character/status view
        ├── view_inventory.go           # Inventory view
        ├── view_quests.go               # Quest view
        ├── view_group.go                # Group view
        ├── view_combat.go               # Combat view
        ├── view_talk.go                 # NPC conversation view
        │
        ├── animations/                 # Small UI animation helpers
        │   ├── cursor.go
        │   ├── damage.go
        │   ├── health.go
        │   ├── loot.go
        │   ├── notification.go
        │   └── quest.go
        │
        └── layout/                     # Terminal layout, styles and reusable components
            ├── main.go
            ├── layout_manager.go
            ├── layout_requirements.go
            ├── layout_types.go
            ├── measure.go
            ├── components.go
            ├── header.go
            ├── sidebar.go
            ├── ascii_art.go
            ├── theme.go
            └── separador.go
```

The tree above focuses on the CLI's main source files. It omits tests
and unrelated repository files.

---

## 4. Network Layer

The `frontend/cli/network` package isolates TCP communication from the
UI.

### 4.1 Connection and authentication

`connection.go` is responsible for establishing the TCP connection and
completing the initial protocol handshake:

- `Dial` opens the TCP connection.
- The client reads and validates the server greeting.
- The supported greeting is `OK hello proto=1`.
- `Authenticate` sends `CONNECT <username>` and validates the
    response.
- Handshake operations use a timeout so the client does not wait
    indefinitely.

`client.go` owns the higher-level client lifecycle. It tracks connection
state, starts the reader and writer after successful authentication,
exposes `Send` and `Events`, and coordinates shutdown.

### 4.2 Transport and message framing

`transport.go` provides line-oriented TCP I/O. TAP messages are UTF-8
text lines terminated by a newline. The transport validates outgoing and
incoming lines and applies write deadlines and message-size limits.

The transport is deliberately separate from the UI: it knows how to
exchange lines, not how to render rooms or combat.

### 4.3 Reader and writer pumps

After authentication, the client starts two background components:

- **Reader (`reader.go`):** continuously reads server lines, parses
    them as TAP messages, and routes responses and events to their
    respective channels.
- **Writer (`writer.go`):** is the single component responsible for
    writing queued messages to the TCP connection.
- **Outbox (`outbox.go`):** a bounded FIFO queue that coordinates
    concurrent senders with the writer.
- **Request manager (`requests.go`):** ensures only one
    request/response operation is active at a time.

This serialization is important because RFC 42TAP messages do not
contain request IDs. If multiple commands were sent concurrently, the
client could not reliably determine which `OK` or `ERR` belongs to which
request.

### 4.4 Response and event separation

The client distinguishes between:

- **Responses:** `OK` and `ERR` messages associated with a command
    request.
- **Events:** asynchronous `EVT` messages that may arrive
    independently of a command.

`network.IncomingMessage` keeps both the parsed `protocol.Message` and
the original raw line. The UI consumes asynchronous messages through
`Client.Events()`.

---

## 5. UI Architecture - Bubble Tea

The UI is implemented using Bubble Tea's Model--Update--View cycle.

| Part | Responsibility |
|---|---|
| `Model` | Holds the current application, navigation, player, world, communication, and interface state. |
| `Update` | Receives keyboard input, window-size changes, command results, server events, and animation ticks; updates the model and schedules follow-up commands. |
| `View` | Renders the current model as terminal text. It does not own networking or authoritative game logic. |

---

### 5.1 State organization

`model.go` separates state into focused structures:

- `ApplicationState` - authentication or active game state.
- `NavigationState` - currently selected view.
- `PlayerState` - username, status, inventory, and quests.
- `WorldState` - current room, group, conversation, combat, and
    online player count.
- `CommunicationState` - status text, event log, chat log, and
    active chat filter.
- `InterfaceState` - terminal dimensions, viewports, input, focus,
    pending refresh, transitions, and animation state.

This organization keeps the top-level `Model` readable and groups values
by purpose.

### 5.2 Views

The UI provides dedicated views for:

- World map
- Current room
- Inventory
- Character status
- Quests
- Group
- Combat
- NPC conversation

The main `View` chooses between the authentication screen and the game
layout. The game layout combines the main viewport, header, and sidebar.
Individual view files render the relevant domain data.

### 5.3 Layout and animations

`ui/layout` contains reusable terminal components, styles, measurements,
and layout calculations. It adapts the main view and sidebar to the
available terminal dimensions.

`ui/animations` contains focused animation helpers for cursor blinking,
health bars, damage, loot, notifications, and quest feedback. These
helpers support presentation only; they do not determine game outcomes.

---

## 6. Command and Event Flow

### 6.1 Sending a command

``` text
User input
    |
    v
Bubble Tea Update
    |
    v
CLI command handling
    |
    v
network.Client.Send
    |
    v
RequestManager (one in-flight request)
    |
    v
Outbox -> Writer -> TCP
    |
    v
TAP Server
```

When the server replies, the reader routes the `OK` / `ERR` response to
the active request. The UI interprets the result and updates its local
display state.

### 6.2 Receiving an asynchronous event

``` text
TAP Server
    |
    v
TCP -> Reader -> protocol parser
                    |
                    v
              Client.Events()
                    |
                    v
             Bubble Tea message
                    |
                    v
               Update()
                    |
                    v
             Updated Model
                    |
                    v
                 View()
```

Events such as room presence, chat, group changes, combat turns, damage,
NPC death, and respawn can arrive while the user is navigating another
view. The UI handles them through the normal Bubble Tea message loop
rather than blocking the terminal input loop.

### 6.3 Server authority

The CLI may cache or project data needed for rendering, but it does not
decide whether a move, item action, quest transition, or combat action
is valid. It sends a command and reflects the server's response and
subsequent events.

---

## 7. Local Commands vs. TAP Commands

The CLI accepts the normal TAP command syntax and forwards game commands
to the server. It also provides a small number of local interface
conveniences.

- TAP commands such as `LOOK`, `MOVE`, `TAKE`, `CHAT`, `ATTACK`, `STATUS`, and `GROUP` are server requests.
- `MAP` is a local view shortcut; it changes the terminal view and is not a TAP protocol command.
- Keyboard controls such as `Tab`, arrow keys, and `<` control navigation, scrolling, chat filters, or panel focus. They are not sent to the server.

This separation keeps the CLI useful for direct protocol interaction
while still providing a navigable terminal interface.

---

## 8. Error Handling and Shutdown

Network errors are propagated from the transport and reader/writer pumps
to the client lifecycle. The client coordinates shutdown through its
context and close-once mechanism, closes the request manager and outbox,
and closes the TCP connection.

At the UI level, authentication failures and command errors are
converted into user-visible status messages or event-log entries. The
CLI should not silently treat an `ERR` response as a successful state
change.

`cmd/cli/main.go` defers `client.Close()` so the network client is
closed when the Bubble Tea program exits.

---

## 9. Design Decisions

| Decision | Reason |
|---|---|
| Bubble Tea Model-Update-View | Separates application state, input/event processing, and rendering. |
| Dedicated network package | Keeps TCP and protocol concerns out of the UI code. |
| Reader and writer pumps | Allows incoming events to be processed continuously and gives each connection a single writer. |
| Bounded outgoing queue | Provides controlled buffering between command producers and the TCP writer. |
| One in-flight request | Required because TAP responses do not carry request correlation IDs. |
| Local read-only game projection | Supports responsive rendering without making the client authoritative. |
| Direct TAP command syntax | Makes the client transparent for protocol testing and compatible with other TAP servers. |

---


## 10. Related Documentation

- [`CLI_USER_GUIDE.md`](../CLI_USER_GUIDE.md) - player-facing CLI
    usage and keyboard controls.
- [`gui-architecture.md`](gui-architecture.md) - architecture of the
    desktop GUI client.
- [`server-architecture.md`](server-architecture.md) - server
    architecture and concurrency model.
- [`protocol-rfc.html`](../protocol-rfc.html) - TAP wire protocol
    specification.
