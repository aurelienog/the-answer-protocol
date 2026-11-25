// ==========================================================================
// CORE MODULE: NETWORK CLIENT & TAP PROTOCOL PARSER
// ==========================================================================

const App = {
    eventSource: null,
    worldMetaLoaded: false,

    async loadWorldMeta() {
        if (this.worldMetaLoaded) return;
        try {
            const res = await fetch('/api/world');
            if (res.ok) {
                const worldData = await res.json();
                GameState.syncWorldDefinition(worldData);
                this.worldMetaLoaded = true;
            }
        } catch (e) {
            console.warn("Could not pre-load world metadata:", e);
        }
    },

    async restoreSessionIfActive() {
        try {
            const cfgRes = await fetch('/api/config');
            if (cfgRes.ok) {
                const cfg = await cfgRes.json();
                const hostEl = document.getElementById('login-host');
                const portEl = document.getElementById('login-port');
                if (hostEl && cfg.host) hostEl.value = cfg.host;
                if (portEl && cfg.port) portEl.value = cfg.port;
            }
        } catch (_) {}

        await this.loadWorldMeta();
        await this.initStream();
        try {
            const res = await fetch('/api/session');
            const data = await res.json();
            if (data.connected && data.username) {
                GameState.connected = true;
                GameState.playerName = data.username;
                UI.enterGameScreen();
                UI.addChatLine('LOG', 'System', `Session restored as ${data.username}.`, 'chat-sys');

                const savedEnemy = sessionStorage.getItem('tap_active_combat_enemy');
                if (savedEnemy && savedEnemy !== 'default') {
                    CombatModal.open(savedEnemy, undefined, undefined, true);
                }

                await this.sendCommand('LOOK');
                await this.sendCommand('STATUS');
                await this.sendCommand('WHO');
                await this.sendCommand('INVENTORY');
                await this.sendCommand('QUESTS');
                await this.sendCommand('GROUP INFO');
            }
        } catch (_) {}
    },

    async connect() {
        const username = document.getElementById('login-username').value.trim();
        const host = document.getElementById('login-host').value.trim();
        const port = document.getElementById('login-port').value.trim();

        if (!username) {
            UI.showLoginError("Please enter a character name.");
            return;
        }

        UI.showLoginError("");
        await this.loadWorldMeta();
        if (!this.eventSource) {
            await this.initStream();
        }

        try {
            const res = await fetch('/api/connect', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ host, port, username })
            });
            const data = await res.json();

            if (data.status !== 'OK') {
                UI.showLoginError(data.error || "Connection failed");
                return;
            }

            GameState.connected = true;
            GameState.playerName = username;
            GameState.hp = 100;
            GameState.maxHp = 100;
            GameState.status = "healthy";
            GameState.currentGroup = null;
            GameState.groupInvitations = [];
            GameState.inventory = [];
            GameState.inventoryCounts = {};
            GameState.quests = [];
            GameState.resetCombatTracking();
            GameState.chats = {
                ROOM: [],
                GLOBAL: [],
                GROUP: [],
                COMBAT: [],
                LOG: []
            };
            UI.updateHUDStats();
            ChatView.renderChatMessages();
            UI.enterGameScreen();
            UI.addChatLine('LOG', 'System', `Connected to ${host}:${port} as ${username}.`, 'chat-sys');

            await this.sendCommand('LOOK');
            await this.sendCommand('WHO');
            await this.sendCommand('INVENTORY');
            await this.sendCommand('QUESTS');
        } catch (err) {
            UI.showLoginError("Network error: " + err.message);
        }
    },

    initStream() {
        return new Promise((resolve) => {
            if (this.eventSource) {
                this.eventSource.close();
            }
            let settled = false;
            const finish = () => {
                if (!settled) {
                    settled = true;
                    resolve();
                }
            };

            this.eventSource = new EventSource('/api/stream');
            this.eventSource.onopen = () => finish();
            setTimeout(finish, 300);

            this.eventSource.onmessage = (e) => {
                try {
                    const packet = JSON.parse(e.data);
                    this.handleServerPacket(packet);
                } catch (err) {
                    console.error("Failed to parse server event:", err);
                }
            };
            this.eventSource.onerror = async () => {
                finish();
                try {
                    await fetch('/api/session');
                } catch (_) {
                    window.close();
                }
            };
        });
    },

    async sendCommand(rawCommand) {
        const cmd = rawCommand.trim();
        if (!cmd) return;

        const upper = cmd.toUpperCase();
        if (GameState.inCombat && upper.startsWith('MOVE ')) {
            UI.addChatLine('COMBAT', '⚔️ In Combat', 'You cannot leave the room during combat! Use "🏃 Flee" to retreat.', 'chat-err');
            return;
        }

        let verb = upper.split(/\s+/)[0];
        if (upper.startsWith('GROUP CREATE'))      verb = 'GROUP_CREATE';
        if (upper.startsWith('GROUP JOIN'))        verb = 'GROUP_JOIN';
        if (upper.startsWith('GROUP INVITE '))     verb = 'GROUP_INVITE';
        if (upper.startsWith('GROUP LEAVE'))       verb = 'GROUP_LEAVE';
        if (upper.startsWith('GROUP INFO'))        verb = 'GROUP_INFO';
        if (upper.startsWith('GROUP INVITATIONS')) verb = 'GROUP_INVITATIONS';
        if (upper.startsWith('COMBAT DEFEND'))     verb = 'COMBAT_DEFEND';
        if (upper.startsWith('COMBAT HELP'))       verb = 'COMBAT_HELP';
        if (upper.startsWith('COMBAT JOIN')) {
            verb = 'COMBAT_JOIN';
            const cmdParts = cmd.split(/\s+/);
            GameState.lastJoinCombatId = cmdParts[2] || null;
        }
        if (verb === 'ATTACK') {
            const rawTarget = cmd.slice(6).trim();
            const targetMeta = GameState.getNpcMeta(rawTarget);
            GameState.lastAttackEnemyId = (targetMeta && targetMeta.id !== 'default') ? targetMeta.id : rawTarget;
        }
        GameState.commandQueue.push(verb);

        UI.addChatLine('LOG', 'Client →', cmd, '');

        try {
            const res = await fetch('/api/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            });
            const data = await res.json();
            if (data.status !== 'OK') {
                GameState.commandQueue.pop();
                UI.addChatLine('LOG', 'Error', data.error || 'Failed to send command', 'chat-err');
            }
        } catch (err) {
            GameState.commandQueue.pop();
            UI.addChatLine('LOG', 'Error', 'Failed to send command', 'chat-err');
        }
    },

    sendChatMessage() {
        const input = document.getElementById('chat-input');
        const text = input.value.trim();
        if (!text) return;
        input.value = '';

        if (GameState.activeChatScope === 'LOG') {
            this.sendCommand(text);
            return;
        }

        const scope = GameState.activeChatScope === 'COMBAT' ? 'ROOM' : GameState.activeChatScope;
        this.sendCommand(`CHAT ${scope} ${text}`);
    },

    formatFriendlyError(raw, failedVerb = '') {
        if (raw.includes('NO_EXIT'))                 return ['🧭 No Exit', 'There is no passage in that direction. Check the compass for available routes!'];
        if (raw.includes('NPC_NOT_HOSTILE'))         return ['🛡️ Friendly NPC', 'This character is peaceful and cannot be attacked.'];
        if (raw.includes('NPC_RESPAWNING'))          return ['⏳ Respawning', 'This enemy was recently defeated and is recovering. Try again shortly!'];
        if (raw.includes('NPC_ALREADY_IN_COMBAT'))   return ['⚔️ Busy in Combat', 'This enemy is already fighting another player! Wait for them to press "📯 Help" to join!'];
        if (raw.includes('COMBAT_HELP_REQUIRED'))    return ['📯 Help Not Requested', 'You can only join an ongoing battle if the player calls for help (📯 Help) or is in your party.'];
        if (raw.includes('COMBAT_NOT_FOUND'))        return ['⚔️ Combat Ended', 'That battle has already concluded.'];
        if (raw.includes('NOT_YOUR_TURN'))           return ['⏳ Wait Your Turn', 'It is not your turn to act yet!'];
        if (raw.includes('NOT_IN_COMBAT_ROOM'))      return ['🧭 Wrong Location', 'You must be in the same room as the battle to join it.'];
        if (raw.includes('NOT_IN_COMBAT'))           return ['⚔️ Combat Ended', 'You are no longer in an active combat.'];
        if (failedVerb === 'COMBAT_JOIN' && raw.includes('INTERNAL_ERROR')) {
            return ['⚔️ Combat Ended', 'That battle has already concluded.'];
        }
        if (raw.includes('ALREADY_IN_COMBAT'))       return ['⚔️ Cannot Rejoin', 'You have already participated in or fled from this battle and cannot rejoin it.'];
        if (raw.includes('ITEM_ALREADY_OWNED'))      return ['🎒 Already Owned', 'You already carry this item in your inventory.'];
        if (raw.includes('ITEM_CANNOT_BE_DROPPED'))  return ['🔒 Bound Item', 'Equipment, Unique, and Quest items are bound to you and cannot be dropped.'];
        if (raw.includes('ITEM_NOT_CONSUMABLE'))     return ['🧪 Not Consumable', 'This item cannot be consumed directly.'];
        if (raw.includes('ITEM_HAS_NO_EFFECT'))      return ['❤️ Full Health', 'You are already at maximum health!'];
        if (raw.includes('AMBIGUOUS_ITEM_NAME'))     return ['🔍 Ambiguous Item', 'Multiple items match that name. Please use the exact item ID.'];
        if (raw.includes('REQUIRED_ITEM_MISSING'))   return ['📜 Missing Objective', 'You do not have the required quest item in your bag yet.'];
        if (raw.includes('QUEST_GIVER_NOT_AVAILABLE')) return ['📜 Quest Giver Away', 'You must be in the same room as the quest giver to complete this quest.'];
        if (raw.includes('NO_QUEST_AVAILABLE'))      return ['📜 No Quest Available', 'This character has no new quests for you right now.'];
        if (raw.includes('ALREADY_IN_GROUP')) {
            if (failedVerb === 'GROUP_INVITE') {
                return ['👥 Player Unavailable', 'That player is already a member of another party and cannot be invited.'];
            }
            return ['👥 Already in Party', 'You are already in a party! Leave your current party first before creating or joining another.'];
        }
        if (raw.includes('GROUP_ALREADY_EXISTS'))    return ['👥 Name Taken', 'A party with that name already exists. Choose another name!'];
        if (raw.includes('RATE_LIMITED'))            return ['⏳ Slow Down', 'You are performing actions too quickly! Wait a moment.'];
        if (raw.includes('COMMAND_FAILED')) {
            if (failedVerb === 'GROUP_INVITE') {
                return ['👥 Already Invited', 'You have already invited this player (or the player is not online).'];
            }
            if (failedVerb === 'GROUP_JOIN') {
                return ['👥 Cannot Join Party', 'Party not found or you do not have a pending invitation to join it.'];
            }
        }
        return ['⚠️ Notice', raw.replace(/^ERR\s+\d+\s+/, '').replace(/_/g, ' ')];
    },

    handleServerPacket(packet) {
        const raw = packet.raw;

        if (packet.type === 'DISCONNECTED') {
            UI.addChatLine('LOG', 'System', 'Disconnected from server.', 'chat-err');
            UI.returnToLogin('Disconnected from server.');
            return;
        }

        if (packet.type === 'ERR') {
            UI.addChatLine('LOG', 'Server ←', raw, 'chat-err');
            const failedVerb = GameState.commandQueue.shift();

            if (failedVerb === 'COMBAT_DEFEND') {
                CombatModal.cancelDefendAnim();
            }

            // 1. Auto-join ONLY if there is a valid party/help combat for THIS enemy in OUR CURRENT ROOM
            if (raw.includes('NPC_ALREADY_IN_COMBAT')) {
                const joinTarget = GameState.findJoinableCombatInRoom(GameState.lastAttackEnemyId);
                if (joinTarget) {
                    this.sendCommand(`COMBAT JOIN ${joinTarget.id}`);
                    return;
                }
            }

            // 2. Clean up stale combat entry if COMBAT JOIN failed
            if (failedVerb === 'COMBAT_JOIN' && GameState.lastJoinCombatId) {
                const failedCid = GameState.lastJoinCombatId;
                GameState.lastJoinCombatId = null;
                if (raw.includes('ALREADY_IN_COMBAT')) {
                    GameState.fledCombatIds[failedCid] = true;
                    UI.renderRoom();
                } else if (
                    raw.includes('COMBAT_NOT_FOUND') ||
                    raw.includes('COMBAT_HELP_REQUIRED') ||
                    raw.includes('INTERNAL_ERROR')
                ) {
                    GameState.removeKnownCombat(failedCid);
                    UI.renderRoom();
                }
            }

            if (raw.includes('NOT_IN_GROUP')) {
                GroupModal.renderCurrentGroup(null);
                if (failedVerb === 'GROUP_INFO') return;
            }

            if (raw.includes('PLAYER_ALREADY_IN_COMBAT')) {
                const enemyInRoom = GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile) || CombatModal.enemyId || 'npc.dire_wolf';
                CombatModal.open(enemyInRoom, GameState.hp, undefined, true);
                return;
            }

            if (raw.includes('NOT_IN_COMBAT') && !raw.includes('NOT_IN_COMBAT_ROOM') && GameState.inCombat) {
                if (CombatModal.combatId) {
                    GameState.removeKnownCombat(CombatModal.combatId);
                }
                CombatModal.close();
                this.sendCommand('LOOK');
                this.sendCommand('STATUS');
            } else if (GameState.inCombat) {
                CombatModal.syncActionButtonsState();
            }

            const [title, friendlyMsg] = this.formatFriendlyError(raw, failedVerb);
            const isCombatErr = GameState.inCombat ||
                raw.includes('COMBAT') ||
                raw.includes('NOT_YOUR_TURN') ||
                raw.includes('NPC_NOT_HOSTILE') ||
                raw.includes('NPC_RESPAWNING');
            UI.addChatLine(isCombatErr ? 'COMBAT' : 'ROOM', title, friendlyMsg, 'chat-err');
            return;
        }

        if (packet.type === 'EVT') {
            UI.addChatLine('LOG', 'Event ←', raw, 'chat-sys');
            this.handleEventLine(raw);
            return;
        }

        if (packet.type === 'OK') {
            UI.addChatLine('LOG', 'Server ←', raw, 'chat-sys');
            this.handleOKLine(raw);
        }
    },

    handleOKLine(raw) {
        const lastVerb = GameState.commandQueue.shift() || '';
        const payload = raw.slice(2).trim();

        if (lastVerb === 'GROUP_LEAVE') {
            GameState.clearPartyCombats();
            GroupModal.renderCurrentGroup(null);
            UI.addChatLine('ROOM', '👥 Group', 'You left the group.', 'chat-sys');
            return;
        }
        if (lastVerb === 'GROUP_INVITE') {
            UI.addChatLine('ROOM', '👥 Group', 'Party invitation sent!', 'chat-sys');
            this.sendCommand('GROUP INFO');
            return;
        }
        if (payload === 'bye') {
            GameState.resetCombatTracking();
            UI.returnToLogin('');
            return;
        }
        if (payload.startsWith('room=')) {
            const newRoom = payload.split('=')[1];
            GameState.pruneNonPartyCombats(newRoom);
            UI.closeAllModals();
            UI.addChatLine('ROOM', '🧭 Travel', `You moved to ${newRoom}.`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }
        if (payload.startsWith('taken=')) {
            const itemId = payload.split('=')[1];
            const meta = GameState.getItemMeta(itemId);
            UI.addChatLine('ROOM', '✋ Item', `Picked up ${meta.name}.`, 'chat-sys');
            this.sendCommand('INVENTORY');
            this.sendCommand('LOOK');
            return;
        }
        if (payload.startsWith('dropped=')) {
            const itemId = payload.split('=')[1];
            const meta = GameState.getItemMeta(itemId);
            UI.addChatLine('ROOM', '🎒 Item', `Dropped ${meta.name}.`, 'chat-sys');
            this.sendCommand('INVENTORY');
            this.sendCommand('LOOK');
            return;
        }
        if (payload.startsWith('used=')) {
            const itemMatch = payload.match(/used=([^\s]+)/);
            const hpMatch = payload.match(/hp=(\d+)/);
            const itemId = itemMatch ? itemMatch[1] : '';
            const healed = hpMatch ? parseInt(hpMatch[1], 10) : 0;
            const meta = GameState.getItemMeta(itemId);

            if (GameState.inCombat && healed > 0) {
                CombatModal.playHealEffect(healed);
            } else if (healed > 0) {
                GameState.hp = Math.min(GameState.maxHp, GameState.hp + healed);
                UI.updateHUDStats();
            }
            UI.addChatLine(GameState.inCombat ? 'COMBAT' : 'ROOM', '🧪 Used Item', `Used ${meta.name} (+${healed} HP)!`, 'chat-sys');
            this.sendCommand('INVENTORY');
            return;
        }
        if (payload.startsWith('defended=')) {
            const redMatch = payload.match(/(?:reduction|defense_bonus)=(\d+)/);
            const pct = parseInt(redMatch ? redMatch[1] : '50', 10);
            UI.addChatLine('COMBAT', '🛡️ Defend', `Defensive stance active (+${pct}% defense on next hit).`, 'chat-sys');
            return;
        }
        if (payload.startsWith('fled=')) {
            const hpMatch = payload.match(/hp=(\d+)/);
            const dmgMatch = payload.match(/damage=(\d+)/);
            const cMatch = payload.match(/combat=([^\s]+)/);
            const fledCombatId = (cMatch && cMatch[1]) || CombatModal.combatId;
            if (fledCombatId) {
                GameState.fledCombatIds[fledCombatId] = true;
            }
            if (hpMatch) GameState.hp = parseInt(hpMatch[1], 10);
            UI.updateHUDStats();
            CombatModal.close();
            UI.addChatLine('COMBAT', '🏃 Retreated', `You fled from combat (took ${dmgMatch ? dmgMatch[1] : 0} escape penalty damage).`, 'chat-err');
            this.sendCommand('LOOK');
            return;
        }
        if (payload.startsWith('help_requested=') || (lastVerb === 'COMBAT_HELP' && payload.startsWith('combat='))) {
            UI.addChatLine('COMBAT', '📯 Call for Help', 'Your call for help echoed through the room!', 'chat-sys');
            return;
        }
        if (payload.startsWith('joined=') || (lastVerb === 'COMBAT_JOIN' && payload.startsWith('combat='))) {
            const cMatch = payload.match(/combat=([^\s]+)/);
            const joinedCombatId = (cMatch && cMatch[1]) || GameState.lastJoinCombatId;
            const known = joinedCombatId ? GameState.knownCombats[joinedCombatId] : null;
            const caller = (known && known.player) || CombatModal.activeTurnOwner || 'Ally';

            GameState.lastJoinCombatId = null;
            if (joinedCombatId) {
                CombatModal.combatId = joinedCombatId;
            }
            if (!GameState.inCombat) {
                const thpMatch = payload.match(/target_hp=(\d+)/);
                const currentEnemyHp = thpMatch ? parseInt(thpMatch[1], 10) : undefined;
                const enemyInRoom = (known && known.enemyId)
                    || GameState.lastAttackEnemyId
                    || GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile)
                    || GameState.npcsInRoom[0]
                    || 'npc.dire_wolf';
                sessionStorage.setItem('tap_combat_turn_owner', caller);
                CombatModal.open(enemyInRoom, GameState.hp, currentEnemyHp, false, true);
                CombatModal.setInitialParticipants([caller]);
                CombatModal.addJoinedParticipant(GameState.playerName);
                CombatModal.queueTurnBanner(caller, undefined);
            }
            UI.addChatLine('COMBAT', '⚔️ Joined Battle', `You joined the battle against ${CombatModal.enemyName}! You will act on the next round!`, 'chat-sys');
            UI.renderRoom();
            return;
        }
        if (payload.startsWith('target=')) {
            return;
        }
        if (payload.startsWith('players=')) {
            GameState.serverPlayerCount = parseInt(payload.split('=')[1], 10) || 1;
            UI.updateHUDStats();
            return;
        }
        if (payload.startsWith('group=')) {
            GameState.groupInvitations = [];
            UI.addChatLine('ROOM', '👥 Group', `Group updated (${payload})`, 'chat-sys');
            this.sendCommand('GROUP INFO');
            return;
        }

        if (payload.startsWith('{') || payload.startsWith('[')) {
            try {
                const data = JSON.parse(payload);

                if (data.room && data.room.id) {
                    GameState.room = data.room;
                    GameState.playersInRoom = Array.isArray(data.players) && data.players.length > 0
                        ? data.players
                        : [GameState.playerName];

                    const counts = {};
                    const uniqueIds = [];
                    (data.items || []).forEach(it => {
                        const id = GameState.normalizeItem(it);
                        counts[id] = (counts[id] || 0) + 1;
                        if (!uniqueIds.includes(id)) uniqueIds.push(id);
                    });
                    GameState.roomItemCounts = (data.room.items && typeof data.room.items === 'object' && !Array.isArray(data.room.items))
                        ? data.room.items
                        : counts;
                    GameState.itemsInRoom = uniqueIds;
                    GameState.npcsInRoom = (data.npcs || []).map(n => GameState.normalizeNpc(n));

                    if (GameState.inCombat) {
                        const hasHostileHere = GameState.npcsInRoom.some(n => GameState.getNpcMeta(n).hostile);
                        if (!hasHostileHere && CombatModal.isFallen) {
                            if (!CombatModal.outcomeShown) {
                                CombatModal.showOutcome(false, 'Your party was defeated! Respawned at Harbor Square.');
                                this.sendCommand('STATUS');
                            }
                        } else {
                            CombatModal.syncRoomBackground();
                        }
                    }

                    UI.renderRoom();
                    return;
                }

                if (data.npc && data.dialogue !== undefined) {
                    const lines = Array.isArray(data.dialogue) ? data.dialogue : [String(data.dialogue)];
                    const npcObj = typeof data.npc === 'object' ? data.npc : { id: String(data.npc), name: String(data.npc) };
                    DialogueModal.renderConversation({ ...data, npc: npcObj, dialogue: lines });
                    UI.addChatLine('ROOM', `💬 ${npcObj.name || npcObj.id}`, lines.join(' '));
                    return;
                }

                if (lastVerb === 'WHO' && data.server !== undefined) {
                    GameState.serverPlayerCount = parseInt(data.server, 10) || 1;
                    UI.updateHUDStats();
                    return;
                }

                if ((data.ID || data.id) && Array.isArray(data.Members || data.members)) {
                    GroupModal.renderCurrentGroup(data);
                    return;
                }

                if (data.id && data.name && data.description && !data.type) {
                    UI.addChatLine('ROOM', '📜 Quest Accepted', `${data.name}: ${data.description}`, 'chat-sys');
                    DialogueModal.onQuestAccepted(data);
                    this.sendCommand('QUESTS');
                    return;
                }

                if (data.quest_id && data.status === 'completed') {
                    UI.addChatLine('ROOM', '🏆 Quest Completed', `${data.message || 'Completed'}: ${data.quest_id}`, 'chat-sys');
                    if (DialogueModal.activeNpcId) this.sendCommand(`TALK ${DialogueModal.activeNpcId}`);
                    this.sendCommand('INVENTORY');
                    this.sendCommand('QUESTS');
                    return;
                }
                if (data.quest_id && data.status) {
                    UI.addChatLine('ROOM', '📜 Quest', `${data.quest_id}: ${data.description || data.status}`, 'chat-sys');
                    this.sendCommand('QUESTS');
                    return;
                }

                if (Array.isArray(data)) {
                    if (lastVerb === 'GROUP' || lastVerb === 'GROUP_INVITATIONS') {
                        GroupModal.renderInvitations(data);
                    } else if (lastVerb === 'QUEST' || lastVerb === 'QUESTS' || (data[0] && data[0].status)) {
                        GameState.quests = data;
                        UI.renderQuests();
                    } else {
                        const invIds = [];
                        const invCounts = {};
                        data.forEach(it => {
                            const id = GameState.normalizeItem(it);
                            const qty = (typeof it === 'object' && it !== null && it.quantity) ? it.quantity : 1;
                            if (!invIds.includes(id)) invIds.push(id);
                            invCounts[id] = qty;
                        });
                        GameState.inventory = invIds;
                        GameState.inventoryCounts = invCounts;
                        if (GameState.selectedInventoryItem && !invIds.includes(GameState.selectedInventoryItem)) {
                            GameState.selectedInventoryItem = null;
                            document.getElementById('btn-drop-selected').disabled = true;
                            document.getElementById('btn-use-selected').disabled = true;
                        }
                        UI.renderInventory();
                    }
                    return;
                }

                if (data.hp !== undefined && data.max_hp !== undefined) {
                    GameState.hp = data.hp;
                    GameState.maxHp = data.max_hp;
                    GameState.status = data.status || 'healthy';
                    UI.updateHUDStats();

                    if (GameState.inCombat && CombatModal.isFallen && data.hp > 0) {
                        if (!CombatModal.outcomeShown) {
                            CombatModal.showOutcome(false, 'Combat ended! Respawned at Harbor Square.');
                            this.sendCommand('LOOK');
                        }
                        return;
                    }

                    if (GameState.inCombat) {
                        CombatModal.updateBars();
                    }
                    return;
                }

                if (data.combat_id || data.attacker_hp !== undefined) {
                    const startedEnemyId = CombatModal.enemyId || GameState.lastAttackEnemyId || 'npc.dire_wolf';
                    if (data.combat_id) {
                        CombatModal.combatId = data.combat_id;
                    }
                    if (data.combat_id && GameState.currentGroup) {
                        this.sendCommand(`CHAT GROUP [COMBAT_ID:${data.combat_id}|${GameState.room.id}|${startedEnemyId}]`);
                    }
                    if (!GameState.inCombat) {
                        CombatModal.open(startedEnemyId, data.attacker_hp, data.target_hp);
                    }
                    UI.addChatLine('COMBAT', '⚔️ Combat', `Battle started against ${CombatModal.enemyName}!`, 'chat-sys');
                    return;
                }

            } catch (e) {
                console.warn("Non-standard JSON payload:", payload);
            }
        }

        if (lastVerb === 'TALK' && payload) {
            const npcId = DialogueModal.activeNpcId || 'NPC';
            const meta = GameState.getNpcMeta(npcId);
            DialogueModal.renderConversation({
                npc: { id: meta.id, name: meta.name },
                dialogue: [payload]
            });
            UI.addChatLine('ROOM', `💬 ${meta.name}`, payload);
        }
    },

    handleEventLine(raw) {
        const parts = raw.split(' ');
        const category = parts[1];

        if (category === 'STATS' && parts[2] && parts[2].startsWith('players=')) {
            GameState.serverPlayerCount = parseInt(parts[2].split('=')[1], 10) || 1;
            UI.updateHUDStats();
            return;
        }

        if (category === 'ROOM' && parts[2] === 'PRESENCE') {
            const isEnter = parts[3] === 'ENTER';
            const user = parts.slice(4).join(' ');
            UI.addChatLine('ROOM', '👣 Presence', `${user} ${isEnter ? 'entered' : 'left'} the location.`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }
        if (category === 'ROOM_PRESENCE_ENTER' || category === 'ROOM_PRESENCE_LEAVE') {
            const user = parts[2];
            const isEnter = category === 'ROOM_PRESENCE_ENTER';
            UI.addChatLine('ROOM', '👣 Presence', `${user} ${isEnter ? 'entered' : 'left'} the location.`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }

        if ((category === 'ROOM' || category === 'GLOBAL' || category === 'GROUP') && parts[2] === 'CHAT') {
            const rest = parts.slice(3).join(' ');
            const kvMatch = rest.match(/^player=(\S+)\s+message=(.*)$/);
            const author = kvMatch ? kvMatch[1] : (parts[3] || 'Player');
            const msg = kvMatch ? kvMatch[2] : parts.slice(4).join(' ');

            const cidMatch = msg.match(/\[COMBAT_ID:(combat:\d+)(?:\|([^|\]]+))?(?:\|([^|\]]+))?\]/);
            if (cidMatch) {
                const newCombatId = cidMatch[1];
                const combatRoom = cidMatch[2] || null;
                const combatEnemy = cidMatch[3] || null;
                const alreadyFled = !!(GameState.fledCombatIds && GameState.fledCombatIds[newCombatId]);
                const isMyCurrentCombat = GameState.inCombat && CombatModal.combatId === newCombatId;

                if (!alreadyFled && !isMyCurrentCombat && author.toLowerCase() !== GameState.playerName.toLowerCase()) {
                    GameState.registerKnownCombat(newCombatId, author, combatRoom, {
                        enemyId: combatEnemy,
                        viaParty: true
                    });
                    UI.renderRoom();
                }
                UI.addChatLine('GROUP', '⚔️ Party Combat', `${author} started a battle (${newCombatId})! Travel to their room and click Attack or ⚔️ Aid to join!`, 'chat-sys');
                return;
            }

            UI.addChatLine(category, author, msg);
            return;
        }

        if (category === 'GROUP') {
            const action = parts[2];
            const details = parts.slice(3).join(' ');
            UI.addChatLine('GROUP', '👥 Party', `${action}: ${details}`, 'chat-sys');
            UI.addChatLine('ROOM', '👥 Party', `${action}: ${details}`, 'chat-sys');
            if (action === 'JOIN' && GameState.inCombat && CombatModal.combatId) {
                this.sendCommand(`CHAT GROUP [COMBAT_ID:${CombatModal.combatId}|${GameState.room.id}|${CombatModal.enemyId || ''}]`);
            }
            if (action === 'INVITE') {
                this.sendCommand('GROUP INVITATIONS');
            } else {
                this.sendCommand('GROUP INFO');
            }
            return;
        }

        if (category === 'ITEM' && parts[2] === 'DROPPED') {
            UI.addChatLine('ROOM', '🧰 World', `Items updated in room (${parts.slice(3).join(' ')}).`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }
        if (category === 'UNIQUE' && parts[2] === 'ITEMS' && parts[3] === 'RETURNED') {
            UI.addChatLine('ROOM', '🧰 World', `Items updated in room (${parts.slice(4).join(' ')}).`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }
        if (category === 'ITEM' && parts[2] === 'USED') {
            this.sendCommand('STATUS');
            return;
        }

        if (category === 'QUEST' && parts[2] === 'COMPLETED') {
            const rest = parts.slice(3).join(' ');
            const qMatch = rest.match(/quest=([^\s]+)/);
            const qId = qMatch ? qMatch[1] : '';
            if (qId) {
                const qDef = GameState.worldQuests && GameState.worldQuests[qId];
                const qName = (qDef && qDef.name) || qId;
                DialogueModal.announceQuestCompletion(qId, qName);
            }
            return;
        }

        if (category === 'NPC' && parts[2] === 'RESPAWN') {
            const rest = parts.slice(3).join(' ');
            const npcMatch = rest.match(/npc=([^\s]+)/);
            const nameMatch = rest.match(/name=(.+?)(?:\s+room=|$)/);
            const npcId = npcMatch ? npcMatch[1] : '';
            const npcName = (nameMatch && nameMatch[1].trim()) || (npcId ? GameState.getNpcMeta(npcId).name : 'An enemy');

            UI.addChatLine('ROOM', '⚔️ Respawn', `${npcName} has returned to the area!`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }

        if (category === 'RESPAWN') {
            const rest = parts.slice(2).join(' ');
            const pMatch = rest.match(/player=([^\s]+)/);
            const rMatch = rest.match(/room=([^\s]+)/);
            const hpMatch = rest.match(/hp=(\d+)/);
            const maxHpMatch = rest.match(/max_hp=(\d+)/);
            const respawnPlayer = pMatch ? pMatch[1] : '';

            if (respawnPlayer && respawnPlayer.toLowerCase() !== GameState.playerName.toLowerCase()) {
                return;
            }
            const respawnRoom = rMatch ? rMatch[1] : 'loc.central_harbor_square';
            const newHp = hpMatch ? parseInt(hpMatch[1], 10) : GameState.hp;
            const newMaxHp = maxHpMatch ? parseInt(maxHpMatch[1], 10) : GameState.maxHp;

            UI.addChatLine('ROOM', '🏥 Respawned', `You respawned in ${respawnRoom} with ${newHp}/${newMaxHp} HP.`, 'chat-sys');
            if (!GameState.inCombat) {
                GameState.hp = newHp;
                GameState.maxHp = newMaxHp;
                UI.updateHUDStats();
                this.sendCommand('LOOK');
            }
            return;
        }

        if (category === 'COMBAT') {
            const spacedSub = `${parts[2]} ${parts[3]}`;
            const spacedCombatEvents = ['PLAYER DIED', 'NPC DIED', 'PLAYER FLED', 'PLAYER JOINED', 'HELP REQUESTED'];
            const hasSpacedSub = spacedCombatEvents.includes(spacedSub);
            const sub = hasSpacedSub ? spacedSub : parts[2];
            const rest = parts.slice(hasSpacedSub ? 4 : 3).join(' ');

            if (sub === 'STARTED') {
                const cMatch = rest.match(/combat=([^\s]+)/);
                if (cMatch) {
                    CombatModal.combatId = cMatch[1];
                }
                const pIdx = rest.indexOf('participants=');
                let playerNames = [];
                let enemyInfo = null;

                if (pIdx !== -1) {
                    const rawParts = rest.slice(pIdx + 'participants='.length).trim();
                    if (rawParts.startsWith('[')) {
                        try {
                            const parsed = JSON.parse(rawParts);
                            if (Array.isArray(parsed)) {
                                playerNames = parsed.filter(p => p.is_player).map(p => p.name || p.id);
                                const npc = parsed.find(p => !p.is_player);
                                if (npc) {
                                    enemyInfo = { id: npc.id, name: npc.name, hp: npc.hp, maxHp: npc.max_hp };
                                }
                            }
                        } catch (e) {
                            console.warn('Failed to parse combat participants JSON:', e);
                        }
                    } else {
                        playerNames = rawParts.split(/\s+/)[0].split(',').filter(Boolean);
                    }
                }

                if (playerNames.length > 0) {
                    CombatModal.setInitialParticipants(playerNames);
                    sessionStorage.setItem('tap_combat_turn_owner', playerNames[0]);
                }

                const amIParticipant = playerNames.length === 0 ||
                    playerNames.some(p => p.toLowerCase() === GameState.playerName.toLowerCase());
                if (amIParticipant) {
                    if (!GameState.inCombat) {
                        const enemyId = (enemyInfo && enemyInfo.id) ||
                            GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile) ||
                            GameState.npcsInRoom[0] ||
                            'npc.dire_wolf';
                        CombatModal.open(
                            enemyId,
                            GameState.hp,
                            enemyInfo ? enemyInfo.hp : undefined,
                            false,
                            false,
                            enemyInfo ? enemyInfo.maxHp : undefined
                        );
                        UI.renderRoom();
                    } else if (enemyInfo && enemyInfo.id) {
                        CombatModal.syncEnemyFromServer(enemyInfo);
                    }
                }
                return;
            }

            if (sub === 'TURN') {
                const typeMatch = rest.match(/participant_type=([^\s]+)/);
                const roundMatch = rest.match(/round=(\d+)/);
                const partMatch = rest.match(/participant=(.+?)\s+round=/);
                const actionsMatch = rest.match(/actions=(\d+)/);
                const isNpcTurn = !!(typeMatch && typeMatch[1].toUpperCase() === 'NPC');
                const roundNum = roundMatch ? parseInt(roundMatch[1], 10) : undefined;
                const who = partMatch ? partMatch[1].trim() : GameState.playerName;
                const actions = actionsMatch ? parseInt(actionsMatch[1], 10) : 1;
                CombatModal.queueTurnBanner(who, roundNum, actions, isNpcTurn);
                return;
            }

            if (sub === 'DAMAGE') {
                const dmgMatch = rest.match(/^attacker=(.+?)\s+target=(.+?)\s+amount=(\d+)\s+remaining_hp=(\d+)$/);
                if (!dmgMatch) return;

                const attackerName = dmgMatch[1].trim();
                const targetName = dmgMatch[2].trim();
                const dmg = parseInt(dmgMatch[3], 10);
                const remHp = parseInt(dmgMatch[4], 10);

                const targetIsMe = (targetName.toLowerCase() === GameState.playerName.toLowerCase());
                const targetNpcMeta = GameState.getNpcMeta(targetName);
                const targetIsNpc = (targetName === CombatModal.enemyName) || (ASSETS.npcs[targetNpcMeta.id] !== undefined && targetNpcMeta.id !== 'default');

                if (targetIsNpc) {
                    const banner = (attackerName.toLowerCase() === GameState.playerName.toLowerCase())
                        ? 'YOUR STRIKE'
                        : `${attackerName.toUpperCase()} STRIKES`;
                    CombatModal.playHitEffect('enemy_hit', dmg, remHp, banner, '', attackerName);
                    UI.addChatLine('COMBAT', '🗡️ Strike', `${attackerName} hits ${targetName} for ${dmg} damage! (Enemy HP: ${remHp})`, 'chat-sys');
                } else if (targetIsMe) {
                    CombatModal.playHitEffect('me_hit', dmg, remHp);
                    UI.addChatLine('COMBAT', '⚔️ Enemy Strike', `${attackerName} attacks you for ${dmg} damage! (HP: ${remHp})`, 'chat-err');
                } else {
                    CombatModal.playHitEffect('ally_hit', dmg, undefined, '', targetName);
                    UI.addChatLine('COMBAT', '🗡️ Ally Hit', `${attackerName} strikes ${targetName} for ${dmg} damage! (${targetName} HP: ${remHp})`, 'chat-err');
                }
                return;
            }

            if (sub === 'HELP REQUESTED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const cMatch = rest.match(/combat=([^\s]+)/);
                const rMatch = rest.match(/room=([^\s]+)/);
                const caller = pMatch ? pMatch[1] : 'An ally';
                const combatId = cMatch ? cMatch[1] : '';
                const helpRoom = rMatch ? rMatch[1] : GameState.room.id;
                const alreadyFled = !!(combatId && GameState.fledCombatIds && GameState.fledCombatIds[combatId]);
                if (combatId && !alreadyFled && caller.toLowerCase() !== GameState.playerName.toLowerCase()) {
                    GameState.registerKnownCombat(combatId, caller, helpRoom, { helpRequested: true });
                    UI.renderRoom();
                }
                UI.addChatLine('COMBAT', '📯 Call for Arms!', `${caller} is requesting help in combat!${alreadyFled ? '' : ' Click "⚔️ Aid" in the player list or attack the enemy to join!'}`, 'chat-err');
                return;
            }

            if (sub === 'PLAYER JOINED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const ally = pMatch ? pMatch[1] : '';
                if (ally) {
                    CombatModal.addJoinedParticipant(ally);
                }
                UI.addChatLine('COMBAT', '🛡️ Ally Joined!', `${ally || 'An ally'} has joined the battle!`, 'chat-sys');
                return;
            }

            if (sub === 'PLAYER FLED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const runner = pMatch ? pMatch[1] : 'A player';
                UI.addChatLine('COMBAT', '🏃 Ally Fled', `${runner} fled from the battle!`, 'chat-err');
                CombatModal.handleAllyFled(runner);
                return;
            }

            if (sub === 'ENDED') {
                const cMatch = rest.match(/combat=([^\s]+)/);
                const endedCombatId = (cMatch && cMatch[1]) || CombatModal.combatId;
                if (endedCombatId) {
                    GameState.removeKnownCombat(endedCombatId);
                }

                const rewardNames = [];
                const rewIdx = rest.indexOf('rewards=');
                if (rewIdx !== -1) {
                    try {
                        const rewardsArr = JSON.parse(rest.slice(rewIdx + 'rewards='.length).trim());
                        if (Array.isArray(rewardsArr)) {
                            rewardsArr.forEach(r => {
                                (r.Items || r.items || []).forEach(it => {
                                    if (it && (it.name || it.id)) {
                                        rewardNames.push(it.name || GameState.getItemMeta(it.id).name);
                                    }
                                });
                            });
                        }
                    } catch (_) {}
                }

                if (rewardNames.length > 0) {
                    UI.addChatLine('COMBAT', '🎁 Loot Received', `Obtained: ${rewardNames.join(', ')}!`, 'chat-sys');
                    this.sendCommand('INVENTORY');
                }

                CombatModal.handleCombatEnded(rewardNames);
                return;
            }

            if (sub === 'NPC DIED') {
                const loot = CombatModal.pendingCombatRewards || [];
                const lootMsg = loot.length > 0
                    ? `${CombatModal.enemyName} defeated! Obtained: ${loot.join(', ')}.`
                    : `${CombatModal.enemyName} defeated! Rewards claimed!`;
                UI.addChatLine('COMBAT', '🏆 Victory!', lootMsg, 'chat-sys');
                if (GameState.inCombat) {
                    CombatModal.showOutcome(true, lootMsg);
                }
                this.sendCommand('INVENTORY');
                return;
            }

            if (sub === 'PLAYER DIED') {
                const pMatch = rest.match(/^participant=(.+?)\s+killer=/);
                const fallenName = pMatch ? pMatch[1].trim() : '';
                const diedIsMe = (fallenName.toLowerCase() === GameState.playerName.toLowerCase());

                if (diedIsMe) {
                    UI.addChatLine('COMBAT', '💀 Fallen', 'You fell in battle!', 'chat-err');
                } else {
                    UI.addChatLine('COMBAT', '💀 Ally Fallen', `${fallenName || 'Your ally'} has fallen in battle!`, 'chat-err');
                }

                CombatModal.handlePlayerDeath(diedIsMe, fallenName);
                return;
            }
            return;
        }
    }
};