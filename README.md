*This project has been created as part of the 42 curriculum by oshtohri, aunoguei.*

# The Answer Protocol (TAP)

## Table of Contents

- [Description](#description)
- [Instructions](#instructions)
- [Architecture](#architecture)
- [Design Patterns](#Design-Patterns)
- [Protocol Implementation](#protocol-implementation)
- [Combat System](#combat-system)
- [Quest System](#implementation)
- [World Design](#World-Design)
- [Server Logging](#Server-Logging)
- [Group Contributions](#group-contributions)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Resources](#resources)

## Description

The Answer Protocol (TAP) is a TCP-based multiplayer text adventure protocol and game server written in Go.

TAP defines an application-level communication protocol for multiplayer text-based adventures. It specifies how clients connect to a server, authenticate players, send commands, receive responses, and process asynchronous events.

The project includes:

- A TCP game server.
- A command-line client.
- A desktop GUI client.
- A command registration and dispatching system.
- Session management and connection handling.
- A shared game world with rooms and exits.
- Item definitions, item instances, and player inventories.
- NPC interactions and dialogue.
- A quest system with player-specific progress.
- Player groups and invitation management.
- Global, room, and group chat.
- Asynchronous notifications for relevant game events.

The goal is to provide a multiplayer adventure environment in which players can explore a shared world, interact with other players and NPCs, manage items, complete quests, and communicate with each other.

## Instructions

### Requirements

- Go installed and configured.
- The world data file available at `data/world.json`.

### Start the server

Run the server using the repository's configured build or run target.

```bash
make run-server
```

The server loads the world and listens on its configured TCP port.

### Start the CLI client

Run the CLI client using the repository's configured build or run target.

```bash
make run-client
```

The client connects to the configured server address and port.

Authenticate using:

```text
CONNECT alice
```

Then use a command like `look` to inspect the current room:

```text
LOOK
```

To terminate the session:

```text
QUIT
```

### Start the GUI client

Run the CLI client using the repository's configured build or run target.

```bash
make run-client-gui
```

The client connects to the configured server address and port.

## Architecture

TAP separates network communication, protocol handling, command execution, session management, and game logic.

The main components have distinct responsibilities:

| Component | Responsibility |
|---|---|
| `cmd/server` | Starts the TCP server and initializes application dependencies. |
| `cmd/cli` | Runs the terminal client and handles user input. |
| `cmd/desktop` | Runs the GUI client and handles user input. |
| `internal/protocol` | Defines TAP messages, error codes, and response encoding.|
| `internal/commands` | Registers commands and dispatches requests to their handlers. |
| `internal/server` | Manages TCP connections, sessions, input/output pumps, notifications, and connected players. |
| `internal/world` | Maintains the game world, player state, items, NPCs, quests, and groups. |
| `data/` | Contains the declarative world data. |

### High-level architecture

```text
                    TAP CLIENTS
                         |
                         | TCP
                         v
                  +--------------+
                  |  TCP Server  |
                  +--------------+
                         |
                         v
                  +--------------+
                  |   Session    |
                  |  Management  |
                  +--------------+
                         |
                         v
                  +--------------+
                  |   Protocol   |
                  |    Parser    |
                  +--------------+
                         |
                         v
                  +--------------+
                  |  Dispatcher  |
                  +--------------+
                         |
              +----------+----------+
              |                     |
              v                     v
         LOOK Command          QUIT Command
              |                     |
              v                     v
          World Data          Session Cleanup

                       TAP CLIENTS
                            |
                            | TCP
                            v
                     +--------------+
                     |  TCP Server  |
                     +--------------+
                            |
                            v
                     +--------------+
                     |   Session    |
                     |  Management  |
                     +--------------+
                            |
                            v
                     +--------------+
                     |   Protocol   |
                     |    Handler   |
                     +--------------+
                            |
                            v
                     +--------------+
                     |  Dispatcher  |
                     +--------------+
                            |
                +-----------+-----------+
                |                       |
                v                       v
          Command Handler          Session Cleanup
                |
                v
           World Logic
                |
                v
          Game State Changes
                |
                v
           Internal Events
                |
                v
      Notification Dispatcher
                |
                v
          Session Outboxes
                |
                v
            TCP Clients
```

This diagram represents the logical flow. The implementation also uses separate goroutines for connection reading and writing.

### World State Management

The World component centralizes the shared game state.

It maintains:

- Room definitions and connections between rooms.
- Player positions.
-Item definitions and individual item instances.
-Items currently available in rooms.
-Player inventories.
-NPC definitions and dialogue.
-Quest definitions and player quest progress.
-Player groups and group invitations.

Access to shared world state is protected by a sync.RWMutex, allowing concurrent reads while protecting state modifications.

## Design Patterns

### 1. Command Pattern

TAP uses a command-oriented architecture to separate command dispatching from command execution.

The `Dispatcher` acts as the command registry and routing component. Each registered command has its own handler and can be developed independently of the network connection lifecycle.

The command flow is:

```text
Client Input
     |
     v
Command Parser
     |
     v
Dispatcher
     |
     +------ LOOK ------> Look Handler
     |
     +------ MOVE ------> Move Handler
     |
     +------ TAKE ------> Take Handler
     |
     +------ DROP ------> Drop Handler
     |
     +------ CHAT ------> Chat Handler
     |
     +------ GROUP -----> Group Handler
     |
     +------ QUEST -----> Quest Handler
```

The server does not need a separate network-handling branch for every game command. Instead, it delegates recognized commands to the corresponding handler.

This separation provides several benefits:

- New commands can be added without rewriting the main connection loop.
- Command execution can be tested independently of TCP transport.
- Network handling and game rules remain separate concerns.
- Unknown commands can be handled consistently.
- Command-specific logic is easier to maintain.

The command handlers receive a context containing the information needed to execute a command, such as the player's identity and the relevant world state and event publisher.

### 2. Reader–Writer–Outbox Concurrency Model

Each client session uses separate reading and writing goroutines.

The purpose of this design is to ensure that a single goroutine owns TCP writes for each connection.

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

#### ReadPump

The reader consumes incoming TCP data and forwards complete messages to the connection handler.

It does not need to render UI output or execute terminal-specific behavior.

#### Command Handling

The connection handler processes incoming commands and delegates gameplay operations to the dispatcher.

Command responses and asynchronous notifications are queued through the session's output mechanism.

#### Session.Outbox

Each session owns a buffered output channel.

The outbox decouples message producers from the actual network write operation. Commands and broadcasts can enqueue messages without writing directly to the socket.

#### WritePump

The writer consumes messages from the outbox and writes them to the TCP connection.

**The key invariant is that each session has one TCP writer.**

Without this invariant, multiple goroutines could write concurrently to the same connection, making message ordering and framing harder to reason about.

The outbox also provides a natural place to implement output backpressure and slow-client policies in the future.

### 3. Session Registry

The `Registry` maintains the collection of active sessions.

Its responsibilities include:

- Registering authenticated sessions.
- Preventing duplicate active usernames.
- Removing disconnected sessions.
- Retrieving sessions by player name.
- Broadcasting messages to registered sessions.
- Queuing notifications for individual sessions.

The registry is a session-management component, not the game world's source of truth.

For example, knowing that two players are connected does not automatically determine their room membership. Room membership and other gameplay relationships are maintained by the world model.

### 4. Observer and Publish-Subscribe:

```
Player executes MOVE
        |
        v
   MoveCommand
        |
        v
 World.MovePlayer()
        |
        v
  PlayerMoved Event
        |
        v
     Event Bus
        |
        v
Notification Dispatcher
        |
        +-----------------------+
        |                       |
        v                       v
Previous Room Players     New Room Players
        |                       |
        +-----------+-----------+
                    |
                    v
              Session Registry
                    |
                    v
                session.Outbox
                    |
                    v
                WritePump
                    |
                    v
                   TCP
```
The notification dispatcher handles several event categories:


| Internal event | TAP notification | Recipients|  
|---|---|---|
| PlayerMoved| ROOM_PRESENCE_LEAVE and ROOM_PRESENCE_ENTER | Players in the previous and destination rooms, excluding the moving player.| 
| PlayerEntered | ROOM_PRESENCE_ENTER | Other players in the room.|
| PlayerLeft | ROOM_PRESENCE_LEAVE | Other players in the room.| 
| ItemDropped | ITEM_DROPPED | Other players in the room.| 
| UniqueItemsReturned | UNIQUE_ITEMS_RETURNED | All registered sessions.| 
| GroupInvite | GROUP INVITE | The invited player.| 
| GroupJoin | GROUP JOIN | Other members of the group.| 
| GroupLeave | GROUP LEAVE | Remaining members of the group.|
| ChatMessage | Scope-specific CHAT event | All recipients selected by the chat scope.| 

The dispatcher uses the session registry to locate recipients and queues notifications through their outboxes. It does not write directly to client TCP connections.

## Protocol Implementation

The protocol specification is the authority for message syntax, command semantics, and error codes. 

TAP messages are UTF-8 text lines exchanged over TCP.

Each outgoing message is encoded as a line terminated by a line feed (\n). Embedded carriage returns and line feeds are rejected in protocol fields.

Usernames may contain 1-24 Unicode letters or numbers, combining marks, and underscores. Whitespace and punctuation are not allowed.

The protocol defines three primary message categories:

| Prefix | Purpose | Example |
|---|---|---|
| `OK` | Successful response | `OK connected` |
| `ERR` | Error response | `ERR 201 NAME_IN_USE` |
| `EVT` | Asynchronous event | `EVT PLAYER_ENTER alice` |

### Message Formats

Successful response:
```
OK [payload]
```
A successful response can contain an optional payload.

Error response:
```
ERR <code> <name>
```
Error codes are numeric three-digit values.

Asynchronous event:
```
EVT <event-type> [data]
```
Events are generated by server-side game operations and sent to the relevant connected clients.

### Commands Reference

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

#### Group command reference

| Subcommand | Syntax | Behaviour | Successful response |
|---|---|---|---|
| `CREATE` | `GROUP CREATE <name>` | Creates a group with the supplied name. The name is required by this implementation. | `OK group=<id> name=<name>` |
| `INVITE` | `GROUP INVITE <player>` | Invites a registered player. |`OK` |
| `INVITATIONS` | `GROUP INVITATIONS` | Lists pending group invitations. | `OK <JSON>` |
| `JOIN` | `GROUP JOIN <name>` | Joins a group using a pending invitation. | `OK group=<id>` |
| `LEAVE` | `GROUP LEAVE` | Leaves the current group. | `OK` |
| `INFO` | `GROUP INFO` | Retrieves the current group information. | `OK <JSON>`  |

### RFC deviations and implementation decisions

The following response shapes and command semantics intentionally extend or differ from the RFC examples. Clients should use the contracts below rather than assuming the illustrative RFC payloads are exhaustive.

#### LOOK and inventory payloads

The RFC describes `LOOK`'s `items` and `npcs` as arrays of IDs. This implementation returns item definitions and NPC display names instead. `room.items` also contains item counts. For example:

```json
{
  "room": {
    "id": "harbor_square",
    "name": "Harbor Square",
    "description": "A busy square.",
    "exits": {"north": "market_street"},
    "items": {"item.herbs": 2}
  },
  "players": ["alice"],
  "items": [
    {"id": "item.herbs", "name": "Healing Herbs", "description": "...", "type": "common"}
  ],
  "npcs": ["Herbalist"]
}
```

`INVENTORY` returns one object per item ID with an aggregated `quantity`, rather than one ID per individual instance:

```json
[{"id": "item.herbs", "quantity": 2}]
```

This keeps room and inventory responses useful to clients that display item names/details and stack quantities, while the world still tracks individual item instances internally. Inventory array order is not guaranteed.

#### NPC conversations and quest payloads

The RFC's `TALK` example returns dialogue text. This server returns a JSON conversation object containing the NPC summary, dialogue lines, and optional quest information:

```json
{
  "npc": {"id": "npc.herbalist", "name": "Herbalist", "role": "merchant", "hostile": false, "combat_stats": {"hp": 100, "max_hp": 100, "attack": 10, "defense": 5, "speed": 10}},
  "dialogue": ["Welcome!"],
  "quest": {"id": "quest.herbs", "name": "Gather Herbs", "description": "Bring me herbs"},
  "quest_status": "available"
}
```

When `QUEST <npc>` finds an available quest, it accepts that quest immediately and returns its summary (`id`, `name`, `description`, and optional `progress`); the response does not include a `status` field. This combines requesting and accepting a quest into one interaction; the RFC does not define a separate acceptance command.

`QUESTS` returns a JSON array of active and completed quests. Each entry uses `id`, `name`, `description`, `status`, and optional `progress`. Available quests are discovered by talking to their giver and are not included in this listing until accepted.

Quest progression is per player. Talking to a quest giver discovers an available quest; quest chains unlock their next quest after completion. For active quests with required items, talking to the giver again completes the quest once the required quantity is in the player's inventory. The server consumes required items only when the quest definition requests it, then grants the configured rewards. This automatic completion avoids requiring a separate completion command.

#### Group creation

The RFC writes the syntax as `GROUP CREATE` without specifying a group name. This implementation requires `GROUP CREATE <name>` so the newly created group can be identified and displayed. The name may contain spaces.

#### Combat and quest design choices

The RFC leaves combat and quest mechanics to implementations and requires those choices to be documented.

Combat is turn-based. Effective speed determines the first participant in a round and grants speed points; each combat action costs 10 points. Effective attack minus effective defense determines damage, with a minimum of 1 damage. NPC turns are processed automatically, and a player turn can expire so one inactive player does not stall the encounter. Players can attack, defend, use eligible consumables, request help, join an active combat, or flee. Defending grants a 50% defense bonus; fleeing incurs a health penalty but cannot reduce the player below 1 HP. These rules were chosen to keep actions deterministic and combat moving without requiring real-time client coordination.

Quests use `available`, `active`, `completed`, and `locked` states. Definitions can require item quantities, optionally consume those items, grant item rewards, and link to a next quest. Quest chains are validated against cycles. Acceptance is explicit through the `QUEST` command, while completion is automatic when the active quest's requirements are met and the player next interacts with its giver. This keeps quest state player-specific and makes the configured quest data control progression and rewards.

#### Chat Command Reference

```
CHAT <scope> <message>
```

TAP supports three chat scopes:

| Scope | Recipients |  
|---|---|  
| GLOBAL | All registered players in the world. |  
| ROOM | Players currently occupying the sender's room. |  
| GROUP | Members of the sender's group. |  

Examples:

```
CHAT GLOBAL Hello everyone!
CHAT ROOM Hello, everyone in this room!
CHAT GROUP Let's meet up!
```

Chat messages are delivered asynchronously to the selected recipients, including the sender.

The original message content is preserved after the scope separator, allowing messages to contain spaces.

### Asynchronous Events

The server sends asynchronous events to inform clients about relevant world and social changes.

Examples include:

**Room presence:**

```
EVT ROOM_PRESENCE_ENTER alice
EVT ROOM_PRESENCE_LEAVE alice
```

**Dropped items:**

```
EVT ITEM_DROPPED iron_sword
```

**Unique items returned to a room:**

```
EVT UNIQUE_ITEMS_RETURNED instance_1,instance_2
```

**Group invitations:**

```
EVT GROUP INVITE player=alice group=Adventurers
```

**Group membership changes:**

```
EVT GROUP JOIN player=bob
EVT GROUP LEAVE player=bob
```

**Chat:**

```
EVT GLOBAL CHAT player=alice message=Hello everyone!
EVT ROOM CHAT player=alice message=Hello, room!
EVT GROUP CHAT player=alice message=Let's meet up!
```

Notifications are queued through the recipient sessions' outboxes. The dispatcher excludes the acting player from room presence, item-drop, and group membership notifications where appropriate. Chat notifications are sent to all selected recipients, including the sender.

### TAP Error Codes

The server uses error codes to communicate protocol errors, gameplay restrictions, and operational failures. It preserves the RFC's distinction between non-fatal `4xx` errors and system-level `9xx` errors, while defining more specific codes and symbolic names where that improves client handling.

Each error response contains a numeric code and a symbolic error name. Clients should use these values to identify the error condition rather than relying on human-readable descriptions.

| Code | Message | Description |
|---:|---|---|
| `201` | `NAME_IN_USE` | The requested username is already taken. |
| `301` | `NO_EXIT` | No exit is available in the requested direction from the current room. |
| `400` | `INVALID_REQUEST` | The request is empty or malformed. |
| `401` | `NOT_IN_GROUP` | The requested operation requires group membership. |
| `402` | `ALREADY_IN_GROUP` | The player already belongs to a group. |
| `403` | `CONNECT_REQUIRED` | A command other than `CONNECT` was received before authentication. |
| `404` | `ITEM_NOT_FOUND` | The requested item does not exist or cannot be identified.|
| `405` | `NPC_NOT_HOSTILE` | The NPC cannot be attacked because it is not hostile. |
| `406` | `NO_QUEST_AVAILABLE` | No quest is currently available from the requested source. |
| `407` | `INVALID_USERNAME` | The username does not meet the protocol's requirements. |
| `408` | `UNKNOWN_COMMAND` | The requested command is not recognized by the server. |
| `409` | `COMMAND_FAILED` | The command could not be completed because of an unexpected execution failure. |
| `410` | `ITEM_NOT_IN_ROOM` | The requested item is not present in the player's current room. |
| `411` | `ITEM_NOT_IN_INVENTORY` | The requested item is not present in the player's inventory. |
| `412` | `ITEM_CANNOT_BE_DROPPED` | The item cannot be dropped because of its gameplay rules. |
| `413` | `ITEM_ALREADY_OWNED` | The player already owns an instance of the requested item. |
| `414` | `NPC_NOT_FOUND` | The requested NPC does not exist or is not present in the current context. |
| `422` | `INVALID_ARGUMENTS` | The command is recognized, but its arguments are invalid or incomplete. |
| `500` | `INTERNAL_ERROR` | An unexpected internal server error occurred. |
| `900` | `CONNECTION_FAILED` | The connection could not be established or the session could not be registered. |
| `901` | `SEND_FAILED` | The server failed to transmit a message. |

#### Error-code extensions and recovery

The RFC provides representative error codes, but requires that non-fatal `4xx` errors allow the client to continue and says `9xx` system errors may require reconnecting. This implementation extends the examples with more specific `4xx` codes, including `410`–`416` and `422`, so clients can distinguish gameplay and argument failures without treating them as fatal. These are intentional protocol deviations; clients should handle unknown `4xx` codes as recoverable and should not assume that every `4xx` value is listed in the RFC.

The server also uses command-specific symbolic names for some errors, such as `ITEM_REQUIRED`, `AMBIGUOUS_ITEM_NAME`, and `GROUP_ALREADY_EXISTS`. These names refine the error condition while retaining the associated numeric code. A missing NPC consistently returns `414 NPC_NOT_FOUND` from commands that look up an NPC, including `TALK`, `QUEST`, and `ATTACK`.

Clients should follow the RFC's recovery guidance: continue the session after non-fatal `4xx` responses, and consider re-establishing the connection after `9xx` system errors when necessary.

### Error Handling Principles

- **Gameplay restrictions** should return specific error codes whenever available. For example, attempting to drop a quest item should return `ITEM_CANNOT_BE_DROPPED`, not `COMMAND_FAILED`.
- **Invalid requests** should be distinguished from recognized commands that have invalid or missing arguments.
- **Unexpected execution failures** should use `COMMAND_FAILED` or `INTERNAL_ERROR`, depending on where the failure occurs.
- **Connection and transmission failures** should use the dedicated connection error codes.
- Error responses must not expose internal implementation details or sensitive server information.

### Combat System

Combat is implemented and registered. `ATTACK <npc>` starts an encounter with a hostile NPC; `COMBAT` handles actions during an active encounter, and `STATUS` reports player health. The turn model, damage calculation, NPC actions, and escape behavior are documented under [RFC deviations and implementation decisions](#rfc-deviations-and-implementation-decisions).

### Quest System

TAP includes a quest system connected to NPC interactions and player inventory.

Quests are defined in the world model and can include:

- A unique quest ID.
- A name and description.
- A giver NPC.
- A required item.
- An option to consume the required item.
- Reward item IDs.
- An optional next quest, allowing quest chains.

Quest progress is tracked separately for each player.

#### Quest States

|Status|Description|
|---|---|
|available|The quest can be accepted.|
|active|The player has accepted the quest and it is in progress.|
|completed|The player has completed the quest.|
|locked|The quest is not currently available to the player.|

### NPC Dialogue and Quests

NPCs can provide different dialogue depending on the player's quest state.

Dialogue categories include:

- Default dialogue.
- Available quest dialogue.
- Active quest dialogue.
- Completed quest dialogue.
- Locked quest dialogue.

If no dialogue is defined for a particular quest state, the NPC can fall back to its default dialogue.

The TALK command retrieves NPC dialogue and optional quest information.

The QUEST command accepts an available quest directly. The QUESTS command retrieves the player's discovered quests and their statuses.

## World Design

The game world is represented through rooms, items, NPCs, quests, and groups.

The World component maintains the state of these entities and their relationships.

### Rooms

Rooms represent locations in the shared game world.

Each room contains:
- A unique ID.
- A name.
- A description.
- A collection of exits.
- Item counts.
- References to NPCs present in the room.

Exits connect rooms using directional identifiers and destination room IDs.

Players have a current room, which determines their location and the recipients of room-scoped interactions.

### Items and Item Instances

The world distinguishes between item definitions and individual item instances.

An item definition contains:

-A unique ID.
-A name.
-A description.
-An item type.

The supported item types are:

|Type|Description|
|---|---|
|common|A regular item.|
|quest|An item associated with quest mechanics.|
|unique|An item with unique instance behaviour.|

An ItemInstance represents a concrete occurrence of an item definition.

This distinction allows the world to track individual items independently, even when multiple instances share the same item definition.

### Player Inventories

Each player's inventory stores references to individual item instances.

The world tracks which instances are located in rooms and which are owned by players.

The TAKE, DROP, and INVENTORY commands provide the corresponding player interactions.

Items can also have restrictions that prevent them from being dropped.

### NPCs

NPCs are entities located in the game world.

Their data includes:

- A unique ID and name.
- A role.
- A hostility flag.
- Current and maximum HP.
- A damage value.
- An optional associated quest.
- Default and quest-specific dialogue.

NPC interactions are handled through the TALK and QUEST commands.

### Groups

Groups are shared social entities that allow players to organize and communicate.

The world maintains:
- Group IDs and names.
- Group membership.
- Pending invitations.
- A mapping between players and their current groups.

A player can belong to only one group at a time. Group invitations and membership changes generate events that can be delivered to the relevant connected players.

### World Initialization

The server loads the initial world from data/world.json.

The world model is responsible for maintaining the runtime state after initialization, including player positions, item instances, inventories, NPC interactions, quest progress, and group membership.

## Server Logging

The server uses Go's standard logging facilities to report operational issues.

The notification dispatcher logs failures when it cannot queue asynchronous notifications for a recipient session.

For example, a failed notification queue operation is reported with the affected player name and the underlying error.

Logging helps identify problems in asynchronous message delivery without requiring notification handlers to write directly to client connections.

## Abuse Controls

The server's `internal/abuse` package applies connection admission and request-rate policies independently of game rules. By default, it allows up to 100 active connections globally, 5 simultaneous connections per IP address, 10 connection attempts per IP per minute, and 20 unauthenticated connections globally. Authenticated connections no longer count against the unauthenticated limit.

The limits can be configured with server flags or environment variables:

| Flag | Environment variable | Default |
|---|---|---:|
| `-max-connections` | `TAP_MAX_CONNECTIONS` | 100 |
| `-max-connections-per-ip` | `TAP_MAX_CONNECTIONS_PER_IP` | 5 |
| `-max-connection-attempts-per-ip` | `TAP_MAX_CONNECTION_ATTEMPTS_PER_IP` | 10 per minute |
| `-max-unauthenticated-connections` | `TAP_MAX_UNAUTHENTICATED_CONNECTIONS` | 20 |

Per authenticated player, the server allows up to 30 commands and 5 `CHAT` commands in each rolling 10-second window. A rate-limited command receives `ERR 429 RATE_LIMITED`; this is an implementation extension because RFC 42TAP does not define a rate-limit error code. Rejected connections are closed, and expired per-IP and rate-limit records are periodically removed.

## Group Contributions

---  

---  


```
the-answer-protocol/
│
├── cmd/                          # Executable entry points
│   ├── server/
│   │   └── main.go               # Launches the central TCP game server
│   ├── cli/
│   │   └── main.go               # Launches the classic CLI terminal client
│   └── desktop/
│       └── main.go               # Initializes the Wails application and binds the backend
│
├── internal/                     # Private application code
│   │
│   ├── server/
│   │   ├── server.go             # TCP server life cycle management
│   │   ├── connection.go         # Low-level network connection handling
│   │   └── session.go            # Connected player state and session management
│   │
│   ├── desktop/                  # Go-to-JavaScript native bridge
│   │   └── app.go                # Go methods bound and exposed directly to frontend JS
│   │
│   ├── protocol/                 # 42TAP RFC specification implementation
│   │   ├── parser.go             # Parses and interprets incoming raw text messages
│   │   ├── encoder.go            # Serializes outgoing responses and events
│   │   ├── messages.go           # Protocol structural message definitions
│   │   └── errors.go             # Specification-specific error codes
│   │
│   ├── commands/                 # Command Design Pattern
│   │   ├── dispatcher.go         # Command registry and routing orchestrator
│   │   ├── connect.go
│   │   ├── look.go
│   │   ├── move.go
│   │   ├── chat.go
│   │   ├── take.go
│   │   ├── drop.go
│   │   ├── inventory.go
│   │   ├── talk.go
│   │   ├── attack.go
│   │   ├── status.go
│   │   ├── quest.go
│   │   ├── quests.go
│   │   ├── who.go
│   │   ├── group.go
│   │   └── quit.go
│   │
│   ├── game/                     # Core business logic and world rules
│   │   ├── world.go              # Global universe instance manager
│   │   ├── room.go               # Room entities, descriptions, and exits layout
│   │   ├── player.go             # Player models, attributes, and health pools
│   │   ├── item.go               # Item blueprints and concrete world instances
│   │   ├── npc.go                # Non-Player Character AI behaviors and enemy stats
│   │   ├── inventory.go          # Container storage and inventory logic
│   │   ├── movement.go           # Map navigation and boundary validations
│   │   ├── combat.go             # Combat calculation systems
│   │   └── quest.go              # Quest parameters and reward distribution
│   │
│   ├── events/                   # Observer Design Pattern / Event Bus
│   │   ├── event.go              # Structural system event models
│   │   ├── bus.go                # Central publish and subscribe mechanism
│   │   └── router.go             # Event routing matrix across dynamic clients
│   │
│   ├── worlddata/                # Static world asset management
│   │   ├── loader.go             # YAML/JSON filesystem asset parser
│   │   └── validate.go           # Graph validation for IDs, paths, and room exits
│   │
│   └── logging/
│       └── logger.go             # Structured JSON streaming application logger
│
├── frontend/                     # Wails web graphical interface resources
│   ├── index.html                # Application layout canvas shell
│   ├── wailsjs/                  # Automatically generated by Wails (Go-to-JS mappings)
│   │   └── go/
│   │       └── desktop/
│   │           └── App.js        # Dynamic JS layer invoking compiled Go code natively
│   ├── src/
│   │   ├── app.js                # Frontend core application lifecycle script
│   │   ├── state.js              # GUI state engine monitoring user actions
│   │   └── ui.js                 # DOM engine rendering visual interface outputs
│   └── styles/
│       └── main.css              # Theme styling sheets for retro text-adventure UI
│
├── data/
│   ├── world.json                # Declarative map rooms, default instances, and static NPCs
│   └── quests.json               # Adventure milestone goals and reward structures
│
├── tests/
│   ├── integration/              # Full server pipeline integration test suites
│   └── protocol/                 # RFC specification compliance test runners
│
├── docs/
│   ├── architecture.md           # Technical blueprint and design pattern rationale
│   └── protocol-decisions.md     # Architecture RFC log tracking core constraints
│
├── wails.json                    # Meta configurations for Wails compiler and windows
├── go.mod                        # Go module boundary descriptor file
├── go.sum                        # Dependencies cryptographical lock tracking file
├── Makefile                      # Standard automation compiler wrapper for 42 environment
├── README.md                     # General repository documentation
└── .gitignore                    # Local environment compiler and cache exclusion filters
```

• cmd/ para los puntos de entrada: En Go, la carpeta cmd/ contiene las aplicaciones principales. Tener ahí dentro server/main.go y cli/main.go es limpio porque separa los arrancadores de la lógica de negocio.
• internal/ para evitar plagios y proteger tu API: En el ecosistema de 42, el código debe ser propio. Guardar tus submódulos dentro de internal/ asegura que el compilador de Go prohíba que cualquier otro paquete o proyecto externo importe tu código directamente. Solo lo puede usar tu propio proyecto.
• Separación de Patrones: Tienes internal/commands/ dedicado al patrón Command y internal/events/ al patrón Observer. Meter esto dentro de un directorio src/ solo añadiría un nivel de anidamiento innecesario que no aporta nada al compilador.



un detalle técnico importante: el estado del juego no está compartido. Si el "Jugador A" se conecta, el servidor lo trata de forma aislada; si el "Jugador B" se conecta en otra terminal, el servidor no sabe que están en el mismo mundo ni pueden hablar entre ellos.
Para solucionar esto y empezar a aplicar el Observer Pattern que tienes planeado en tu arquitectura (internal/events/), el servidor necesita pasar de manejar "conexiones sueltas" a manejar un registro de sesiones activas

tres responsabilidades:

| Componente | Responsabilidad | Patrón |
|---|---|---|
| `Dispatcher` | Seleccionar y ejecutar comandos | Command |
| `Registry` | Mantener las sesiones conectadas | Gestión de sesiones |
| `EventBus` | Publicar eventos y distribuirlos a sus suscriptores | Observer / Publish-Subscribe |


command:
Cliente TCP
    │
    ▼
cmd/server/main.go
    │
    ▼
Dispatcher
    │
    ├── LOOK  ──► LookCommand
    │
    └── QUIT  ──► QuitCommand


```
                 main.go
                    |
                    v
             RegisterCommands()
                    |
                    v
               Dispatcher
                    |
         +----------+----------+
         |          |          |
         v          v          v
       LOOK       MOVE       ATTACK
         |          |          |
         v          v          v
     LookCommand MoveCommand AttackCommand
         |          |          |
         +----------+----------+
                    |
                    v
               Game Context
                    |
                    v
             Shared World
```

TAP, utiliza este diseño:
- Una goroutine de lectura por conexión: recibe los comandos del jugador.
- Una goroutine de escritura por conexión: envía los mensajes pendientes.
- Un canal de salida por sesión: recibe los mensajes que deben enviarse al jugador.
- El Registry: administra las sesiones y distribuye los broadcasts.

Cada conexión tendrá una goroutine para leer y otra para escribir. El canal Outbox será el punto de comunicación entre el resto del servidor y la goroutine de escritura.

                  TCP Connection
                        |
             +----------+----------+
             |                     |
       Read goroutine        Write goroutine
             |                     ^
             |                     |
       Dispatcher             Session.Outbox
             |                     ^
             |                     |
             +------ Registry -----+
                    Broadcast


Las responsabilidades serán:
- handleConnection: gestiona la lectura de comandos y el ciclo de vida de la sesión.
- writePump: es la única goroutine que escribe en la conexión.
- Session.Outbox: mantiene los mensajes pendientes de envío.
- Registry.Broadcast: distribuye los mensajes a las sesiones sin escribir directamente en sus sockets.
- Los comandos: tendrán que utilizar el mismo sistema de salida para evitar escrituras concurrentes.
```
                 TCP Server
                     |
          +----------+----------+
          |                     |
     Alice Session          Bob Session
          |                     |
    +-----+-----+         +-----+-----+
    |           |         |           |
  Reader      Writer    Reader      Writer
    |           ^         |           ^
    |           |         |           |
    |        Outbox      Outbox       |
    |           ^         ^           |
    +----> Registry <-----+           |
               ^                       |
               +-----------------------+
                   Broadcast
```

session:
                    TAP SERVER
                         │
             ┌───────────┴───────────┐
             │       Session         │
             │                       │
 Cliente ── TCP ──► readPump         │
             │          │            │
             │          ▼            │
             │      Dispatcher       │
             │          │            │
             │          ▼            │
             │       Outbox ◄── Eventos
             │          │            │
             │          ▼            │
             │       writePump       │
             └──────────┬────────────┘
                        │
                       TCP
                        │
                     Cliente
El principio más importante es que cada sesión tenga un único escritor de TCP. Así evitamos que dos goroutines escriban simultáneamente mensajes distintos en la misma conexión y mezclen la salida.

observer:

                  ┌─────────────────┐
Cliente ─────────►│    Dispatcher   │
                  └────────┬────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │     Command     │
                  └────────┬────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │     EventBus    │
                  └────────┬────────┘
                           │
                  ┌────────┴────────┐
                  ▼                 ▼
             Registry          Game Events
                  │
                  ▼
          Sesiones suscritas



Cliente                  Servidor
   |                         |
   |------ LOOK ------------>|
   |                         |
   |                  Dispatcher.Dispatch()
   |                         |
   |                  LOOK handler
   |                         |
   |<----- OK + JSON --------|
   |                         |
   | ParseServerMessage()    |
   |                         |
   | json.Unmarshal()        |
   |                         |
   | RoomView                |
   |                         |
   | RenderRoom()            |

## Resources

[GO Tour](https://es-go-dev.shuijingwanwq.com/tour/list)  
[GO Synthax](https://go.dev/blog/declaration-syntax)  
[GO Playground](https://go.dev/play/)  
[GO Examples](https://gobyexample.com/)  
[GO exercises](https://github.com/gophercises)  
[GO tutorial](https://www.w3schools.com/go/)  
[YAML vs JSON](https://medium.com/geekculture/yaml-vs-json-vs-xml-in-go-bf4ebd1066f2)  
[GO & JSON](https://go.dev/blog/json)  
[Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)  
[Advanced Go Concurrency Patterns](https://www.youtube.com/watch?v=QDDwwePbDtw)  


[](https://medium.com/%40eikhapoetra/why-are-many-backend-switching-to-go-golang-abd3c5431997)
[](https://medium.com/%40pickleprat/building-a-tcp-chatroom-in-golang-6f6bd2b11bb7)
[](https://dev.to/shrsv/unpacking-go-channels-a-peek-under-the-hood-2c42?utm_source=chatgpt.com)

[Input Validation](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)