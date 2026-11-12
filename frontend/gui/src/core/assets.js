// ==========================================================================
// CORE MODULE: VISUAL ASSETS CATALOG (Pure Visual Mapping — Zero Hardcoded Stats!)
// Names, descriptions, combat stats, item bonuses, and quests are synced
// automatically at runtime from the server's world.json via /api/world.
// ==========================================================================

const ASSETS = {
    combat: {
        heroes: {
            male:   "/assets/combat/hero_male.webp",
            female: "/assets/combat/hero_female.webp"
        },
        enemies: {
            "dire_wolf":   "/assets/combat/dire_wolf.webp",
            "smuggler":    "/assets/combat/smuggler.webp",
            "dock_warden": "/assets/combat/dock_warden.webp"
        },
        backgrounds: {
            "docks":       "/assets/combat/bg_docks.webp",
            "forest_edge": "/assets/combat/bg_forest_edge.webp",
            "warehouse":   "/assets/combat/bg_warehouse.webp",
            "dock_warden": "/assets/combat/bg_docks.webp",
            "dire_wolf":   "/assets/combat/bg_forest_edge.webp",
            "smuggler":    "/assets/combat/bg_warehouse.webp"
        }
    },

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
        "item.iron_sword": {
            icon:    "🗡️",
            img:     "/assets/items/card/iron_sword.webp",
            iconImg: "/assets/items/icons/iron_sword.webp"
        },
        "item.oak_shield": {
            icon:    "🛡️",
            img:     "/assets/items/card/oak_shield.webp",
            iconImg: "/assets/items/icons/oak_shield.webp"
        },
        "item.leather_boots": {
            icon:    "🥾",
            img:     "/assets/items/card/leather_boots.webp",
            iconImg: "/assets/items/icons/leather_boots.webp"
        },
        "item.iron_gauntlets": {
            icon:    "🧤",
            img:     "/assets/items/card/iron_gauntlets.webp",
            iconImg: "/assets/items/icons/iron_gauntlets.webp"
        },
        "item.steel_breastplate": {
            icon:    "🦺",
            img:     "/assets/items/card/iron_chest_armor.webp",
            iconImg: "/assets/items/icons/iron_chest_armor.webp"
        },
        "item.royal_helmet": {
            icon:    "🪖",
            img:     "/assets/items/card/iron_helmet.webp",
            iconImg: "/assets/items/icons/iron_helmet.webp"
        },
        "item.healing_potion": {
            icon:    "🧪",
            img:     "/assets/items/card/health_potion.webp",
            iconImg: "/assets/items/icons/health_potion.webp"
        },
        "item.bread": {
            icon:    "🍞",
            img:     "/assets/items/card/fresh_bread.webp",
            iconImg: "/assets/items/icons/fresh_bread.webp"
        },
        "item.wine": {
            icon:    "🍷",
            img:     "/assets/items/card/harbor_vintage_wine.webp",
            iconImg: "/assets/items/icons/harbor_vintage_wine.webp"
        },
        "item.explorer_token": {
            icon:    "🏅",
            img:     "/assets/items/card/explorer_token.webp",
            iconImg: "/assets/items/icons/explorer_token.webp"
        },
        "item.herbs": {
            icon:    "🌿",
            img:     "/assets/items/card/healing_herbs.webp",
            iconImg: "/assets/items/icons/healing_herbs.webp"
        },
        "item.compromise_letter": {
            icon:    "✉️",
            img:     "/assets/items/card/incriminating_letter.webp",
            iconImg: "/assets/items/icons/incriminating_letter.webp"
        },
        "item.missing_ships_report": {
            icon:    "📋",
            img:     "/assets/items/card/missing_ships_report.webp",
            iconImg: "/assets/items/icons/missing_ships_report.webp"
        },
        "item.gold_coins": {
            icon:    "🪙",
            img:     "/assets/items/card/gold_coins.webp",
            iconImg: "/assets/items/icons/gold_coins.webp"
        },
        "item.rope": {
            icon:    "🪢",
            img:     "/assets/items/card/hempen_rope.webp",
            iconImg: "/assets/items/icons/hempen_rope.webp"
        },
        "item.lantern": {
            icon:    "🏮",
            img:     "/assets/items/card/brass_lantern.webp",
            iconImg: "/assets/items/icons/brass_lantern.webp"
        },
        "item.pouch": {
            icon:    "👝",
            img:     "/assets/items/card/leather_pouch.webp",
            iconImg: "/assets/items/icons/leather_pouch.webp"
        },
        "default": {
            name:    "Item",
            icon:    "🎒",
            img:     "/assets/ui/card/look-items.jpeg",
            iconImg: "/assets/icons/command/inventory.png",
            type:    "common",
            stats:   "Utility",
            desc:    "An interactive object discovered in the world."
        }
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