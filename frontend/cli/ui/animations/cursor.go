package animations

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type CursorBlinkMsg time.Time

func CursorBlink() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return CursorBlinkMsg(t)
	})
}