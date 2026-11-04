package world

// ItemType identifies how an item behaves in the world.
type ItemType string

const (
	ItemTypeCommon     ItemType = "common"
	ItemTypeQuest      ItemType = "quest"
	ItemTypeUnique     ItemType = "unique"
	ItemTypeEquipment  ItemType = "equipment"
	ItemTypeConsumable ItemType = "consumable"
)

// Item defines an item that can exist in the world.
type Item struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        ItemType `json:"type"`

	Attack  int `json:"attack,omitempty"`
	Defense int `json:"defense,omitempty"`
	Speed   int `json:"speed,omitempty"`
	HP      int `json:"hp,omitempty"`
}

// ItemInstance represents a unique instance of an item definition.
type ItemInstance struct {
    ID     string `json:"id"`
    ItemID string `json:"item_id"`
}