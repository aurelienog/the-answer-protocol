package world

import "errors"

var (
	ErrPlayerNotFound       = errors.New("player not found")
	ErrPlayerAlreadyExists  = errors.New("player already exists")
	ErrItemNotFound         = errors.New("item not found")
	ErrItemNotInRoom        = errors.New("item not in room")
	ErrItemNotInInventory   = errors.New("item not in inventory")
	ErrItemAlreadyOwned     = errors.New("item already owned")
	ErrItemCannotBeDropped  = errors.New("item cannot be dropped")
	ErrNPCNotFound          = errors.New("NPC not found")
	ErrNPCNotHostile        = errors.New("NPC is not hostile")
	ErrNoExit               = errors.New("no exit in the requested direction")
	ErrNoQuestAvailable     = errors.New("no quest available")
	ErrInvalidWorldState    = errors.New("invalid world state")
	ErrItemNameAmbiguous = errors.New("item name is ambiguous")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrQuestNotFound = errors.New("quest not found")
)