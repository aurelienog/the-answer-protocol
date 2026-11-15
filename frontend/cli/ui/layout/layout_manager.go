
package layout

type LayoutManager struct{}

type HorizontalLayout struct {
    MainView          ComponentLayout
    Sidebar           ComponentLayout
    EventLog          ComponentLayout
    Chat              ComponentLayout
    AvailableCommands ComponentLayout
	Status 			  ComponentLayout
}

// Calculate distributes terminal space among the UI components.
func (lm LayoutManager) Calculate(
    input LayoutInput,
    horizontal HorizontalLayout,
) LayoutResult {

	width := input.Terminal.Width
	height := input.Terminal.Height

	result := LayoutResult{}

	if width <= 0 || height <= 0 {
		return result
	}

	// Resolve the heights of the fixed components.
	headerHeight := resolveHeight(input.Header)
	inputHeight := resolveHeight(input.Input)
	statusHeight := resolveHeight(input.Status)

	result.Header = ComponentLayout{
		Visible: true,
		Width:   width,
		Height:  headerHeight,
	}
	result.Input = ComponentLayout{
		Visible: true,
		Width:   width,
		Height:  inputHeight,
	}
	result.Status = ComponentLayout{
		Visible: true,
		Width:   width,
		Height:  statusHeight,
	}

	availableHeight := height -
		headerHeight -
		inputHeight -
		statusHeight

	mainConstraints := input.MainView.Constraints
	minMainHeight := mainConstraints.MinHeight
	minMainWidth := mainConstraints.MinWidth

	// Preserve the Main View minimum dimensions.
	if width < minMainWidth || availableHeight < minMainHeight {
		return result
	}

	remainingHeight := availableHeight

	// Event Log and Chat share one row.
	eventLog := input.EventLog
	chat := input.Chat

	eventLogMin := eventLog.Constraints.MinHeight
	chatMin := chat.Constraints.MinHeight

	eventLogPreferred := resolveHeight(eventLog)
	chatPreferred := resolveHeight(chat)

	bottomMinHeight := maxInt(eventLogMin, chatMin)
	bottomPreferredHeight := maxInt(
		eventLogPreferred,
		chatPreferred,
	)

	bottomHeight := 0

	bottomPanelsEnabled := bottomPreferredHeight > 0

	if bottomPanelsEnabled &&
		remainingHeight-minMainHeight >= bottomMinHeight {

		bottomHeight = minInt(
			bottomPreferredHeight,
			remainingHeight-minMainHeight,
		)

		remainingHeight -= bottomHeight
	}

	// AvailableCommands has lower priority than the bottom panels.
	commandsHeight := 0
	commandsRequirements := input.AvailableCommands
	commandsPreferred := resolveHeight(commandsRequirements)

	if commandsPreferred > 0 &&
		remainingHeight-minMainHeight >= commandsPreferred {

		commandsHeight = commandsPreferred
		remainingHeight -= commandsHeight
	}

	// Main View receives the remaining vertical space.
	mainHeight := remainingHeight

	if mainConstraints.MaxHeight > 0 {
		mainHeight = minInt(
			mainHeight,
			mainConstraints.MaxHeight,
		)
	}

	if mainHeight < minMainHeight {
		return LayoutResult{
			Header: result.Header,
			Input:  result.Input,
			Status: result.Status,
		}
	}

	result.MainView = ComponentLayout{
		Visible: horizontal.MainView.Visible,
		Width:   horizontal.MainView.Width,
		Height:  mainHeight,
	}

	sidebarMinHeight := input.Sidebar.Constraints.MinHeight

	if horizontal.Sidebar.Visible && mainHeight >= sidebarMinHeight {
		result.Sidebar = ComponentLayout{
			Visible: true,
			Width:   horizontal.Sidebar.Width,
			Height:  mainHeight,
		}
	}

	// Assign the previously calculated widths to the bottom panels.
	if bottomHeight > 0 {
		if horizontal.EventLog.Visible {
			result.EventLog = ComponentLayout{
				Visible: true,
				Width:   horizontal.EventLog.Width,
				Height:  bottomHeight,
			}
		}

		if horizontal.Chat.Visible {
			result.Chat = ComponentLayout{
				Visible: true,
				Width:   horizontal.Chat.Width,
				Height:  bottomHeight,
			}
		}
	}

	if commandsHeight > 0 && horizontal.AvailableCommands.Visible {
		result.AvailableCommands = ComponentLayout{
			Visible: true,
			Width:   horizontal.AvailableCommands.Width,
			Height:  commandsHeight,
		}
	}

	return result
}

