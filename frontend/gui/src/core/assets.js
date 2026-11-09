// ==========================================================================
// CORE MODULE: ASSETS CATALOG (Rooms, Items, NPCs & Hero Avatars)
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
            icon: "⚓",
            img: "/assets/rooms/harbor_square.webp",
            cardImg: "/assets/rooms/cards/harbor_square.jpeg",
            iconImg: "/assets/rooms/icons/harbor_square.png",
            region: "Harbor District · Kingdom of Arvendale"
        },
        "market_street": {
            icon: "🎪",
            img: "/assets/rooms/market_street.webp",
            cardImg: "/assets/rooms/cards/market_street.jpeg",
            iconImg: "/assets/rooms/icons/market_street.png",
            region: "Trade Quarter · Kingdom of Arvendale"
        },
        "blacksmith_forge": {
            icon: "⚒️",
            img: "/assets/rooms/blacksmith_forge.webp",
            cardImg: "/assets/rooms/cards/blacksmith_forge.jpeg",
            iconImg: "/assets/rooms/icons/blacksmith_forge.png",
            region: "Artisan Quarter · Kingdom of Arvendale"
        },
        "apothecary": {
            icon: "⚗️",
            img: "/assets/rooms/apothecary.webp",
            cardImg: "/assets/rooms/cards/apothecary.jpeg",
            iconImg: "/assets/rooms/icons/apothecary.png",
            region: "Alchemist Alley · Kingdom of Arvendale"
        },
        "tavern": {
            icon: "🍺",
            img: "/assets/rooms/tavern.webp",
            cardImg: "/assets/rooms/cards/tavern.jpeg",
            iconImg: "/assets/rooms/icons/tavern.png",
            region: "Harbor District · Kingdom of Arvendale"
        },
        "docks": {
            icon: "⛵",
            img: "/assets/rooms/docks.webp",
            cardImg: "/assets/rooms/cards/docks.jpeg",
            iconImg: "/assets/rooms/icons/docks.png",
            region: "The Great Pier · Kingdom of Arvendale"
        },
        "warehouse": {
            icon: "📦",
            img: "/assets/rooms/warehouse.webp",
            cardImg: "/assets/rooms/cards/warehouse.jpeg",
            iconImg: "/assets/rooms/icons/warehouse.png",
            region: "Outer Docks · Smuggler's Lair"
        },
        "town_gate": {
            icon: "⛩️",
            img: "/assets/rooms/town_gate.webp",
            cardImg: "/assets/rooms/cards/town_gate.jpeg",
            iconImg: "/assets/rooms/icons/town_gate.png",
            region: "Northern Fortifications"
        },
        "forest_edge": {
            icon: "🌲",
            img: "/assets/rooms/forest_edge.webp",
            cardImg: "/assets/rooms/cards/forest_edge.jpeg",
            iconImg: "/assets/rooms/icons/forest_edge.png",
            region: "Whispering Woods"
        },
        "default": {
            icon: "🏰",
            img: "",
            cardImg: "",
            iconImg: "",
            region: "Kingdom of Arvendale"
        }
    },
    items: {
        "item.iron_sword":           { name: "Iron Sword", icon: "🗡️", img: "/assets/items/iron_sword.jpg", type: "equipment", stats: "+6 Attack", desc: "A sturdy iron blade forged by Master Borin. Passively increases Attack by +6 in combat." },
        "item.oak_shield":           { name: "Oak Shield", icon: "🛡️", img: "/assets/items/oak_shield.jpg", type: "equipment", stats: "+4 Defense", desc: "A heavy oak shield reinforced with iron bands. Passively increases Defense by +4 in combat." },
        "item.leather_boots":        { name: "Pathfinder Boots", icon: "🥾", img: "/assets/items/leather_boots.jpg", type: "equipment", stats: "+2 Def, +5 Spd", desc: "Reinforced leather boots awarded for scouting Arvendale. Increases Defense by +2 and Speed by +5." },
        "item.iron_gauntlets":       { name: "Alchemist's Iron Gauntlets", icon: "🧤", img: "/assets/items/iron_gauntlets.jpg", type: "equipment", stats: "+3 Atk, +3 Def", desc: "Runed gauntlets gifted by Apprentice Elisa. Increases Attack by +3 and Defense by +3." },
        "item.steel_breastplate":    { name: "Royal Guard Breastplate", icon: "🦺", img: "/assets/items/steel_breastplate.jpg", type: "equipment", stats: "+6 Defense", desc: "Polished steel armor awarded by Captain Eliza. Increases Defense by +6 against heavy strikes." },
        "item.royal_helmet":         { name: "Helm of Arvendale", icon: "🪖", img: "/assets/items/royal_helmet.jpg", type: "equipment", stats: "+2 Atk, +5 Def", desc: "The champion's crested helm awarded for solving the harbor conspiracy. Increases Attack by +2 and Defense by +5." },
        "item.healing_potion":       { name: "Healing Elixir", icon: "🧪", img: "/assets/items/healing_potion.jpg", type: "consumable", stats: "Restores 40 HP", desc: "A bright crimson alchemical elixir that restores 40 HP when used." },
        "item.bread":                { name: "Fresh Bread", icon: "🍞", img: "/assets/items/bread.jpg", type: "consumable", stats: "Restores 15 HP", desc: "A warm, crusty loaf baked at The Salty Siren Tavern. Restores 15 HP when eaten." },
        "item.wine":                 { name: "Harbor Vintage Wine", icon: "🍷", img: "/assets/items/wine.jpg", type: "consumable", stats: "Restores 25 HP", desc: "A bottle of rich spiced wine from The Salty Siren. Restores 25 HP when used." },
        "item.explorer_token":       { name: "Northern Gate Seal", icon: "🏅", img: "/assets/items/explorer_token.jpg", type: "quest", stats: "Quest Item", desc: "A stamped brass token from the North Town Gate proving you have scouted the city perimeter." },
        "item.herbs":                { name: "Healing Herbs", icon: "🌿", img: "/assets/items/herbs.jpg", type: "quest", stats: "Quest Item", desc: "Fragrant medicinal herbs gathered from the Whispering Forest or dropped by the Dire Wolf." },
        "item.compromise_letter":    { name: "Incriminating Letter", icon: "✉️", img: "/assets/items/compromise_letter.jpg", type: "quest", stats: "Quest Item", desc: "A sealed letter proving the smuggling conspiracy. Bring it to Captain Eliza on Harbor Square." },
        "item.missing_ships_report": { name: "Missing Ships Report", icon: "📋", img: "/assets/items/missing_ships_report.jpg", type: "quest", stats: "Quest Item", desc: "The Dock Warden's secret ledger detailing every ship that vanished from Arvendale. Bring it to Captain Eliza." },
        "item.gold_coins":           { name: "Purse of Gold Coins", icon: "🪙", img: "/assets/items/gold_coins.jpg", type: "unique", stats: "Trophy", desc: "A heavy purse of royal gold coins looted from the harbor conspirators." },
        "item.rope":                 { name: "Hempen Rope", icon: "🪢", img: "/assets/items/rope.jpg", type: "common", stats: "Utility", desc: "A strong coil of braided sailors' rope found on the harbor square." },
        "item.lantern":              { name: "Brass Lantern", icon: "🏮", img: "/assets/items/lantern.jpg", type: "common", stats: "Utility", desc: "A polished brass oil lantern that illuminates dark piers and storehouses." },
        "item.pouch":                { name: "Leather Pouch", icon: "👝", img: "/assets/items/pouch.jpg", type: "common", stats: "Utility", desc: "A sturdy utility belt pouch sold at the market stalls." },
        "default":                   { name: "Item", icon: "🎒", img: "", type: "common", stats: "-", desc: "An interactive object discovered in the world." }
    },
    npcs: {
        "captain_eliza": { id: "captain_eliza", name: "Captain Eliza",    hasQuest: true,  aliases: ["captain eliza"], icon: "⚔️", img: "/assets/avatars/captain_eliza.jpeg", iconImg: "/assets/avatars/icons/captain_eliza.jpeg", role: "Harbor Captain", hostile: false },
        "town_guide":    { id: "town_guide",    name: "Town Guide",       hasQuest: false, aliases: ["town guide cedric"], icon: "🗺️", img: "/assets/avatars/town_guide.jpeg", iconImg: "/assets/avatars/icons/town_guide.jpeg", role: "City Guide", hostile: false },
        "quest_advisor": { id: "quest_advisor", name: "Quest Advisor",    hasQuest: true,  aliases: ["asesor del mercado", "guild scout"], icon: "🧭", img: "/assets/avatars/quest_advisor.jpeg", iconImg: "/assets/avatars/icons/quest_advisor.jpeg", role: "Quest Advisor", hostile: false },
        "dock_warden":   { id: "dock_warden",   name: "Dock Warden",      hasQuest: false, aliases: ["corrupt dock warden"], icon: "⚓", img: "/assets/avatars/dock_warden.jpeg", iconImg: "/assets/avatars/icons/dock_warden.jpeg", role: "Harbor Boss", hostile: true },
        "blacksmith":    { id: "blacksmith",    name: "Master Borin",     hasQuest: false, aliases: ["borin"], icon: "⚒️", img: "/assets/avatars/blacksmith.jpeg", iconImg: "/assets/avatars/icons/blacksmith.jpeg", role: "Royal Blacksmith", hostile: false },
        "herbalist":     { id: "herbalist",     name: "Apprentice Elisa", hasQuest: true,  aliases: ["herbalist", "elisa"], icon: "🌿", img: "/assets/avatars/herbalist.jpeg", iconImg: "/assets/avatars/icons/herbalist.jpeg", role: "Apothecary", hostile: false },
        "tavern_keeper": { id: "tavern_keeper", name: "Mara",             hasQuest: true,  aliases: ["tavern keeper", "elara"], icon: "🍺", img: "/assets/avatars/tavern_keeper.jpeg", iconImg: "/assets/avatars/icons/tavern_keeper.jpeg", role: "Innkeeper", hostile: false },
        "smuggler":      { id: "smuggler",      name: "Shadow Smuggler",  hasQuest: false, aliases: ["smuggler"], icon: "🗡️", img: "/assets/avatars/smuggler.jpeg", iconImg: "/assets/avatars/icons/smuggler.jpeg", role: "Smuggler", hostile: true },
        "gate_guard":    { id: "gate_guard",    name: "North Gate Guard", hasQuest: false, aliases: ["gate guard"], icon: "🛡️", img: "/assets/avatars/gate_guard.jpeg", iconImg: "/assets/avatars/icons/gate_guard.jpeg", role: "Sentinel", hostile: false },
        "dire_wolf":     { id: "dire_wolf",     name: "Dire Wolf",        hasQuest: false, aliases: ["wolf"], icon: "🐺", img: "/assets/avatars/dire_wolf.jpeg", iconImg: "/assets/avatars/icons/dire_wolf.jpeg", role: "Forest Beast", hostile: true },
        "default":       { id: "default",       name: "Citizen",          hasQuest: false, aliases: [], icon: "🗣️", img: "", iconImg: "", role: "Townsperson", hostile: false }
    }
};