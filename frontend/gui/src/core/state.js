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
        id: "loc.central_harbor_square",
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
    worldRooms: {},
    worldQuests: {},
    selectedInventoryItem: null,

    currentGroup: null,
    groupInvitations: [],
    knownCombats: {},
    fledCombatIds: {},
    lastJoinCombatId: null,
    lastAttackEnemyId: null,

    serverPlayerCount: 1,
    activeChatScope: "GLOBAL",
    commandQueue: [],

    chats: {
        GLOBAL: [],
        ROOM: [],
        GROUP: [],
        EVENTS: [],
        LOG: []
    },

    resetCombatTracking() {
        this.knownCombats = {};
        this.fledCombatIds = {};
        this.lastJoinCombatId = null;
        this.lastAttackEnemyId = null;
    },

    registerKnownCombat(combatId, playerName, roomId, { enemyId = null, viaParty = false, helpRequested = false } = {}) {
        if (!combatId || !playerName) return;
        const lowerPlayer = playerName.toLowerCase();

        // One player can only be in one active combat at a time: drop older combats by this player
        Object.keys(this.knownCombats).forEach(cid => {
            if (cid !== combatId && this.knownCombats[cid].player.toLowerCase() === lowerPlayer) {
                delete this.knownCombats[cid];
            }
        });

        const prev = this.knownCombats[combatId] || {};
        this.knownCombats[combatId] = {
            id: combatId,
            player: playerName,
            room: roomId || prev.room || null,
            enemyId: enemyId || prev.enemyId || null,
            viaParty: !!(prev.viaParty || viaParty),
            helpRequested: !!(prev.helpRequested || helpRequested)
        };
    },

    removeKnownCombat(combatId) {
        if (combatId && this.knownCombats[combatId]) {
            delete this.knownCombats[combatId];
        }
    },

    clearPartyCombats() {
        Object.keys(this.knownCombats).forEach(cid => {
            const c = this.knownCombats[cid];
            if (c.helpRequested) {
                c.viaParty = false;
            } else if (c.viaParty) {
                delete this.knownCombats[cid];
            }
        });
    },

    pruneNonPartyCombats(newRoomId) {
        Object.keys(this.knownCombats).forEach(cid => {
            const c = this.knownCombats[cid];
            if (!c.viaParty && c.room && c.room !== newRoomId) {
                delete this.knownCombats[cid];
            }
        });
    },

    pruneCombatsInRoom(roomId) {
        if (!roomId) return;
        Object.keys(this.knownCombats).forEach(cid => {
            if (this.knownCombats[cid].room === roomId) {
                delete this.knownCombats[cid];
            }
        });
    },

    // Finds a joinable combat in the CURRENT room matching targetEnemyId
    findJoinableCombatInRoom(targetEnemyId) {
        const curRoom = this.room && this.room.id;
        const roomPlayersLower = (this.playersInRoom || []).map(p => p.toLowerCase());
        const myNameLower = (this.playerName || '').toLowerCase();
        const targetLower = targetEnemyId ? String(targetEnemyId).toLowerCase() : null;

        const candidates = Object.values(this.knownCombats).filter(c => {
            if (!c || !c.id) return false;
            if (this.fledCombatIds && this.fledCombatIds[c.id]) return false;
            if (c.player.toLowerCase() === myNameLower) return false;
            if (!c.viaParty && !c.helpRequested) return false;
            if (c.room && curRoom && c.room !== curRoom) return false;
            if (!roomPlayersLower.includes(c.player.toLowerCase())) return false;

            // Reject if combat is explicitly against a DIFFERENT NPC in the same room
            if (c.enemyId && targetLower && c.enemyId.toLowerCase() !== targetLower) {
                return false;
            }
            return true;
        });

        if (candidates.length === 0) return null;

        // Prioritize exact enemyId match over unknown-enemyId (e.g. non-party HELP REQUESTED)
        if (targetLower) {
            const exact = candidates.find(c => c.enemyId && c.enemyId.toLowerCase() === targetLower);
            if (exact) return exact;
        }
        return candidates[0];
    },

    // Checks whether we should render the "Aid" button next to a player in our room
    getAidCombatForPlayer(playerName) {
        if (this.inCombat || !playerName) return null;
        const lower = playerName.toLowerCase();
        if (lower === (this.playerName || '').toLowerCase()) return null;

        const curRoom = this.room && this.room.id;
        const inSameRoom = (this.playersInRoom || []).some(p => p.toLowerCase() === lower);
        if (!inSameRoom) return null;

        return Object.values(this.knownCombats).find(c => {
            if (!c || c.player.toLowerCase() !== lower) return false;
            if (this.fledCombatIds && this.fledCombatIds[c.id]) return false;
            if (!c.viaParty && !c.helpRequested) return false;
            if (c.room && curRoom && c.room !== curRoom) return false;
            return true;
        }) || null;
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
                    hp: cs.hp !== undefined ? cs.hp : (existing.hp !== undefined ? existing.hp : 60),
                    maxHp:    cs.max_hp || cs.hp || existing.maxHp || 60,
                    atk:      cs.attack || existing.atk || 8,
                    def:      cs.defense !== undefined ? cs.defense : (existing.def || 5),
                    spd:      cs.speed || existing.spd || 10,
                    loot:     rewardNames.length > 0 ? rewardNames.join(', ') : (npc.hostile ? 'None' : '-')
                };
            });
        }

        if (worldData.rooms) {
            this.worldRooms = worldData.rooms;
        }
        if (worldData.quests) {
            this.worldQuests = worldData.quests;
        }
    },

    normalizeItem(entry) {
        if (typeof entry === 'object' && entry !== null && entry.id) {
            const existing = ASSETS.items[entry.id] || ASSETS.items.default;
            const merged = {
                ...existing,
                name:    entry.name || existing.name || entry.id,
                type:    entry.type || existing.type || "common",
                desc:    entry.description || existing.desc || "",
                attack:  entry.attack  !== undefined ? entry.attack  : (existing.attack  || 0),
                defense: entry.defense !== undefined ? entry.defense : (existing.defense || 0),
                speed:   entry.speed   !== undefined ? entry.speed   : (existing.speed   || 0),
                hp:      entry.hp      !== undefined ? entry.hp      : (existing.hp      || 0)
            };
            merged.stats = this.formatItemStats(merged);
            ASSETS.items[entry.id] = merged;

            if (typeof StatusModal !== 'undefined' && (merged.type === 'equipment' || merged.type === 'unique')) {
                if (merged.attack || merged.defense || merged.speed) {
                    StatusModal.gearBonuses[entry.id] = {
                        atk: merged.attack || 0,
                        def: merged.defense || 0,
                        spd: merged.speed || 0
                    };
                }
            }
            return entry.id;
        }
        return String(entry);
    },

    getRoomMeta(roomId) {
        const id = String(roomId || '');
        const visual = ASSETS.rooms[id] || ASSETS.rooms[`loc.${id}`] || ASSETS.rooms.default;
        const worldDef = (this.worldRooms && (this.worldRooms[id] || this.worldRooms[`loc.${id}`])) || null;
        const cleanFallback = id.replace(/^[^.]+\./, '').replace(/_/g, ' ');
        const capFallback = cleanFallback ? cleanFallback.charAt(0).toUpperCase() + cleanFallback.slice(1) : 'Unknown';
        return {
            ...ASSETS.rooms.default,
            ...visual,
            id,
            name: (worldDef && worldDef.name) || capFallback
        };
    },

    getItemMeta(id) {
        if (ASSETS.items[id] && ASSETS.items[id].name) return ASSETS.items[id];
        const existing = ASSETS.items[id] || ASSETS.items.default;
        const cleanName = String(id).replace(/^item\./, "").replace(/_/g, " ");
        return {
            name:    cleanName.charAt(0).toUpperCase() + cleanName.slice(1),
            icon:    existing.icon || ASSETS.items.default.icon,
            img:     existing.img || ASSETS.items.default.img,
            iconImg: existing.iconImg || existing.img || ASSETS.items.default.iconImg,
            type:    existing.type || "common",
            stats:   existing.stats || "Utility",
            desc:    existing.desc || ASSETS.items.default.desc
        };
    },

    normalizeNpc(entry) {
        if (typeof entry === 'object' && entry !== null && entry.id) {
            const existing = ASSETS.npcs[entry.id] || ASSETS.npcs.default;
            const cs = entry.combat_stats || {};
            ASSETS.npcs[entry.id] = {
                ...existing,
                id:      entry.id,
                name:    entry.name || existing.name || entry.id,
                role:    entry.role ? (entry.role.charAt(0).toUpperCase() + entry.role.slice(1).replace(/_/g, ' ')) : existing.role,
                hostile: entry.hostile !== undefined ? !!entry.hostile : !!existing.hostile,
                hp:      cs.hp !== undefined ? cs.hp : (existing.hp !== undefined ? existing.hp : 60),
                maxHp:   cs.max_hp || cs.hp || existing.maxHp || 60,
                atk:     cs.attack || existing.atk || 10,
                def:     cs.defense !== undefined ? cs.defense : (existing.def || 5),
                spd:     cs.speed || existing.spd || 10
            };
            return entry.id;
        }
        return String(entry);
    },

    getNpcMeta(keyOrName) {
        if (typeof keyOrName === 'object' && keyOrName !== null) {
            keyOrName = this.normalizeNpc(keyOrName);
        }
        const direct = ASSETS.npcs[keyOrName] || ASSETS.npcs[`npc.${keyOrName}`];
        if (direct && direct.name) {
            return direct;
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