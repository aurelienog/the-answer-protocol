
package ui

type AvailableCommand struct {
    Name        string
    Usage       string
    Description string
}

var availableCommands = []AvailableCommand{
    {
        Name:        "LOOK",
        Usage:       "LOOK",
        Description: "Look around the current room.",
    },
    {
        Name:        "QUIT",
        Usage:       "QUIT",
        Description: "Disconnect from the server.",
    },
    {
        Name:        "MOVE",
        Usage:       "MOVE <direction>",
        Description: "Move in a direction.",
    },
    {
        Name:        "TAKE",
        Usage:       "TAKE <item>",
        Description: "Pick up an item from the room.",
    },
    {
        Name:        "DROP",
        Usage:       "DROP <item>",
        Description: "Drop an item from your inventory.",
    },
    {
        Name:        "INVENTORY",
        Usage:       "INVENTORY",
        Description: "Display your inventory.",
    },
    {
        Name:        "TALK",
        Usage:       "TALK <npc>",
        Description: "Talk to an NPC.",
    },
    {
        Name:        "QUEST",
        Usage:       "QUEST <npc>",
        Description: "Request and accept a quest from an NPC.",
    },
    {
        Name:        "QUESTS",
        Usage:       "QUESTS",
        Description: "Display your quests.",
    },
    {
        Name:        "WHO",
        Usage:       "WHO",
        Description: "Display the number of connected players.",
    },
    {
        Name:        "GROUP",
        Usage:       "GROUP <subcommand>",
        Description: "Manage your group.",
    },
    {
        Name:        "CHAT",
        Usage:       "CHAT <GLOBAL|ROOM|GROUP> <message>",
        Description: "Send a chat message.",
    },
    {
        Name:        "STATUS",
        Usage:       "STATUS",
        Description: "Display your character's health status.",
    },
    {
        Name:        "ATTACK",
        Usage:       "ATTACK <target>",
        Description: "Start combat against an NPC.",
    },
    {
        Name:        "COMBAT",
        Usage:       "COMBAT <subcommand>",
        Description: "Perform actions during combat.",
    },
    {
        Name:        "USE",
        Usage:       "USE <item>",
        Description: "Use a consumable item from your inventory.",
    },
}

var groupCommands = []AvailableCommand{
    {
        Name:        "CREATE",
        Usage:       "GROUP CREATE <name>",
        Description: "Create a group.",
    },
    {
        Name:        "INVITE",
        Usage:       "GROUP INVITE <player>",
        Description: "Invite a player to your group.",
    },
    {
        Name:        "INVITATIONS",
        Usage:       "GROUP INVITATIONS",
        Description: "Display your pending group invitations.",
    },
    {
        Name:        "JOIN",
        Usage:       "GROUP JOIN <name>",
        Description: "Join a group.",
    },
    {
        Name:        "LEAVE",
        Usage:       "GROUP LEAVE",
        Description: "Leave your current group.",
    },
    {
        Name:        "INFO",
        Usage:       "GROUP INFO",
        Description: "Display information about your group.",
    },
}

var combatCommands = []AvailableCommand{
    {
        Name:        "ATTACK",
        Usage:       "COMBAT ATTACK",
        Description: "Attack during your combat turn.",
    },
    {
        Name:        "FLEE",
        Usage:       "COMBAT FLEE",
        Description: "Attempt to escape combat.",
    },
    {
        Name:        "DEFEND",
        Usage:       "COMBAT DEFEND",
        Description: "Defend during your combat turn.",
    },
    {
        Name:        "USE",
        Usage:       "COMBAT USE",
        Description: "Use an item during combat.",
    },
    {
        Name:        "HELP",
        Usage:       "COMBAT HELP",
        Description: "Request help during combat.",
    },
    {
        Name:        "JOIN",
        Usage:       "COMBAT JOIN",
        Description: "Join an ongoing combat.",
    },
}

func (m Model) hasHostileNPC() bool {
    for _, npc := range m.currentRoom.NPCs {
        if npc.Hostile {
            return true
        }
    }
    return false
}

func (m Model) hasFriendlyNPC() bool {
    for _, npc := range m.currentRoom.NPCs {
        if !npc.Hostile {
            return true
        }
    }
    return false
}

func (m Model) hasAvailableExits() bool {
    return len(m.currentRoom.Room.Exits) > 0
}

func (m Model) hasRoomItems() bool {
    return len(m.currentRoom.Items) > 0
}

func (m Model) hasGroup() bool {
    return m.group.ID != ""
}