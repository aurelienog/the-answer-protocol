package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type TransitionPhase uint8

const (
	TransitionReveal TransitionPhase = iota + 1
	TransitionComplete
)

type TransitionMsg struct {
	Phase TransitionPhase
}

const transitionDuration = 120 * time.Millisecond

func NextTransition(phase TransitionPhase) tea.Cmd {
	return tea.Tick(transitionDuration, func(time.Time) tea.Msg {
		return TransitionMsg{Phase: phase}
	})
}

func (m *Model) startViewTransition(view View) tea.Cmd {
    if m.ui.Transitioning {
        m.ui.PendingView = view
        return nil
    }

    if m.navigation.CurrentView == view {
        return nil
    }

    m.ui.Transitioning = true
    m.ui.TransitionPhase = 0
    m.ui.PendingView = view

    return NextTransition(TransitionReveal)
}