// resolveHeight uses the intrinsic height when available,
// then applies the component's declared constraints.
func resolveHeight(requirements ComponentRequirements) int {
	height := requirements.IntrinsicSize.Height

	if height <= 0 {
		height = requirements.Constraints.PreferredHeight
	}

	constraints := requirements.Constraints

	if constraints.MinHeight > 0 {
		height = maxInt(height, constraints.MinHeight)
	}

	if constraints.MaxHeight > 0 {
		height = minInt(height, constraints.MaxHeight)
	}

	return height
}

// CalculateHorizontal distributes available width among UI components.
func (lm LayoutManager) CalculateHorizontal(input LayoutInput) HorizontalLayout {
    width := input.Terminal.Width

    result := HorizontalLayout{}

    if width <= 0 {
        return result
    }

	result.Status = ComponentLayout{
		Visible: true,
		Width:   width,
	}
    mainConstraints := input.MainView.Constraints
    sidebarConstraints := input.Sidebar.Constraints

    minMainWidth := mainConstraints.MinWidth
    mainWidth := width

    if mainConstraints.MaxWidth > 0 {
        mainWidth = minInt(mainWidth, mainConstraints.MaxWidth)
    }

    sidebarMinWidth := sidebarConstraints.MinWidth
    sidebarMaxWidth := sidebarConstraints.MaxWidth

    sidebarVisible :=
        width >= MinWidthForSidebar &&
            width >= minMainWidth+sidebarMinWidth

    sidebarWidth := 0

    if sidebarVisible {
        availableSidebarWidth := width - mainWidth

        if availableSidebarWidth < sidebarMinWidth {
            mainWidth = maxInt(
                minMainWidth,
                width-sidebarMinWidth,
            )

            availableSidebarWidth = width - mainWidth
        }

        sidebarWidth = minInt(
            availableSidebarWidth,
            sidebarMaxWidth,
        )

        if sidebarWidth < sidebarMinWidth {
            sidebarVisible = false
            sidebarWidth = 0
        }
    }

    result.MainView = ComponentLayout{
        Visible: true,
        Width:   mainWidth,
    }

    if sidebarVisible {
        result.Sidebar = ComponentLayout{
            Visible: true,
            Width:   sidebarWidth,
        }
    }

    // EventLog and Chat share the available terminal width.
	showEventLog := resolveHeight(input.EventLog) > 0
	showChat := resolveHeight(input.Chat) > 0
	contentWidth := result.MainView.Width + result.Sidebar.Width
	switch {
	case showEventLog && showChat:
		eventLogWidth := contentWidth / 2

		result.EventLog = ComponentLayout{
			Visible: true,
			Width:   eventLogWidth,
		}

		result.Chat = ComponentLayout{
			Visible: true,
			Width:   contentWidth - eventLogWidth,
		}

	case showEventLog:
		result.EventLog = ComponentLayout{
			Visible: true,
			Width:   contentWidth,
		}

	case showChat:
		result.Chat = ComponentLayout{
			Visible: true,
			Width:   contentWidth,
		}
	}

    result.AvailableCommands = ComponentLayout{
		Visible: true,
		Width:   contentWidth,
	}

    return result
}

func resolveWidth(requirements ComponentRequirements) int {
    width := requirements.IntrinsicSize.Width

    if width <= 0 {
        width = requirements.Constraints.PreferredWidth
    }

    constraints := requirements.Constraints

    if constraints.MinWidth > 0 {
        width = maxInt(width, constraints.MinWidth)
    }

    if constraints.MaxWidth > 0 {
        width = minInt(width, constraints.MaxWidth)
    }

    return width
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func maxInt(a, b int) int {
	if a > b {
		 return a
	}

	return b
}

// RenderPanel renders content using the allocated outer dimensions.
func RenderPanel(content string, width, height int) string {
	return PanelStyle.
		Width(maxInt(0, width-PanelBorderWidth)).
		Height(maxInt(0, height-PanelBorderHeight)).
		Render(content)
}