package layout_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

func TestRenderPanelUsesAllocatedHeightForDifferentContent(t *testing.T) {
	const panelHeight = 12

	shortPanel := layout.RenderPanel("short", 50, panelHeight)
	tallPanel := layout.RenderPanel(strings.Join([]string{
		"line one",
		"line two",
		"line three",
		"line four",
	}, "\n"), 50, panelHeight)

	if got := lipgloss.Height(shortPanel); got != panelHeight {
		t.Errorf("short panel height = %d, want %d", got, panelHeight)
	}

	if got := lipgloss.Height(tallPanel); got != panelHeight {
		t.Errorf("tall panel height = %d, want %d", got, panelHeight)
	}
}

func TestRenderPanelClipsContentToAllocatedHeight(t *testing.T) {
	const (
		panelWidth  = 30
		panelHeight = 8
	)

	panel := layout.RenderPanel(
		strings.Repeat("a long message that wraps across the panel ", 20),
		panelWidth,
		panelHeight,
	)

	if got := lipgloss.Height(panel); got != panelHeight {
		t.Errorf("panel height = %d, want %d", got, panelHeight)
	}

	if got := lipgloss.Width(panel); got != panelWidth {
		t.Errorf("panel width = %d, want %d", got, panelWidth)
	}
}

func TestRenderedSectionsUseSameWidth(t *testing.T) {
	const totalWidth = 100

	header := layout.RenderHeader("adventurer", 10, 10, totalWidth)
	body := layout.RenderMain("main", "sidebar", 60, 40, 10, true)
	bottom := lipgloss.JoinHorizontal(
		lipgloss.Top,
		layout.RenderPanel("event log", 50, 12),
		layout.RenderPanel("chat", 50, 12),
	)

	for name, section := range map[string]string{
		"header":           header,
		"main and sidebar": body,
		"bottom panels":    bottom,
	} {
		if got := lipgloss.Width(section); got != totalWidth {
			t.Errorf("%s width = %d, want %d", name, got, totalWidth)
		}
	}
}
