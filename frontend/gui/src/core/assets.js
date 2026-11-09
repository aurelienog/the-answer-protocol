// ==========================================================================
// CORE MODULE: VISUAL ASSETS CATALOG (Pure Visual Mapping — Zero Hardcoded Stats!)
// Names, descriptions, combat stats, item bonuses, and quests are synced
// automatically at runtime from the server's world.json via /api/world.
// ==========================================================================

const ASSETS = {
    avatars: {
        male: {
            img:     "/assets/avatars/hero_male.jpeg",
            iconImg: "/assets/avatars/icons/hero_male.jpeg",
            icon:    "🧙‍♂️"
        },
        female: {
            img:     "/assets/avatars/hero_female.jpeg",
            iconImg: "/assets/avatars/icons/hero_female.jpeg",
            icon:    "🧝‍♀️"
        }
    },

    rooms: {
        "harbor_square": {
            icon:    "⚓",
            img:     "/assets/rooms/harbor_square.webp",
            cardImg: "/assets/rooms/cards/harbor_square.jpeg",
            iconImg: "/assets/rooms/icons/harbor_square.png",
            region:  "Harbor District · Kingdom of Arvendale"
        },
        "market_street": {
            icon:    "🎪",
            img:     "/assets/rooms/market_street.webp",
            cardImg: "/assets/rooms/cards/market_street.jpeg",
            iconImg: "/assets/rooms/icons/market_street.png",
            region:  "Trade Quarter · Kingdom of Arvendale"
        },
        "blacksmith_forge": {
            icon:    "⚒️",
            img:     "/assets/rooms/blacksmith_forge.webp",
            cardImg: "/assets/rooms/cards/blacksmith_forge.jpeg",
            iconImg: "/assets/rooms/icons/blacksmith_forge.png",
            region:  "Artisan Quarter · Kingdom of Arvendale"
        },
        "apothecary": {
            icon:    "⚗️",
            img:     "/assets/rooms/apothecary.webp",
            cardImg: "/assets/rooms/cards/apothecary.jpeg",
            iconImg: "/assets/rooms/icons/apothecary.png",
            region:  "Alchemist Alley · Kingdom of Arvendale"
        },
        "tavern": {
            icon:    "🍺",
            img:     "/assets/rooms/tavern.webp",
            cardImg: "/assets/rooms/cards/tavern.jpeg",
            iconImg: "/assets/rooms/icons/tavern.png",
            region:  "Harbor District · Kingdom of Arvendale"
        },
        "docks": {
            icon:    "⛵",
            img:     "/assets/rooms/docks.webp",
            cardImg: "/assets/rooms/cards/docks.jpeg",
            iconImg: "/assets/rooms/icons/docks.png",
            region:  "The Great Pier · Kingdom of Arvendale"
        },
        "warehouse": {
            icon:    "📦",
            img:     "/assets/rooms/warehouse.webp",
            cardImg: "/assets/rooms/cards/warehouse.jpeg",
            iconImg: "/assets/rooms/icons/warehouse.png",
            region:  "Outer Docks · Smuggler's Lair"
        },
        "town_gate": {
            icon:    "⛩️",
            img:     "/assets/rooms/town_gate.webp",
            cardImg: "/assets/rooms/cards/town_gate.jpeg",
            iconImg: "/assets/rooms/icons/town_gate.png",
            region:  "Northern Fortifications"
        },
        "forest_edge": {
            icon:    "🌲",
            img:     "/assets/rooms/forest_edge.webp",
            cardImg: "/assets/rooms/cards/forest_edge.jpeg",
            iconImg: "/assets/rooms/icons/forest_edge.png",
            region:  "Whispering Woods"
        },
        "default": {
            icon:    "🏰",
            img:     "/assets/rooms/harbor_square.webp",
            cardImg: "/assets/rooms/cards/harbor_square.jpeg",
            iconImg: "/assets/rooms/icons/harbor_square.png",
            region:  "Kingdom of Arvendale"
        }
    },

    items: {
        "item.iron_sword":           { icon: "🗡️", img: "/assets/items/iron_sword.jpg" },
        "item.oak_shield":           { icon: "🛡️", img: "/assets/items/oak_shield.jpg" },
        "item.leather_boots":        { icon: "🥾", img: "/assets/items/leather_boots.jpg" },
        "item.iron_gauntlets":       { icon: "🧤", img: "/assets/items/iron_gauntlets.jpg" },
        "item.steel_breastplate":    { icon: "🦺", img: "/assets/items/steel_breastplate.jpg" },
        "item.royal_helmet":         { icon: "🪖", img: "/assets/items/royal_helmet.jpg" },
        "item.healing_potion":       { icon: "🧪", img: "/assets/items/healing_potion.jpg" },
        "item.bread":                { icon: "🍞", img: "/assets/items/bread.jpg" },
        "item.wine":                 { icon: "🍷", img: "/assets/items/wine.jpg" },
        "item.explorer_token":       { icon: "🏅", img: "/assets/items/explorer_token.jpg" },
        "item.herbs":                { icon: "🌿", img: "/assets/items/herbs.jpg" },
        "item.compromise_letter":    { icon: "✉️", img: "/assets/items/compromise_letter.jpg" },
        "item.missing_ships_report": { icon: "📋", img: "/assets/items/missing_ships_report.jpg" },
        "item.gold_coins":           { icon: "🪙", img: "/assets/items/gold_coins.jpg" },
        "item.rope":                 { icon: "🪢", img: "/assets/items/rope.jpg" },
        "item.lantern":              { icon: "🏮", img: "/assets/items/lantern.jpg" },
        "item.pouch":                { icon: "👝", img: "/assets/items/pouch.jpg" },
        "default":                   { name: "Item", icon: "🎒", img: "/assets/ui/card/look-items.jpeg", type: "common", stats: "Utility", desc: "An interactive object discovered in the world." }
    },

    npcs: {
        "captain_eliza": { id: "captain_eliza", icon: "⚔️", img: "/assets/avatars/captain_eliza.jpeg", iconImg: "/assets/avatars/icons/captain_eliza.jpeg", role: "Harbor Captain" },
        "town_guide":    { id: "town_guide",    icon: "🗺️", img: "/assets/avatars/town_guide.jpeg",    iconImg: "/assets/avatars/icons/town_guide.jpeg",    role: "City Guide" },
        "quest_advisor": { id: "quest_advisor", icon: "🧭", img: "/assets/avatars/quest_advisor.jpeg", iconImg: "/assets/avatars/icons/quest_advisor.jpeg", role: "Quest Advisor" },
        "dock_warden":   { id: "dock_warden",   icon: "⚓", img: "/assets/avatars/dock_warden.jpeg",   iconImg: "/assets/avatars/icons/dock_warden.jpeg",   role: "Harbor Boss" },
        "blacksmith":    { id: "blacksmith",    icon: "⚒️", img: "/assets/avatars/blacksmith.jpeg",    iconImg: "/assets/avatars/icons/blacksmith.jpeg",    role: "Royal Blacksmith" },
        "herbalist":     { id: "herbalist",     icon: "🌿", img: "/assets/avatars/herbalist.jpeg",     iconImg: "/assets/avatars/icons/herbalist.jpeg",     role: "Apothecary" },
        "tavern_keeper": { id: "tavern_keeper", icon: "🍺", img: "/assets/avatars/tavern_keeper.jpeg", iconImg: "/assets/avatars/icons/tavern_keeper.jpeg", role: "Innkeeper" },
        "smuggler":      { id: "smuggler",      icon: "🗡️", img: "/assets/avatars/smuggler.jpeg",      iconImg: "/assets/avatars/icons/smuggler.jpeg",      role: "Shadow Smuggler" },
        "gate_guard":    { id: "gate_guard",    icon: "🛡️", img: "/assets/avatars/gate_guard.jpeg",    iconImg: "/assets/avatars/icons/gate_guard.jpeg",    role: "North Gate Guard" },
        "dire_wolf":     { id: "dire_wolf",     icon: "🐺", img: "/assets/avatars/dire_wolf.jpeg",     iconImg: "/assets/avatars/icons/dire_wolf.jpeg",     role: "Forest Beast" },
        "default":       { id: "default",       name: "Citizen", icon: "🗣️", img: "/assets/ui/card/look-npcs.jpeg", iconImg: "", role: "Townsperson", hostile: false }
    }
};