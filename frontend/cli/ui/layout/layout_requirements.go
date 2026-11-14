package layout

const (
    MinWidthForSidebar = 90
    MinTerminalWidth   = 50
    MinMainHeight      = 10
    MinSidebarHeight   = 8
    MinSidebarWidth    = 20
    MaxMainWidth       = 90
    MaxSidebarWidth    = 32
)
const (
	InputHeight             = 1
	StatusHeight            = 1
	EventLogPreferredHeight = 8
	ChatPreferredHeight     = 8
)

type LayoutInput struct {
	Terminal Size

	Header            ComponentRequirements
	MainView          ComponentRequirements
	Sidebar           ComponentRequirements
	EventLog          ComponentRequirements
	Chat              ComponentRequirements
	AvailableCommands ComponentRequirements
	Input             ComponentRequirements
	Status            ComponentRequirements
}

func DefaultRequirements() LayoutInput {
	return LayoutInput{
		Header: ComponentRequirements{
			Constraints: Constraints{},
			Priority:    PriorityCritical,
			Required:    true,
		},
		MainView: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        MinTerminalWidth,
				PreferredWidth:  MaxMainWidth,
				MaxWidth:        MaxMainWidth,
				MinHeight:       MinMainHeight + PanelHeightOffset,
				PreferredHeight: MinMainHeight + PanelHeightOffset,
				MaxHeight:       0,
			},
			Priority: PriorityCritical,
			Required: true,
		},
		Sidebar: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        MinSidebarWidth + PanelWidthOffset,
				PreferredWidth:  MaxSidebarWidth,
				MaxWidth:        MaxSidebarWidth,
				MinHeight:       MinSidebarHeight + PanelHeightOffset,
				PreferredHeight: MinSidebarHeight + PanelHeightOffset,
				MaxHeight:       0,
			},
			Priority: PriorityNormal,
			Required: false,
		},
		EventLog: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        0,
				PreferredWidth: 0,
				MaxWidth:        0,
				MinHeight:       EventLogPreferredHeight,
				PreferredHeight: EventLogPreferredHeight,
				MaxHeight:       EventLogPreferredHeight,
			},
			Priority: PriorityNormal,
			Required: false,
		},
		Chat: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        0,
				PreferredWidth: 0,
				MaxWidth:        0,
				MinHeight:       ChatPreferredHeight,
				PreferredHeight: ChatPreferredHeight,
				MaxHeight:       ChatPreferredHeight,
			},
			Priority: PriorityNormal,
			Required: false,
		},
		AvailableCommands: ComponentRequirements{
			Constraints: Constraints{
				MinHeight:       0,
				PreferredHeight: 0,
				MaxHeight:       0,
			},
			Priority: PriorityLow,
			Required: false,
		},
		Input: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        0,
				PreferredWidth: 0,
				MaxWidth:        0,
				MinHeight:       InputHeight,
				PreferredHeight: InputHeight,
				MaxHeight:       InputHeight,
			},
			Priority: PriorityCritical,
			Required: true,
		},
		Status: ComponentRequirements{
			Constraints: Constraints{
				MinWidth:        0,
				PreferredWidth: 0,
				MaxWidth:        0,
				MinHeight:       StatusHeight,
				PreferredHeight: StatusHeight,
				MaxHeight:       StatusHeight,
			},
			Priority: PriorityCritical,
			Required: true,
		},
	}
}