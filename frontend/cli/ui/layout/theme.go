
package layout

import "github.com/charmbracelet/lipgloss"

// Medieval frame used by panels throughout the interface.
var medievalBorder = lipgloss.Border{
	Top:         "═",
	Bottom:      "═",
	Left:        "║",
	Right:       "║",
	TopLeft:     "╔",
	TopRight:    "╗",
	BottomLeft:  "╚",
	BottomRight: "╝",
	MiddleLeft:  "╠",
	MiddleRight: "╣",
}

// Theme palette.
var (
	Primary   = lipgloss.Color("#C9A44C") // Ancient Gold
	Secondary = lipgloss.Color("#477FA8") // Arcane Blue
	Success   = lipgloss.Color("#39845A") // Emerald
	Danger    = lipgloss.Color("#A83232") // Blood Red
	Warning   = lipgloss.Color("#D69B36") // Amber
	Info      = lipgloss.Color("#75A9C9") // Magic Blue
	Muted     = lipgloss.Color("#77736A") // Stone
	Accent    = lipgloss.Color("#9166A8") // Mystic Purple
)

const (
    MinWidthForSidebar = 90
    MinTerminalWidth   = 50
    MinMainHeight      = 3
    MinSidebarHeight   = 8
)

// Base component styles.
var (
	PanelStyle = lipgloss.NewStyle().
		Border(medievalBorder).
		BorderForeground(Primary).
		Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	StatusStyle = lipgloss.NewStyle().
		Foreground(Muted)

	InputStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Success)
)

const (
    PanelBorderWidth   = 2
    PanelBorderHeight  = 2
    PanelPaddingWidth  = 4
    PanelPaddingHeight = 2


    PanelWidthOffset  = PanelBorderWidth + PanelPaddingWidth
    PanelHeightOffset = PanelBorderHeight + PanelPaddingHeight
)

// Semantic styles.
var (
	DangerStyle = lipgloss.NewStyle().Foreground(Danger)
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	InfoStyle = lipgloss.NewStyle().Foreground(Info)
	AccentStyle = lipgloss.NewStyle().Foreground(Accent)
	MutedStyle = lipgloss.NewStyle().Foreground(Muted)
)

// World entity markers.
const (
	MarkerDanger       = "[!]"
	MarkerInteractable = "[*]"
	MarkerAlly         = "[+]"
	MarkerQuestNPC     = "[?]"
	MarkerBlockedExit  = "[X]"
)

// Quest status markers.
const (
	MarkerQuestPending   = "[ ]"
	MarkerQuestCompleted = "[✓]"
	MarkerQuestBlocked   = "[!]"
	MarkerQuestFailed    = "[-]"
)


const (
	PlayerMarker = "[♟]"
	NpcMarker    = "[♞]"
	ObjectMarker = "[✦]"
	ExitMarker   = "[➽]"
    IconCombat   = "⚔"
    IconGroup    = "⛨"
    IconQuest    = "♛"
    IconHealing  = "✚"
    IconLocation = "♜"
	IconInventory = "⛏"
    IconExit     = "➤"
    IconNPC      = "✠"
    IconPlayer   = "🗡"

)

// 1. Los colores tienen significado semántico. Por ejemplo, DangerStyle se utilizará para enemigos hostiles, alertas y errores; SuccessStyle, para acciones completadas o resultados positivos.

// 2. Los indicadores son constantes, no estilos. [!], [*] y [+] representan el significado de una entidad. Más adelante, los componentes decidirán cómo mostrarlos y qué color aplicarles.

// 3. No añadimos datos que todavía no tenemos. Por ejemplo, no definimos estilos para Mana, XP o stamina hasta que formen parte de los datos reales del jugador.

// 4. Los indicadores no implican que el servidor ya proporcione esa información. Por ejemplo, [X] solo se mostrará cuando tengamos información que permita saber que una salida está bloqueada.

// Una vez tengamos este fichero, el siguiente paso será actualizar sidebar.go para que utilice el tema, sin duplicar colores ni estilos.