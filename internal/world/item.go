package world

// ItemType identifies how an item behaves in the world.
type ItemType string

const (
	ItemTypeCommon ItemType = "common"
	ItemTypeQuest  ItemType = "quest"
	ItemTypeUnique ItemType = "unique"
)

// Item defines an item that can exist in the world.
type Item struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        ItemType `json:"type"`
}
