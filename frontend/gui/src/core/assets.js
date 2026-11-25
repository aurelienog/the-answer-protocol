// ==========================================================================
// CORE MODULE: VISUAL ASSETS CATALOG (Synced with RFC 42TAP Full-Name IDs)
// ==========================================================================

const ASSETS = {
    combat: {
        heroes: {
            male:   "/assets/combat/hero_male.webp",
            female: "/assets/combat/hero_female.webp"
        },
        enemies: {
            "npc.dire_wolf":       "/assets/combat/dire_wolf.webp",
            "npc.shadow_smuggler": "/assets/combat/smuggler.webp",
            "npc.dock_warden":     "/assets/combat/dock_warden.webp",
            "default":             "/assets/combat/unknown_enemy.webp"
        },
        backgrounds: {
            "loc.the_great_pier":         "/assets/combat/bg_docks.webp",
            "loc.whispering_forest_edge": "/assets/combat/bg_forest_edge.webp",
            "loc.abandoned_warehouse":    "/assets/combat/bg_warehouse.webp",
            "npc.dock_warden":            "/assets/combat/bg_docks.webp",
            "npc.dire_wolf":              "/assets/combat/bg_forest_edge.webp",
            "npc.shadow_smuggler":        "/assets/combat/bg_warehouse.webp",
            "default":                    "/assets/rooms/unknown_room.webp"
        }
    },

    avatars: {
        male:   { img: "/assets/avatars/hero_male.webp",   iconImg: "/assets/avatars/icons/hero_male.jpeg",   icon: "🧙‍♂️" },
        female: { img: "/assets/avatars/hero_female.webp", iconImg: "/assets/avatars/icons/hero_female.jpeg", icon: "🧝‍♀️" }
    },

    rooms: {
        "loc.central_harbor_square": {
            icon: "⚓", img: "/assets/rooms/harbor_square.webp",
            cardImg: "/assets/rooms/cards/harbor_square.webp", iconImg: "/assets/rooms/icons/harbor_square.png",
            region: "Harbor District · Kingdom of Arvendale"
        },
        "loc.market_street": {
            icon: "🎪", img: "/assets/rooms/market_street.webp",
            cardImg: "/assets/rooms/cards/market_street.webp", iconImg: "/assets/rooms/icons/market_street.png",
            region: "Trade Quarter · Kingdom of Arvendale"
        },
        "loc.royal_blacksmith_forge": {
            icon: "⚒️", img: "/assets/rooms/blacksmith_forge.webp",
            cardImg: "/assets/rooms/cards/blacksmith_forge.webp", iconImg: "/assets/rooms/icons/blacksmith_forge.png",
            region: "Artisan Quarter · Kingdom of Arvendale"
        },
        "loc.apothecary_shop": {
            icon: "⚗️", img: "/assets/rooms/apothecary.webp",
            cardImg: "/assets/rooms/cards/apothecary.webp", iconImg: "/assets/rooms/icons/apothecary.png",
            region: "Alchemist Alley · Kingdom of Arvendale"
        },
        "loc.the_salty_siren_tavern": {
            icon: "🍺", img: "/assets/rooms/tavern.webp",
            cardImg: "/assets/rooms/cards/tavern.webp", iconImg: "/assets/rooms/icons/tavern.png",
            region: "Harbor District · Kingdom of Arvendale"
        },
        "loc.the_great_pier": {
            icon: "⛵", img: "/assets/rooms/docks.webp",
            cardImg: "/assets/rooms/cards/docks.webp", iconImg: "/assets/rooms/icons/docks.png",
            region: "The Great Pier · Kingdom of Arvendale"
        },
        "loc.abandoned_warehouse": {
            icon: "📦", img: "/assets/rooms/warehouse.webp",
            cardImg: "/assets/rooms/cards/warehouse.webp", iconImg: "/assets/rooms/icons/warehouse.png",
            region: "Outer Docks · Smuggler's Lair"
        },
        "loc.north_town_gate": {
            icon: "⛩️", img: "/assets/rooms/town_gate.webp",
            cardImg: "/assets/rooms/cards/town_gate.webp", iconImg: "/assets/rooms/icons/town_gate.png",
            region: "Northern Fortifications"
        },
        "loc.whispering_forest_edge": {
            icon: "🌲", img: "/assets/rooms/forest_edge.webp",
            cardImg: "/assets/rooms/cards/forest_edge.webp", iconImg: "/assets/rooms/icons/forest_edge.png",
            region: "Whispering Woods"
        },
        "default": {
            icon: "🏰", img: "/assets/rooms/unknown_room.webp",
            cardImg: "/assets/rooms/unknown_room.webp", iconImg: "/assets/rooms/icons/unknown_room.webp",
            region: "Unknown Region · Kingdom of Arvendale"
        }
    },

    items: {
        "item.iron_sword":                { icon: "🗡️", img: "/assets/items/card/iron_sword.webp",           iconImg: "/assets/items/icons/iron_sword.webp" },
        "item.oak_shield":                { icon: "🛡️", img: "/assets/items/card/oak_shield.webp",           iconImg: "/assets/items/icons/oak_shield.webp" },
        "item.pathfinder_boots":          { icon: "🥾", img: "/assets/items/card/leather_boots.webp",        iconImg: "/assets/items/icons/leather_boots.webp" },
        "item.alchemists_iron_gauntlets": { icon: "🧤", img: "/assets/items/card/iron_gauntlets.webp",       iconImg: "/assets/items/icons/iron_gauntlets.webp" },
        "item.royal_guard_breastplate":   { icon: "🦺", img: "/assets/items/card/iron_chest_armor.webp",     iconImg: "/assets/items/icons/iron_chest_armor.webp" },
        "item.helm_of_arvendale":         { icon: "🪖", img: "/assets/items/card/iron_helmet.webp",          iconImg: "/assets/items/icons/iron_helmet.webp" },
        "item.healing_elixir":            { icon: "🧪", img: "/assets/items/card/health_potion.webp",        iconImg: "/assets/items/icons/health_potion.webp" },
        "item.fresh_bread":               { icon: "🍞", img: "/assets/items/card/fresh_bread.webp",          iconImg: "/assets/items/icons/fresh_bread.webp" },
        "item.harbor_vintage_wine":       { icon: "🍷", img: "/assets/items/card/harbor_vintage_wine.webp",  iconImg: "/assets/items/icons/harbor_vintage_wine.webp" },
        "item.northern_gate_seal":        { icon: "🏅", img: "/assets/items/card/explorer_token.webp",       iconImg: "/assets/items/icons/explorer_token.webp" },
        "item.healing_herbs":             { icon: "🌿", img: "/assets/items/card/healing_herbs.webp",        iconImg: "/assets/items/icons/healing_herbs.webp" },
        "item.incriminating_letter":      { icon: "✉️", img: "/assets/items/card/incriminating_letter.webp", iconImg: "/assets/items/icons/incriminating_letter.webp" },
        "item.missing_ships_report":      { icon: "📋", img: "/assets/items/card/missing_ships_report.webp", iconImg: "/assets/items/icons/missing_ships_report.webp" },
        "item.purse_of_gold_coins":       { icon: "🪙", img: "/assets/items/card/gold_coins.webp",           iconImg: "/assets/items/icons/gold_coins.webp" },
        "item.hempen_rope":               { icon: "🪢", img: "/assets/items/card/hempen_rope.webp",          iconImg: "/assets/items/icons/hempen_rope.webp" },
        "item.brass_lantern":             { icon: "🏮", img: "/assets/items/card/brass_lantern.webp",        iconImg: "/assets/items/icons/brass_lantern.webp" },
        "item.leather_pouch":             { icon: "👝", img: "/assets/items/card/leather_pouch.webp",        iconImg: "/assets/items/icons/leather_pouch.webp" },
        "default": {
            name: "Item", icon: "🎒",
            img: "/assets/items/card/unknown_item.webp", iconImg: "/assets/items/icons/unknown_item.webp",
            type: "common", stats: "Utility", desc: "An interactive object discovered in the world."
        }
    },

    npcs: {
        "npc.captain_eliza":    { id: "npc.captain_eliza",    icon: "⚔️", img: "/assets/avatars/captain_eliza.webp", iconImg: "/assets/avatars/icons/captain_eliza.jpeg", role: "Harbor Captain" },
        "npc.town_guide":       { id: "npc.town_guide",       icon: "🗺️", img: "/assets/avatars/town_guide.webp",    iconImg: "/assets/avatars/icons/town_guide.jpeg",    role: "City Guide" },
        "npc.quest_advisor":    { id: "npc.quest_advisor",    icon: "🧭", img: "/assets/avatars/quest_advisor.webp", iconImg: "/assets/avatars/icons/quest_advisor.jpeg", role: "Quest Advisor" },
        "npc.dock_warden":      { id: "npc.dock_warden",      icon: "⚓", img: "/assets/avatars/dock_warden.webp",   iconImg: "/assets/avatars/icons/dock_warden.jpeg",   role: "Harbor Boss" },
        "npc.master_borin":     { id: "npc.master_borin",     icon: "⚒️", img: "/assets/avatars/blacksmith.webp",    iconImg: "/assets/avatars/icons/blacksmith.jpeg",    role: "Royal Blacksmith" },
        "npc.apprentice_lina":  { id: "npc.apprentice_lina",  icon: "🌿", img: "/assets/avatars/herbalist.webp",     iconImg: "/assets/avatars/icons/herbalist.jpeg",     role: "Apothecary" },
        "npc.mara":             { id: "npc.mara",             icon: "🍺", img: "/assets/avatars/tavern_keeper.webp", iconImg: "/assets/avatars/icons/tavern_keeper.jpeg", role: "Innkeeper" },
        "npc.shadow_smuggler":  { id: "npc.shadow_smuggler",  icon: "🗡️", img: "/assets/avatars/smuggler.webp",      iconImg: "/assets/avatars/icons/smuggler.jpeg",      role: "Shadow Smuggler" },
        "npc.north_gate_guard": { id: "npc.north_gate_guard", icon: "🛡️", img: "/assets/avatars/gate_guard.webp",    iconImg: "/assets/avatars/icons/gate_guard.jpeg",    role: "North Gate Guard" },
        "npc.dire_wolf":        { id: "npc.dire_wolf",        icon: "🐺", img: "/assets/avatars/dire_wolf.webp",     iconImg: "/assets/avatars/icons/dire_wolf.jpeg",     role: "Forest Beast" },
        "default":              { id: "default", name: "Unknown Character", icon: "🗣️", img: "/assets/avatars/unknown_npc.webp", iconImg: "/assets/avatars/icons/unknown_npc.webp", role: "Stranger", hostile: false }
    }
};