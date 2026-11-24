# CLI User Guide

## 1. Introduction

Welcome to the CLI client of **The Answer Protocol**, a medieval multiplayer text-based adventure game.

The CLI allows players to explore the world of Arvendale, interact with NPCs, collect items, complete quests, form groups with other players, and participate in combat.

The interface is organized into several views, with a main content area, a sidebar, an event log, a chat panel, and an input area for commands.

## 2. Getting Started

### Requirements

- Go 1.27+ or later.
- A running instance of the game server.
- A terminal that supports ANSI colors and Unicode characters.

### Starting the Client

From the project directory, run:

```bash
make run-client / ./bin/tap-cli
```

If the project provides a compiled executable, you can also start the client directly:

```bash
./cli
```

### Authentication

When the client starts, enter your player username to connect to the server.

After successful authentication, the game interface will appear and display the world map.

## 3. Interface Overview

The CLI interface is divided into several areas.

### Main View

The main view displays the currently selected game screen.

Available views include:

| View | Description |
|---|---|
| World | Displays the world map and the player's current location. |
| Room | Displays the current room, players, NPCs, objects, and exits. |
| Inventory | Displays the items carried by the player. |
| Character | Displays the player's name and health status. |
| Quests | Displays available and active quest information. |
| Group | Displays group membership and pending invitations. |
| Combat | Displays information and actions related to combat. |
| Talk | Displays the current NPC conversation. |

### Sidebar

The sidebar provides quick access to quest and group information.

### Event Log

The event log displays server events and important game notifications.

### Chat

The chat panel displays messages exchanged between players.

Chat messages can belong to different channels:

- Global: messages visible to players across the world.
- Room: messages visible to players in the current room.
- Group: messages visible to members of the player's group.

### Available Commands

The available commands area displays commands that can currently be used in the game.

The displayed commands depend on the player's situation, available interactions, and combat state.

### Input Area

Enter commands in the input area and press `Enter` to execute them.

## 4. Keyboard Controls

| Key | Action |
|---|---|
| `Enter` | Execute the entered command. |
| `Tab` | Cycle through chat filters. |
| `Up` / `Down` | Scroll the focused viewport. |
| `Left` / `Right` | Navigate the viewport when supported. |
| `<` | Switch focus between the main view and sidebar. |
| `Esc` | Return to the world view. |

The interface also supports animated transitions between views.

Enter `CHAT` to open the full chat history. Use `Tab` to cycle through all,
group, room chat, and global messages; use the viewport controls to scroll.

## 5. Available Commands

Commands are entered in the input area and executed by pressing `Enter`.

### General Commands

| Command | Description |
|---|---|
| `LOOK` | Look around the current room. |
| `MAP` | Display the world map. |
| `WHO` | Display the number of connected players. |
| `STATUS` | Display the character's health status. |
| `QUIT` | Disconnect from the server. |

### Movement and Exploration

| Command | Description |
|---|---|
| `MOVE <direction>` | Move in the specified direction. |
| `LOOK` | Inspect the current room. |

Available directions depend on the exits of the current room.

### Inventory

| Command | Description |
|---|---|
| `INVENTORY` | Display the player's inventory. |
| `TAKE <item>` | Pick up an item from the current room. |
| `DROP <item>` | Drop an item from the inventory. |
| `USE <item>` | Use a consumable item. |

Only supported item types can be dropped or used.

### NPC Interaction and Quests

| Command | Description |
|---|---|
| `TALK <npc>` | Start a conversation with an NPC. |
| `QUEST <npc>` | Request or accept a quest from an NPC. |
| `QUESTS` | Display the player's quests. |

Quest information includes its status, description, progress, and quest giver when available.

### Groups

| Command | Description |
|---|---|
| `GROUP CREATE <name>` | Create a new group. |
| `GROUP INVITE <player>` | Invite another player. |
| `GROUP JOIN <name>` | Join a group. |
| `GROUP LEAVE` | Leave the current group. |
| `GROUP INFO` | Display information about the current group. |

Group functionality allows players to cooperate and explore the world together.

### Chat

Use the following syntax to send messages:

```text
CHAT GLOBAL <message>
CHAT ROOM <message>
CHAT GROUP <message>
```

Entering `CHAT` without a scope or message opens the chat history.

Each channel has a different audience.

### Combat

Combat commands are available when the player participates in an active combat encounter.

| Command | Description |
|---|---|
| `ATTACK <target>` | Start combat against an NPC. |
| `COMBAT ATTACK` | Attack during the combat turn. |
| `COMBAT DEFEND` | Defend during the combat turn. |
| `COMBAT FLEE` | Attempt to escape combat. |
| `COMBAT USE` | Use an item during combat. |
| `COMBAT HELP` | Request help during combat. |
| `COMBAT JOIN` | Join an ongoing combat. |

Combat commands depend on the current combat state and server rules.

## 6. Views and Navigation

The CLI provides dedicated views for different aspects of the game.

### World Map

The world map displays the known locations in Arvendale.

The player's current location is highlighted in gold, while other locations use muted borders.

Use `MAP` to return to the world map.

### Room

The room view displays:

- Room name and description.
- Other players in the room.
- NPCs and their interaction status.
- Available objects.
- Available exits.

### Inventory

The inventory view organizes items by category and displays their quantities.

### Character

The character view displays the player's username, current health, maximum health, and health condition.

### Quests

The quests view provides detailed information about the player's quests, including their current status and progress.

### Group

The group view displays the group name, identifier, members, and pending invitations.

### Combat

The combat view displays information relevant to the current encounter and available combat actions.

### Talk

The talk view displays dialogue exchanged with the current NPC.

## 7. Chat Filters

The chat panel supports filtering messages by channel.

Press `Tab` to cycle through the available filters:

- All messages.
- Global messages.
- Room messages.
- Group messages.

This makes it easier to follow conversations without mixing messages from different channels.

## 8. Tips for Players

- Use `MAP` to check your location in Arvendale.
- Use `LOOK` to inspect your surroundings before moving.
- Check available exits before using `MOVE`.
- Use `INVENTORY` to review your collected items.
- Talk to NPCs to discover quests and interactions.
- Use `QUESTS` to track your progress.
- Form a group to cooperate with other players.
- Pay attention to the event log for important server notifications.
- Check your health before entering combat.

## 9. Troubleshooting

### Unable to connect

- Verify that the game server is running.
- Check the server address and connection configuration.
- Verify that your network connection is available.

### Commands are not working

- Check the command syntax.
- Verify that the command is available in your current situation.
- Check the status message and event log for server feedback.

### The interface does not display correctly

- Use a terminal with ANSI color support.
- Ensure that the terminal window is large enough.
- Use a Unicode-compatible terminal font.

---

**The Answer Protocol — CLI User Guide**

Explore Arvendale, interact with its inhabitants, complete quests, and adventure with other players.