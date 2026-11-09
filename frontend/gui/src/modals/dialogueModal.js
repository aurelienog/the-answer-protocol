// ==========================================================================
// MODAL MODULE: NPC DIALOGUE & QUEST ACCEPT/COMPLETE WINDOW
// ==========================================================================

const DialogueModal = {
    activeNpcId: null,
    activeQuestId: null,
    activeQuestStatus: null,

    // An NPC window opens with a greeting, allowing the player to choose between talking or accepting/turning in a quest!
    open(npcId) {
        UI.closeAllModals();
        UI.activeModalId = 'dialogue';

        const meta = GameState.getNpcMeta(npcId);
        this.activeNpcId = meta.id;
        this.activeQuestId = null;
        this.activeQuestStatus = null;

        document.getElementById('dlg-header-title').textContent = `TALK — ${meta.name}`;
        document.getElementById('dlg-header-sub').textContent = `Converse with ${meta.role}.`;

        const artEl = document.getElementById('dlg-npc-art');
        UI.setArt(artEl, meta.img, meta.icon, meta.name);

        document.getElementById('dlg-speaker-label').innerHTML = `${meta.name} <span style="color:var(--text-muted);font-size:14px;">· ${meta.role}</span>`;

        // Default GUI greeting before the player presses the button
        const speechEl = document.getElementById('dlg-speech-text');
        if (speechEl) {
            speechEl.textContent = `"Greetings, adventurer! Do you wish to speak with me, or are you looking for a quest?"`;
        }

        document.getElementById('dlg-talk-btn-label').textContent = `Speak with ${meta.name} (TALK)`;

        const questBtn = document.getElementById('dlg-btn-quest');
        const questTitle = document.getElementById('dlg-quest-btn-label');
        if (meta.hasQuest) {
            questBtn.classList.remove('hidden');
            questTitle.textContent = `Ask about Quests / Turn In (QUEST)`;
            speechEl.textContent = `"Greetings, adventurer! Do you wish to speak with me, or are you looking for a quest?"`;
        } else {
            questBtn.classList.add('hidden');
            speechEl.textContent = `"Greetings, traveler! Would you like to hear what I know about Arvendale?"`;
        }

        document.getElementById('modal-dialogue').classList.remove('hidden');
    },

    renderConversation(data) {
        const npc = data.npc || {};
        if (npc.id) {
            this.activeNpcId = npc.id;
            const meta = GameState.getNpcMeta(npc.id);
            document.getElementById('dlg-speaker-label').innerHTML = `${meta.name} <span style="color:var(--text-muted);font-size:14px;">· ${meta.role}</span>`;
        }

        // We are updating the NPC dialogue text right in the window above the buttons!
        const lines = Array.isArray(data.dialogue) ? data.dialogue.join(' ') : String(data.dialogue || '');
        const speechEl = document.getElementById('dlg-speech-text');
        if (speechEl && lines) {
            speechEl.textContent = `"${lines}"`;
        }

        const questBtn = document.getElementById('dlg-btn-quest');
        const titleEl = document.getElementById('dlg-quest-btn-label');

        if (data.quest && data.quest.id) {
            this.activeQuestId = data.quest.id;
            this.activeQuestStatus = data.quest_status || 'available';

            if (this.activeQuestStatus === 'available') {
                questBtn.classList.remove('hidden');
                titleEl.textContent = `Accept Quest: ${data.quest.name}`;
            } else if (this.activeQuestStatus === 'active') {
                questBtn.classList.remove('hidden');
                const prog = data.quest.progress ? ` (${data.quest.progress})` : '';
                titleEl.textContent = `Turn In / Check Quest: ${data.quest.name}${prog}`;
            } else if (this.activeQuestStatus === 'completed') {
                questBtn.classList.add('hidden');
                UI.addChatLine('ROOM', '🏆 Quest Completed!', `Completed "${data.quest.name}"! Rewards added to your inventory.`, 'chat-sys');
                App.sendCommand('INVENTORY');
                App.sendCommand('QUESTS');
            } else {
                questBtn.classList.add('hidden');
            }
        } else {
            this.activeQuestId = null;
            this.activeQuestStatus = null;
            questBtn.classList.add('hidden');
        }
    }
};