// ==========================================================================
// CORE MODULE: NETWORK CLIENT & TAP PROTOCOL PARSER
// ==========================================================================

const App = {
    eventSource: null,

    async restoreSessionIfActive() {
        await this.loadWorldMeta();

        this.initStream();
        try {
            const res = await fetch('/api/session');
            const data = await res.json();
            if (data.connected && data.username) {
                GameState.connected = true;
                GameState.playerName = data.username;
                UI.enterGameScreen();
                UI.addChatLine('LOG', 'System', `Session restored as ${data.username}.`, 'chat-sys');
                await this.sendCommand('LOOK');
                await this.sendCommand('STATUS');
                await this.sendCommand('INVENTORY');
                await this.sendCommand('WHO');

                const savedEnemy = sessionStorage.getItem('tap_active_combat_enemy');
                if (savedEnemy && savedEnemy !== 'default') {
                    CombatModal.open(savedEnemy, undefined, undefined);
                }
            }
        } catch (_) {}
    },

    async loadWorldMeta() {
        try {
            const res = await fetch('/api/world');
            if (res.ok) {
                const worldData = await res.json();
                GameState.syncWorldDefinition(worldData);
            }
        } catch (e) {
            console.warn("Could not pre-load world metadata:", e);
        }
    },

    async connect() {
        await this.loadWorldMeta();

        const username = document.getElementById('login-username').value.trim();
        const host = document.getElementById('login-host').value.trim();
        const port = document.getElementById('login-port').value.trim();

        if (!username) {
            UI.showLoginError("Please enter a character name.");
            return;
        }

        UI.showLoginError("");
        if (!this.eventSource) this.initStream();

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
            GameState.currentGroup = null;
            GameState.groupInvitations = [];
            GameState.inventory = [];
            GameState.inventoryCounts = {};
            GameState.quests = [];
            GameState.activeHelpCombatId = null;
            GameState.chats = {
                ROOM: [],
                GLOBAL: [],
                GROUP: [],
                LOG: []
            };
            ChatView.renderChatMessages();
            UI.enterGameScreen();
            UI.addChatLine('LOG', 'System', `Connected to ${host}:${port} as ${username}.`, 'chat-sys');

            await this.sendCommand('LOOK');
            await this.sendCommand('WHO');
        } catch (err) {
            UI.showLoginError("Network error: " + err.message);
        }
    },

    initStream() {
        if (this.eventSource) {
            this.eventSource.close();
        }
        this.eventSource = new EventSource('/api/stream');
        this.eventSource.onmessage = (e) => {
            try {
                const packet = JSON.parse(e.data);
                this.handleServerPacket(packet);
            } catch (err) {
                console.error("Failed to parse server event:", err);
            }
        };
        this.eventSource.onerror = async () => {
            try {
                await fetch('/api/session');
            } catch (_) {
                window.close();
            }
        };
    },

    async sendCommand(rawCommand) {
        const cmd = rawCommand.trim();
        if (!cmd) return;

        const upper = cmd.toUpperCase();
        let verb = upper.split(/\s+/)[0];
        if (upper.startsWith('GROUP LEAVE')) verb = 'GROUP_LEAVE';
        if (upper.startsWith('GROUP INFO')) verb = 'GROUP_INFO';
        GameState.commandQueue.push(verb);

        UI.addChatLine('LOG', 'Client →', cmd, '');

        if (upper === 'COMBAT DEFEND') {
            CombatModal.playDefendEffect(50);
        }
        try {
            await fetch('/api/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            });
        } catch (err) {
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

        const scope = GameState.activeChatScope;
        this.sendCommand(`CHAT ${scope} ${text}`);
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

            // If the server says we are not in a group, we clear the local group state!
            if (raw.includes('NOT_IN_GROUP')) {
                GroupModal.renderCurrentGroup(null);
                if (failedVerb === 'GROUP_INFO') return; // Do not spam the chat during a silent check.
            }

            if (raw.includes('ALREADY_IN_COMBAT')) {
                const enemyInRoom = GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile) || CombatModal.enemyId || 'dire_wolf';
                CombatModal.open(enemyInRoom, GameState.hp, undefined);
                return;
            }

            UI.addChatLine('ROOM', '⚠️ Error', raw, 'chat-err');
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
            GroupModal.renderCurrentGroup(null);
            UI.addChatLine('ROOM', '👥 Group', 'You left the group.', 'chat-sys');
            return;
        }

        if (payload === 'bye') {
            UI.returnToLogin('');
            return;
        }

        if (payload.startsWith('room=')) {
            const newRoom = payload.split('=')[1];
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
            UI.addChatLine('ROOM', '🧪 Used Item', `Used ${meta.name} (+${healed} HP)!`, 'chat-sys');
            this.sendCommand('INVENTORY');
            this.sendCommand('STATUS');
            return;
        }
        if (payload.startsWith('defended=')) {
            const redMatch = payload.match(/(?:reduction|defense_bonus)=(\d+)/);
            const pct = redMatch ? redMatch[1] : 50;
            UI.addChatLine('ROOM', '🛡️ Defend', `Defensive stance active (+${pct}% defense on next hit).`, 'chat-sys');
            return;
        }
        if (payload.startsWith('fled=')) {
            const hpMatch = payload.match(/hp=(\d+)/);
            const dmgMatch = payload.match(/damage=(\d+)/);
            if (hpMatch) GameState.hp = parseInt(hpMatch[1], 10);
            UI.updateHUDStats();
            CombatModal.close();
            UI.addChatLine('ROOM', '🏃 Retreated', `You fled from combat (took ${dmgMatch ? dmgMatch[1] : 0} escape penalty damage).`, 'chat-err');
            return;
        }
        if (payload.startsWith('help_requested=')) {
            UI.addChatLine('ROOM', '📯 Call for Help', 'Your call for help echoed through the room!', 'chat-sys');
            return;
        }
        if (payload.startsWith('joined=')) {
            GameState.activeHelpCombatId = null;
            const thpMatch = payload.match(/target_hp=(\d+)/);
            const currentEnemyHp = thpMatch ? parseInt(thpMatch[1], 10) : undefined;
            const enemyInRoom = GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile) || GameState.npcsInRoom[0] || 'dire_wolf';
            CombatModal.open(enemyInRoom, GameState.hp, currentEnemyHp);
            UI.addChatLine('ROOM', '⚔️ Joined Battle', `You joined the battle against ${CombatModal.enemyName}!`, 'chat-sys');
            UI.renderRoom();
            return;
        }
        if (payload.startsWith('target=')) {
            // Response to COMBAT ATTACK (the strikes themselves have already occurred via EVT COMBAT DAMAGE)
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
            this.sendCommand('GROUP INVITATIONS');
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
                    GameState.npcsInRoom = data.npcs || [];
                    if (GameState.inCombat && (!CombatModal.enemyId || CombatModal.enemyId === 'default')) {
                        const roomHostile = GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile);
                        if (roomHostile) {
                            CombatModal.open(roomHostile, GameState.hp, undefined);
                        }
                    }
                    UI.renderRoom();
                    return;
                }

                if (data.npc && Array.isArray(data.dialogue)) {
                    DialogueModal.renderConversation(data);
                    UI.addChatLine('ROOM', `💬 ${data.npc.name}`, data.dialogue.join(' '));
                    return;
                }

                if ((data.ID || data.id) && Array.isArray(data.Members || data.members)) {
                    GroupModal.renderCurrentGroup(data);
                    return;
                }

                if (data.id && data.name && data.description && !data.type) {
                    UI.addChatLine('ROOM', '📜 Quest Accepted', `${data.name}: ${data.description}`, 'chat-sys');
                    if (DialogueModal.activeNpcId) this.sendCommand(`TALK ${DialogueModal.activeNpcId}`);
                    this.sendCommand('QUESTS');
                    return;
                }

                if (data.quest_id && data.status) {
                    UI.addChatLine('ROOM', '🏆 Quest Completed', `${data.message}: ${data.quest_id}`, 'chat-sys');
                    if (DialogueModal.activeNpcId) this.sendCommand(`TALK ${DialogueModal.activeNpcId}`);
                    this.sendCommand('INVENTORY');
                    this.sendCommand('QUESTS');
                    return;
                }

                if (Array.isArray(data)) {
                    if (lastVerb === 'GROUP') {
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
                    if (GameState.inCombat) {
                        CombatModal.updateBars();
                    }
                    return;
                }

                if (data.combat_id || data.attacker_hp !== undefined) {
                    CombatModal.open(CombatModal.enemyId || 'dire_wolf', data.attacker_hp, data.target_hp);
                    UI.addChatLine('ROOM', '⚔️ Combat', `Battle started against ${CombatModal.enemyName}!`, 'chat-sys');
                    return;
                }

            } catch (e) {
                console.warn("Non-standard JSON payload:", payload);
            }
        }
    },

    handleEventLine(raw) {
        const parts = raw.split(' ');
        const category = parts[1];

        // 1. Presence events (both RFC space-separated and underscore formats)
        if (category === 'ROOM' && parts[2] === 'PRESENCE') {
            const isEnter = parts[3] === 'ENTER';
            const user = parts.slice(4).join(' ');
            UI.addChatLine('ROOM', '👣 Presence', `${user} ${isEnter ? 'entered' : 'left'} the location.`, 'chat-sys');
            this.sendCommand('LOOK');
            this.sendCommand('WHO');
            return;
        }
        if (category === 'ROOM_PRESENCE_ENTER' || category === 'ROOM_PRESENCE_LEAVE') {
            const user = parts[2];
            const isEnter = category === 'ROOM_PRESENCE_ENTER';
            UI.addChatLine('ROOM', '👣 Presence', `${user} ${isEnter ? 'entered' : 'left'} the location.`, 'chat-sys');
            this.sendCommand('LOOK');
            this.sendCommand('WHO');
            return;
        }

        // 2. Chat events
        if ((category === 'ROOM' || category === 'GLOBAL' || category === 'GROUP') && parts[2] === 'CHAT') {
            const rest = parts.slice(3).join(' ');
            const kvMatch = rest.match(/^player=(\S+)\s+message=(.*)$/);
            if (kvMatch) {
                UI.addChatLine(category, kvMatch[1], kvMatch[2]);
            } else {
                const user = parts[3] || 'Player';
                const msg = parts.slice(4).join(' ');
                UI.addChatLine(category, user, msg);
            }
            return;
        }

        // 3. Group events
        if (category === 'GROUP') {
            const action = parts[2];
            const details = parts.slice(3).join(' ');
            UI.addChatLine('GROUP', '👥 Party', `${action}: ${details}`, 'chat-sys');
            UI.addChatLine('ROOM', '👥 Party', `${action}: ${details}`, 'chat-sys');
            if (action === 'INVITE') {
                this.sendCommand('GROUP INVITATIONS');
            } else {
                this.sendCommand('GROUP INFO');
            }
            if (GameState.currentGroup) this.sendCommand('GROUP INFO');
            return;
        }

        // 4. Item events
        if (category === 'ITEM_DROPPED' || category === 'UNIQUE_ITEMS_RETURNED') {
            UI.addChatLine('ROOM', '🧰 World', `Items updated in room (${parts.slice(2).join(' ')}).`, 'chat-sys');
            this.sendCommand('LOOK');
            return;
        }

        // 5. Combat events
        if (category === 'COMBAT') {
            const sub = parts[2];
            const rest = parts.slice(3).join(' ');

            if (sub === 'STARTED') {
                // If a group ally starts a battle — we open the arena too!
                if (!GameState.inCombat) {
                    const enemyInRoom = GameState.npcsInRoom.find(n => GameState.getNpcMeta(n).hostile) || GameState.npcsInRoom[0] || 'dire_wolf';
                    CombatModal.open(enemyInRoom, GameState.hp, undefined);
                }
                return;
            }

            if (sub === 'TURN') {
                const roundMatch = rest.match(/round=(\d+)/);
                const partMatch = rest.match(/participant=(.+?)\s+round=/);
                const roundNum = roundMatch ? parseInt(roundMatch[1], 10) : undefined;
                const who = partMatch ? partMatch[1] : '';
                const label = (who === GameState.playerName ? 'YOUR TURN' : `${who.toUpperCase()}'S TURN`);
                CombatModal.queueTurnBanner(label, roundNum);
                return;
            }

            if (sub === 'DAMAGE') {
                const dmgMatch = rest.match(/^attacker=(.+?)\s+target=(.+?)\s+amount=(\d+)\s+remaining_hp=(\d+)$/);
                if (!dmgMatch) return;

                const attackerName = dmgMatch[1];
                const targetName = dmgMatch[2];
                const dmg = parseInt(dmgMatch[3], 10);
                const remHp = parseInt(dmgMatch[4], 10);

                const targetIsMe = (targetName.toLowerCase() === GameState.playerName.toLowerCase());
                const targetNpcMeta = GameState.getNpcMeta(targetName);
                const targetIsNpc = (targetName === CombatModal.enemyName) || (ASSETS.npcs[targetNpcMeta.id] !== undefined && targetNpcMeta.id !== 'default');

                if (targetIsNpc) {
                    CombatModal.playHitEffect(true, dmg, remHp);
                    UI.addChatLine('ROOM', '🗡️ Strike', `${attackerName} hits ${targetName} for ${dmg} damage! (Enemy HP: ${remHp})`, 'chat-sys');
                } else if (targetIsMe) {
                    CombatModal.playHitEffect(false, dmg, remHp);
                    UI.addChatLine('ROOM', '🛡️ Enemy Strike', `${attackerName} attacks you for ${dmg} damage! (HP: ${remHp})`, 'chat-err');
                } else {
                    CombatModal.playHitEffect(false, dmg, undefined);
                    UI.addChatLine('ROOM', '🛡️ Ally Hit', `${attackerName} strikes ${targetName} for ${dmg} damage! (${targetName} HP: ${remHp})`, 'chat-err');
                }
                return;
            }

            if (sub === 'HELP_REQUESTED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const cMatch = rest.match(/combat=([^\s]+)/);
                const caller = pMatch ? pMatch[1] : 'An ally';
                const combatId = cMatch ? cMatch[1] : '';
                if (combatId) {
                    GameState.activeHelpCombatId = combatId;
                    GameState.activeHelpPlayer = caller;
                    UI.renderRoom();
                }
                UI.addChatLine('ROOM', '📯 Call for Arms!', `${caller} is requesting help in combat! Click "⚔️ Join Battle" on the stage or player list!`, 'chat-err');
                return;
            }

            if (sub === 'PLAYER_JOINED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const ally = pMatch ? pMatch[1] : 'An ally';
                UI.addChatLine('ROOM', '🛡️ Ally Joined!', `${ally} has joined the battle!`, 'chat-sys');
                return;
            }

            if (sub === 'PLAYER_FLED') {
                const pMatch = rest.match(/player=([^\s]+)/);
                const runner = pMatch ? pMatch[1] : 'A player';
                UI.addChatLine('ROOM', '🏃 Ally Fled', `${runner} fled from the battle!`, 'chat-err');
                return;
            }

            if (sub === 'NPC_DIED') {
                UI.addChatLine('ROOM', '🏆 Victory!', `${CombatModal.enemyName} defeated! Rewards added to your bag!`, 'chat-sys');
                CombatModal.showOutcome(true, `${CombatModal.enemyName} defeated! Rewards claimed!`);
                this.sendCommand('INVENTORY');
                this.sendCommand('STATUS');
                this.sendCommand('LOOK');
                GameState.activeHelpCombatId = null;
                return;
            }

            if (sub === 'PLAYER_DIED') {
                if (rest.includes(`participant=${GameState.playerName}`)) {
                    UI.addChatLine('ROOM', '💀 Defeated', `You were defeated and respawned at Central Harbor Square!`, 'chat-err');
                    CombatModal.showOutcome(false, 'You were defeated! Respawning at Harbor Square...');
                    this.sendCommand('LOOK');
                    this.sendCommand('STATUS');
                    GameState.activeHelpCombatId = null;
                }
                return;
            }
            return;
        }
    }
};