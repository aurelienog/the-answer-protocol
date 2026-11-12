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
    worldQuests: {},
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

    formatItemStats(itemObj) {
        const parts = [];
        if (itemObj.attack)  parts.push(`+${itemObj.attack} Atk`);
        if (itemObj.defense) parts.push(`+${itemObj.defense} Def`);
        if (itemObj.speed)   parts.push(`+${itemObj.speed} Spd`);
        if (itemObj.hp)      parts.push(`Restores ${itemObj.hp} HP`);
        if (parts.length > 0) return parts.join(', ');

        const t = (itemObj.type || 'common').toLowerCase();
        if (t === 'quest') return 'Quest Item';
        if (t === 'unique') return 'Trophy';
        return 'Utility';
    },

    syncWorldDefinition(worldData) {
        if (!worldData) return;

        if (worldData.items) {
            if (typeof StatusModal !== 'undefined') {
                StatusModal.gearBonuses = {};
            }
            Object.values(worldData.items).forEach(it => {
                this.normalizeItem(it);
            });
        }

        if (worldData.npcs) {
            Object.values(worldData.npcs).forEach(npc => {
                const existing = ASSETS.npcs[npc.id] || ASSETS.npcs.default;
                const cs = npc.combat_stats || {};
                const rewardNames = (npc.rewards || []).map(rId => {
                    const rMeta = this.getItemMeta(rId);
                    return rMeta ? rMeta.name : rId;
                });

                ASSETS.npcs[npc.id] = {
                    ...existing,
                    id:       npc.id,
                    name:     npc.name || existing.name || npc.id,
                    role:     npc.role ? (npc.role.charAt(0).toUpperCase() + npc.role.slice(1).replace(/_/g, ' ')) : existing.role,
                    hostile:  !!npc.hostile,
                    hasQuest: !!npc.quest_id,
                    hp:       cs.hp || existing.hp || 60,
                    maxHp:    cs.max_hp || cs.hp || existing.maxHp || 60,
                    atk:      cs.attack || existing.atk || 8,
                    def:      cs.defense !== undefined ? cs.defense : (existing.def || 5),
                    spd:      cs.speed || existing.spd || 10,
                    loot:     rewardNames.length > 0 ? rewardNames.join(', ') : (npc.hostile ? 'None' : '-')
                };
            });
        }

        if (worldData.quests) {
            this.worldQuests = worldData.quests;
        }
    },

    normalizeItem(entry) {
        if (typeof entry === 'object' && entry !== null && entry.id) {
            const existing = ASSETS.items[entry.id] || ASSETS.items.default;
            ASSETS.items[entry.id] = {
                ...existing,
                name:    entry.name || existing.name || entry.id,
                type:    entry.type || existing.type || "common",
                desc:    entry.description || existing.desc || "",
                attack:  entry.attack || 0,
                defense: entry.defense || 0,
                speed:   entry.speed || 0,
                hp:      entry.hp || 0,
                stats:   this.formatItemStats(entry)
            };

            if (typeof StatusModal !== 'undefined' && (entry.type === 'equipment' || entry.type === 'unique')) {
                if (entry.attack || entry.defense || entry.speed) {
                    StatusModal.gearBonuses[entry.id] = {
                        atk: entry.attack || 0,
                        def: entry.defense || 0,
                        spd: entry.speed || 0
                    };
                }
            }
            return entry.id;
        }
        return String(entry);
    },

    getItemMeta(id) {
        if (ASSETS.items[id] && ASSETS.items[id].name) return ASSETS.items[id];
        const existing = ASSETS.items[id] || ASSETS.items.default;
        const cleanName = String(id).replace(/^item\./, "").replace(/_/g, " ");
        return {
            name: cleanName.charAt(0).toUpperCase() + cleanName.slice(1),
            icon: existing.icon || ASSETS.items.default.icon,
            img:  existing.img || "",
            type: existing.type || "common",
            stats: existing.stats || "Utility",
            desc: existing.desc || ASSETS.items.default.desc
        };
    },

    getNpcMeta(keyOrName) {
        if (ASSETS.npcs[keyOrName] && ASSETS.npcs[keyOrName].name) {
            return ASSETS.npcs[keyOrName];
        }
        const lower = String(keyOrName).toLowerCase();
        for (const k in ASSETS.npcs) {
            const npc = ASSETS.npcs[k];
            if ((npc.name && npc.name.toLowerCase() === lower) || (npc.aliases && npc.aliases.includes(lower))) {
                return npc;
            }
        }
        const fallback = ASSETS.npcs[keyOrName] || ASSETS.npcs.default;
        return {
            ...ASSETS.npcs.default,
            ...fallback,
            id: keyOrName,
            name: fallback.name || String(keyOrName)
        };
    }
};