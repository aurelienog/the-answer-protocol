package commands

import "the-answer-protocol/internal/protocol"

type roomData struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

type lookResponse struct {
	Room    roomData `json:"room"`
	Players []string `json:"players"`
	Items   []string `json:"items"`
	NPCs    []string `json:"npcs"`
}

type LookCommand struct{}

func NewLookCommand() *LookCommand {
	return &LookCommand{}
}

func (c *LookCommand) Name() string {
	return "look"
}

func (c *LookCommand) Execute(
	ctx *Context,
) (protocol.Message, error) {
	// TODO: sustituir estos datos por el estado real del mundo.
	response := lookResponse{
		Room: roomData{
			ID:   "42_cluster",
			Name: "The 42 Computer Cluster",
			Description: "Rows of sleek computers fill the room. " +
				"The lights are dim, and you can hear the faint hum " +
				"of fans cooling down compiled binaries. " +
				"The Answer Protocol specification lies open on a desk.",
			Exits: map[string]string{
				"north": "corridor",
			},
		},
		Players: []string{},
		Items:   []string{},
		NPCs:    []string{},
	}

	return protocol.OKJSON(response)
}