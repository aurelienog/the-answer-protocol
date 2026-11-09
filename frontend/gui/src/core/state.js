// ==========================================================================
// CORE MODULE: RUNTIME GAME STATE
// ==========================================================================

const GameState = {
    connected: false,
    playerName: "Arthas",
    gender: localStorage.getItem('tap_gender') || "male",
    hp: 100,
    maxHp: 100,
    status: "healthy",

    room: {
        id: "harbor_square",
        name: "Central Harbor Square",
        description: "",
        exits: {}
    },
    playersInRoom: [],
    itemsInRoom: [],
    roomItemCounts: {},
    npcsInRoom: [],
    inventory: [],
    inventoryCounts: {},
    quests: [],
    selectedInventoryItem: null,

    currentGroup: null,
    groupInvitations: [],

    serverPlayerCount: 1,
    activeChatScope: "ROOM",
    commandQueue: [],

    chats: {
        ROOM: [],
        GLOBAL: [],
        GROUP: [],
        LOG: []
    },

    normalizeItem(entry) {
        if (typeof entry === 'object' && entry !== null && entry.id) {
            const existing = ASSETS.items[entry.id] || ASSETS.items.default;
            if (entry.name || entry.description || entry.type) {
                ASSETS.items[entry.id] = {
                    name:  entry.name || existing.name,
                    icon:  existing.icon,
                    img:   existing.img || "",
                    type:  entry.type || existing.type,
                    stats: existing.stats || "-",
                    desc:  entry.description || existing.desc
                };
            }
            return entry.id;
        }
        return String(entry);
    },

    getItemMeta(id) {
        if (ASSETS.items[id]) return ASSETS.items[id];
        const cleanName = String(id).replace(/^item\./, "").replace(/_/g, " ");
        return {
            name: cleanName.charAt(0).toUpperCase() + cleanName.slice(1),
            icon: ASSETS.items.default.icon,
            img: "",
            type: "common",
            desc: ASSETS.items.default.desc
        };
    },

    getNpcMeta(keyOrName) {
        if (ASSETS.npcs[keyOrName]) return ASSETS.npcs[keyOrName];
        const lower = String(keyOrName).toLowerCase();
        for (const k in ASSETS.npcs) {
            const npc = ASSETS.npcs[k];
            if (npc.name.toLowerCase() === lower || (npc.aliases && npc.aliases.includes(lower))) {
                return npc;
            }
        }
        return {
            id: keyOrName,
            name: String(keyOrName),
            icon: ASSETS.npcs.default.icon,
            img: "",
            iconImg: "",
            role: ASSETS.npcs.default.role,
            hostile: false
        };
    }
};