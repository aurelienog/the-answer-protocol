
package layout

type Constraints struct {
	MinWidth       int
	PreferredWidth int
	MaxWidth       int

	MinHeight       int
	PreferredHeight int
	MaxHeight       int
}

type ComponentRequirements struct {
    Constraints   Constraints
    IntrinsicSize Size
    Priority      int
    Required      bool
}

const (
	PriorityLow      = 1
	PriorityNormal   = 2
	PriorityHigh     = 3
	PriorityCritical = 4
)

type Size struct {
	Width  int
	Height int
}

type ComponentLayout struct {
	Visible bool
	Width   int
	Height  int
}

type LayoutResult struct {
	Header            ComponentLayout
	MainView          ComponentLayout
	Sidebar           ComponentLayout
	EventLog          ComponentLayout
	Chat              ComponentLayout
	AvailableCommands ComponentLayout
	Input             ComponentLayout
	Status            ComponentLayout
}
