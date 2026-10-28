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
- [Server Logging](#project-management)
- [Group Contributions](#group-contributions)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Resources](#resources)

## Description

A TCP-based multiplayer text adventure protocol and game server written in Go.

The Answer Protocol (TAP) is an application-level protocol designed for multiplayer text-based adventures. It defines how clients connect to a server, authenticate players, send commands, receive responses, and process asynchronous events.

The project currently includes a TCP game server, a command-line client, a command dispatching system, session management, and an initial world representation.

The goal is to build a multiplayer adventure with a shared world, room-based interactions, inventory management, combat, quests, and multiple client interfaces.

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
make run-cli
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
make run-gui
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
| `internal/protocol` | Parses incoming protocol messages and constructs outgoing responses and events. |
| `internal/commands` | Registers commands and dispatches requests to their handlers. |
| `internal/server` | Manages TCP connections, sessions, input/output pumps, and connected players. |
| `internal/world` | Loads and represents the game world. |
| `data/` | Contains declarative world data. |

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
```

This diagram represents the logical flow. The actual implementation also uses separate goroutines for connection reading and writing.

## Design Patterns

### 1. Command Pattern

TAP uses a command-oriented architecture to separate command dispatching from command execution.

The `Dispatcher` acts as the command registry and routing component. Each registered command has its own handler and can be developed independently of the network connection lifecycle.

Conceptually, the flow is:

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
     +------ QUIT ------> Quit Handler
     |
     +------ MOVE ------> Move Handler (planned)
```

The server does not need a separate network-handling branch for every game command. Instead, it delegates recognized commands to the corresponding handler.

This separation provides several benefits:

- New commands can be added without rewriting the main connection loop.
- Command execution can be tested independently of TCP transport.
- Network handling and game rules remain separate concerns.
- Unknown commands can be handled consistently.
- Command-specific logic is easier to maintain.

The command handlers receive a context containing the information needed to execute a command, such as the player's identity and the relevant world state.

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
- Broadcasting messages to registered sessions.

The registry is a session-management component, not the game world's source of truth.

For example, knowing that two players are connected does not automatically mean that the server maintains correct room membership or that both players receive only the events relevant to their current room.

Those responsibilities will need to be implemented as the world model and movement system evolve.

### 4. Observer and Publish-Subscribe:

```
             COMANDO MOVE
                  |
                  v
             World / lógica
             del movimiento
                  |
                  v
          Evento interno publicado
                  |
                  v
              EVENT BUS
                  |
                  v
       COMPONENTE DE NOTIFICACIONES
       ¿Quién debe recibir el evento?
                  |
          +-------+-------+
          |       |       |
          v       v       v
       Outbox  Outbox  Outbox
       Alice    Bob    Charlie
          |       |       |
          v       v       v
       TCP      TCP      TCP



                  MoveCommand
                       |
                       v
                 World.MovePlayer
                       |
             Movimiento confirmado
                       |
                       v
              Evento PlayerMoved
                       |
                       v
                   Event Bus
                       |
                       v
           Notification Dispatcher
                 /           \
                v             v
         Jugadores de     Jugadores de
          sala anterior    sala nueva
                \             /
                 v           v
                  Session.Outbox
                       |
                       v
                    WritePump
                       |
                       v
                      TCP


Jugador ejecuta MOVE
        │
        ▼
MoveCommand
        │
        ▼
World.MovePlayer()
  ├── Mueve al jugador
  └── Devuelve PlayerMoved con las listas de ocupantes
        │
        ▼
Event Bus
        │
        ▼
Notification Dispatcher
  ├── Notifica la salida a la sala anterior
  └── Notifica la entrada a la nueva sala
        │
        ▼
Registry → Outbox → WritePump → TCP
```

## Protocol Implementation

TAP defines three primary message categories.

| Prefix | Purpose | Example |
|---|---|---|
| `OK` | Successful response | `OK connected` |
| `ERR` | Error response | `ERR 201 NAME_IN_USE` |
| `EVT` | Asynchronous event | `EVT PLAYER_ENTER alice` |

Messages are UTF-8 text lines exchanged over TCP.

The `internal/protocol` package is responsible for interpreting requests and constructing protocol responses and events.

The protocol specification is the authority for message syntax, command semantics, and error codes. Implementation details should remain consistent with that specification.

### TAP Errors

## 8.2 Standard Error Codes

The server uses standard error codes to communicate protocol errors, gameplay restrictions, and operational failures.

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

### Error Handling Principles

- **Gameplay restrictions** should return specific error codes whenever available. For example, attempting to drop a quest item should return `ITEM_CANNOT_BE_DROPPED`, not `COMMAND_FAILED`.
- **Invalid requests** should be distinguished from recognized commands that have invalid or missing arguments.
- **Unexpected execution failures** should use `COMMAND_FAILED` or `INTERNAL_ERROR`, depending on where the failure occurs.
- **Connection and transmission failures** should use the dedicated connection error codes.
- Error responses must not expose internal implementation details or sensitive server information.

## World Design

The server loads the initial world from `data/world.json`.

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