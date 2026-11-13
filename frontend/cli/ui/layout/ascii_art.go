package layout

import (
    "strings"
    "os"
    "path/filepath"
    "github.com/charmbracelet/lipgloss"
)

const asciiArtPath = "frontend/cli/ui/assets/ascii"

const (
    ASCIIArtWidth  = 42
    ASCIIArtHeight = 12
)


func RenderASCIIArt(art string, availableWidth, availableHeight int) string {
	if availableWidth < ASCIIArtWidth {
		return ""
	}

	lines := strings.Split(strings.TrimRight(art, "\n"), "\n")

	if len(lines) > ASCIIArtHeight || len(lines) > availableHeight {
		return ""
	}

	return lipgloss.NewStyle().
		Width(ASCIIArtWidth).
		Align(lipgloss.Center).
		Render(strings.Join(lines, "\n"))
}

func GetRoomASCIIArt(roomName string) string {
    var filename string

    switch strings.ToLower(strings.TrimSpace(roomName)) {
    case "central harbor square":
        filename = "harbor_square.txt"
    case "tavern":
        filename = "tavern.txt"
    case "forest":
        filename = "forest.txt"
    case "dungeon":
        filename = "dungeon.txt"
    default:
        return ""
    }

    data, err := os.ReadFile(filepath.Join(asciiArtPath, filename))
    if err != nil {
        return ""
    }

    return string(data)
}