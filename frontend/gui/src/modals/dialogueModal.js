// ==========================================================================
// MODAL MODULE: NPC DIALOGUE & QUEST ACCEPT/COMPLETE WINDOW
// ==========================================================================

const DialogueModal = {
    activeNpcId: null,
    activeQuestId: null,
    activeQuestName: null,
    activeQuestStatus: null,

    open(npcId) {
        UI.closeAllModals();
        UI.activeModalId = 'dialogue';

        const meta = GameState.getNpcMeta(npcId);
        this.activeNpcId = meta.id;

        // Check if player already has an ACTIVE quest for this NPC before sending TALK
        const existingActive = this.findPlayerActiveQuestForNpc(meta.id);
        if (existingActive) {
            this.activeQuestId = existingActive.id;
            this.activeQuestName = existingActive.name;
            this.activeQuestStatus = 'active';
        } else {
            this.activeQuestId = null;
            this.activeQuestName = null;
            this.activeQuestStatus = null;
        }

        document.getElementById('dlg-header-title').textContent = `TALK — ${meta.name}`;
        document.getElementById('dlg-header-sub').textContent = `Converse with ${meta.role}.`;

        const artEl = document.getElementById('dlg-npc-art');
        UI.setArt(artEl, meta.img, meta.icon, meta.name);

        document.getElementById('dlg-speaker-label').innerHTML =
            `${meta.name} <span style="color:var(--text-muted);font-size:14px;">· ${meta.role}</span>`;

        const speechEl = document.getElementById('dlg-speech-text');
        if (speechEl) {
            speechEl.textContent = `"..."`;
        }

        document.getElementById('dlg-talk-btn-label').textContent = `Speak with ${meta.name} (TALK)`;

        const questBtn = document.getElementById('dlg-btn-quest');
        questBtn.classList.add('hidden');

        document.getElementById('modal-dialogue').classList.remove('hidden');

        // Automatically trigger TALK on open so the NPC speaks immediately
        // and the server discovers or resolves the quest!
        App.sendCommand(`TALK ${meta.id}`);
    },

    // Finds if the player currently has an active quest belonging to this NPC
    findPlayerActiveQuestForNpc(npcId) {
        const wQuests = GameState.worldQuests || {};
        for (const q of (GameState.quests || [])) {
            const qId = q.id || q.quest_id;
            if ((q.status || '').toLowerCase() === 'active' && wQuests[qId] && wQuests[qId].giver_npc_id === npcId) {
                return {
                    id: qId,
                    name: q.name || wQuests[qId].name || qId,
                    def: wQuests[qId]
                };
            }
        }
        if (this.activeNpcId === npcId && this.activeQuestStatus === 'active' && this.activeQuestId) {
            return {
                id: this.activeQuestId,
                name: this.activeQuestName || this.activeQuestId,
                def: wQuests[this.activeQuestId]
            };
        }
        return null;
    },

    // Checks if player holds the required item for a quest in their inventory
    playerHasRequiredItem(questDef) {
        if (!questDef || !questDef.required_item_id) return false;
        const reqId = questDef.required_item_id;
        const reqQty = questDef.required_item_quantity || 1;
        const haveQty = (GameState.inventoryCounts && GameState.inventoryCounts[reqId])
            || (GameState.inventory.includes(reqId) ? 1 : 0);
        return haveQty >= reqQty;
    },

    announceQuestCompletion(questId, questName) {
        const wQuests = GameState.worldQuests || {};
        const qDef = wQuests[questId];
        let rewardStr = 'Rewards added to your inventory!';
        if (qDef && Array.isArray(qDef.reward_item_ids) && qDef.reward_item_ids.length > 0) {
            const rNames = qDef.reward_item_ids.map(id => GameState.getItemMeta(id).name);
            rewardStr = `Received: ${rNames.join(' + ')}!`;
        }

        // Update local state immediately so it doesn't trigger twice
        GameState.quests = (GameState.quests || []).map(q => {
            if ((q.id || q.quest_id) === questId) {
                return { ...q, status: 'completed' };
            }
            return q;
        });

        UI.addChatLine('ROOM', '🏆 Quest Completed!', `Completed "${questName}"! ${rewardStr}`, 'chat-sys');
        App.sendCommand('INVENTORY');
        App.sendCommand('QUESTS');
    },

    renderConversation(data) {
        const npc = data.npc || {};
        const npcId = npc.id || this.activeNpcId;

        if (npcId) {
            this.activeNpcId = npcId;
            const meta = GameState.getNpcMeta(npcId);
            document.getElementById('dlg-header-title').textContent = `TALK — ${meta.name}`;
            document.getElementById('dlg-speaker-label').innerHTML =
                `${meta.name} <span style="color:var(--text-muted);font-size:14px;">· ${meta.role}</span>`;
            document.getElementById('dlg-talk-btn-label').textContent = `Speak again with ${meta.name} (TALK)`;
            UI.setArt(document.getElementById('dlg-npc-art'), meta.img, meta.icon, meta.name);
        }

        // Detect if a quest was auto-completed by the server during this TALK!
        // Why? Because on the server, if a completed quest has a `next_quest_id` or is `repeatable`,
        // TalkToNPC resolves the active quest and immediately advances to the next/repeatable quest with status="available"!
        const prevActive = this.findPlayerActiveQuestForNpc(npcId);
        let completedDuringTalk = null;

        if (data.quest && data.quest.id) {
            const incomingId = data.quest.id;
            const incomingStatus = data.quest_status || 'available';

            if (incomingStatus === 'completed') {
                completedDuringTalk = { id: incomingId, name: data.quest.name || incomingId };
            } else if (prevActive) {
                // Case A: Server advanced to the next quest in the chain (e.g. Captain Eliza, Apprentice Elisa)
                if (incomingId !== prevActive.id) {
                    completedDuringTalk = { id: prevActive.id, name: prevActive.name };
                }
                // Case B: Server reset a repeatable quest from 'active' back to 'available' (e.g. Mara, Elisa, Eliza)
                else if (incomingStatus === 'available') {
                    completedDuringTalk = { id: prevActive.id, name: prevActive.name };
                }
            }
        }

        let lines = Array.isArray(data.dialogue) ? data.dialogue.join(' ') : String(data.dialogue || '');

        if (completedDuringTalk) {
            this.announceQuestCompletion(completedDuringTalk.id, completedDuringTalk.name);
            // Prepend a clear completion banner in the NPC speech box so the player sees it even when the next quest is offered!
            lines = `🏆 [QUEST COMPLETED: ${completedDuringTalk.name}] — "${lines}"`;
        }

        const speechEl = document.getElementById('dlg-speech-text');
        if (speechEl && lines) {
            speechEl.textContent = completedDuringTalk ? lines : `"${lines}"`;
        }

        const questBtn = document.getElementById('dlg-btn-quest');
        const titleEl = document.getElementById('dlg-quest-btn-label');

        if (data.quest && data.quest.id) {
            this.activeQuestId = data.quest.id;
            this.activeQuestName = data.quest.name || data.quest.id;
            this.activeQuestStatus = data.quest_status || 'available';

            if (this.activeQuestStatus === 'available') {
                questBtn.classList.remove('hidden');
                titleEl.textContent = `Accept Quest: ${this.activeQuestName} (QUEST)`;
            } else if (this.activeQuestStatus === 'active') {
                questBtn.classList.remove('hidden');
                const prog = data.quest.progress ? ` (${data.quest.progress})` : '';
                titleEl.textContent = `Check / Turn In Quest: ${this.activeQuestName}${prog}`;
            } else {
                // completed or locked
                questBtn.classList.add('hidden');
            }
        } else {
            this.activeQuestId = null;
            this.activeQuestName = null;
            this.activeQuestStatus = null;
            questBtn.classList.add('hidden');
        }
    },

    // Called when server responds OK to `QUEST <npc>`
    onQuestAccepted(questData) {
        const qId = questData.id || this.activeQuestId;
        const qName = questData.name || this.activeQuestName || 'Quest';
        this.activeQuestId = qId;
        this.activeQuestName = qName;
        this.activeQuestStatus = 'active';

        // Also record in GameState.quests immediately so findPlayerActiveQuestForNpc knows it's active!
        const exists = (GameState.quests || []).some(q => (q.id || q.quest_id) === qId);
        if (!exists) {
            GameState.quests.push({
                id: qId,
                name: qName,
                description: questData.description || '',
                status: 'active',
                progress: '0/1'
            });
        } else {
            GameState.quests = GameState.quests.map(q =>
                (q.id || q.quest_id) === qId ? { ...q, status: 'active' } : q
            );
        }

        const speechEl = document.getElementById('dlg-speech-text');
        if (speechEl) {
            speechEl.textContent = `"Quest accepted: ${qName}! ${questData.description || ''}"`;
        }

        const questBtn = document.getElementById('dlg-btn-quest');
        const titleEl = document.getElementById('dlg-quest-btn-label');
        if (questBtn && titleEl) {
            questBtn.classList.remove('hidden');
            titleEl.textContent = `Check / Turn In Quest: ${qName}`;
        }
    }
};