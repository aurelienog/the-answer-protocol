package ui

import (
	"fmt"
	"strings"
	"sort"
)

type RoomView struct {
	ID          string
	Name        string
	Description string
	Exits       map[string]string
	Players     []string
	Items       []string
	NPCs        []string
}

func RenderRoom(room RoomView) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("╔══════════════════════════════════════════════╗\n")
	fmt.Fprintf(&b, "║ %-44s ║\n", room.Name)
	b.WriteString("╠══════════════════════════════════════════════╣\n")
	fmt.Fprintf(&b, "║ %-44s ║\n", room.Description)
	b.WriteString("╠══════════════════════════════════════════════╣\n")

	b.WriteString("║ Exits:                                      ║\n")
	fmt.Fprintf(&b, "║   %-42s ║\n", formatExits(room.Exits))

	b.WriteString("║ Players:                                    ║\n")
	fmt.Fprintf(&b, "║   %-42s ║\n", strings.Join(room.Players, ", "))

	b.WriteString("║ Items:                                      ║\n")
	fmt.Fprintf(&b, "║   %-42s ║\n", strings.Join(room.Items, ", "))

	b.WriteString("║ NPCs:                                       ║\n")
	fmt.Fprintf(&b, "║   %-42s ║\n", strings.Join(room.NPCs, ", "))

	b.WriteString("╚══════════════════════════════════════════════╝\n")

	return b.String()
}

func formatExits(exits map[string]string) string {
	if len(exits) == 0 {
		return "None"
	}

	directions := make([]string, 0, len(exits))
	for direction, destination := range exits {
		directions = append(
			directions,
			fmt.Sprintf("%s -> %s", direction, destination),
		)
	}

	sort.Strings(directions)
	return strings.Join(directions, ", ")
}