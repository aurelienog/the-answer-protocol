
package layout

type LayoutManager struct{}

// Calculate distributes terminal space among the UI components.
func (lm LayoutManager) Calculate(input LayoutInput) LayoutResult {
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

	bottomPanelsEnabled :=
		!eventLog.Required &&
			!chat.Required &&
			bottomPreferredHeight > 0

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

	if !commandsRequirements.Required {
		commandsPreferred := resolveHeight(commandsRequirements)
		commandsMin := commandsRequirements.Constraints.MinHeight

		if commandsPreferred > 0 &&
			remainingHeight-minMainHeight >= commandsMin {

			commandsHeight = minInt(
				commandsPreferred,
				remainingHeight-minMainHeight,
			)

			remainingHeight -= commandsHeight
		}
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

	// Calculate the Main View width, respecting its maximum.
	mainWidth := width

	if mainConstraints.MaxWidth > 0 {
		mainWidth = minInt(mainWidth, mainConstraints.MaxWidth)
	}

	// Determine whether the Sidebar can be displayed.
	sidebarConstraints := input.Sidebar.Constraints

	sidebarMinWidth := sidebarConstraints.MinWidth
	sidebarMaxWidth := sidebarConstraints.MaxWidth
	sidebarMinHeight := sidebarConstraints.MinHeight

	sidebarVisible :=
		width >= MinWidthForSidebar &&
		mainHeight >= sidebarMinHeight &&
		width >= minMainWidth+sidebarMinWidth

	sidebarWidth := 0

	if sidebarVisible {
		availableSidebarWidth := width - mainWidth

		// Preserve the minimum Sidebar width by reducing Main View if needed.
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
		Height:  mainHeight,
	}

	if sidebarVisible {
		result.Sidebar = ComponentLayout{
			Visible: true,
			Width:   sidebarWidth,
			Height:  mainHeight,
		}
	}

	// Assign widths to the bottom panels.
	if bottomHeight > 0 {
		showEventLog := eventLogPreferred > 0
		showChat := chatPreferred > 0

		switch {
		case showEventLog && showChat:
			eventLogWidth := width / 2
			chatWidth := width - eventLogWidth

			result.EventLog = ComponentLayout{
				Visible: true,
				Width:   eventLogWidth,
				Height:  bottomHeight,
			}
			result.Chat = ComponentLayout{
				Visible: true,
				Width:   chatWidth,
				Height:  bottomHeight,
			}

		case showEventLog:
			result.EventLog = ComponentLayout{
				Visible: true,
				Width:   width,
				Height:  bottomHeight,
			}

		case showChat:
			result.Chat = ComponentLayout{
				Visible: true,
				Width:   width,
				Height:  bottomHeight,
			}
		}
	}

	if commandsHeight > 0 {
		result.AvailableCommands = ComponentLayout{
			Visible: true,
			Width:   width,
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
