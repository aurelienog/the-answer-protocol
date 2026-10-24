package ui

import (
	"encoding/json"
	"strings"
)

// handleOKResponse interprets the payload of a successful TAP response.
func handleOKResponse(payload string) {
	var lookData struct {
		Room struct {
			ID          string            `json:"id"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Exits       map[string]string `json:"exits"`
		} `json:"room"`
		Players []string `json:"players"`
		Items   []string `json:"items"`
		NPCs    []string `json:"npcs"`
	}

	// A LOOK response contains a room object.
	if strings.HasPrefix(payload, "{") {
		err := json.Unmarshal([]byte(payload), &lookData)
		if err == nil && lookData.Room.ID != "" {
			room := RoomView{
				ID:          lookData.Room.ID,
				Name:        lookData.Room.Name,
				Description: lookData.Room.Description,
				Exits:       lookData.Room.Exits,
				Players:     lookData.Players,
				Items:       lookData.Items,
				NPCs:        lookData.NPCs,
			}

			PrintMessage(RenderRoom(room))
			return
		}
	}

	// Other successful responses may contain plain text or JSON
	// unrelated to a room.
	PrintMessage("OK " + payload)
}
