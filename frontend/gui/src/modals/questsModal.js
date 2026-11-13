// ==========================================================================
// MODAL MODULE: ADVENTURE JOURNAL (QUESTS — Active & Completed Tabs)
// ==========================================================================

const QuestsModal = {
    currentTab: 'active', // 'active' or 'completed'

    switchTab(tab) {
        this.currentTab = tab;
        document.getElementById('q-tab-active').classList.toggle('active', tab === 'active');
        document.getElementById('q-tab-completed').classList.toggle('active', tab === 'completed');
        this.render();
    },

    render() {
        const box = document.getElementById('quests-list');
        if (!box) return;
        box.innerHTML = '';

        const allQuests = GameState.quests || [];
        const activeList = allQuests.filter(q => (q.status || '').toLowerCase() !== 'completed');
        const completedList = allQuests.filter(q => (q.status || '').toLowerCase() === 'completed');

        // Updating counters on the tabs Active (X) | Completed (Y)
        const countActEl = document.getElementById('q-count-active');
        const countCmpEl = document.getElementById('q-count-completed');
        if (countActEl) countActEl.textContent = activeList.length;
        if (countCmpEl) countCmpEl.textContent = completedList.length;

        const list = this.currentTab === 'completed' ? completedList : activeList;

        if (list.length === 0) {
            const emptyMsg = this.currentTab === 'completed'
                ? 'No completed quests yet. Finish active tasks and talk to the quest-giver!'
                : 'No active quests right now. Speak with Quest Advisor, Apprentice Elisa, Captain Eliza, or Mara to accept quests!';
            box.innerHTML = `<li class="card-sub" style="padding:12px 6px;">${emptyMsg}</li>`;
            this.showDetails({
                name: this.currentTab === 'completed' ? 'No Completed Quests' : 'Explore Arvendale',
                description: emptyMsg,
                status: this.currentTab,
                progress: ''
            });
            return;
        }

        list.forEach((q, idx) => {
            const qTitle = q.name || q.id || q.quest_id || 'Quest';
            const statusText = (q.status || 'active').toUpperCase();
            const badgeText = q.progress ? `${statusText} (${q.progress})` : statusText;

            const li = document.createElement('li');
            li.className = 'select-card' + (idx === 0 ? ' selected' : '');
            li.innerHTML = `
                <div class="card-left">
                    <div class="card-thumb"><img src="/assets/icons/command/quest.png" alt="" class="art-img contain"></div>
                    <div>
                        <strong class="card-title">${qTitle}</strong>
                        <span class="card-sub">${q.description || 'Active objective'}</span>
                    </div>
                </div>
                <span class="badge">${badgeText}</span>
            `;
            li.addEventListener('click', () => {
                document.querySelectorAll('#quests-list .select-card').forEach(c => c.classList.remove('selected'));
                li.classList.add('selected');
                this.showDetails(q);
            });
            if (idx === 0) {
                this.showDetails(q);
            }
            box.appendChild(li);
        });
    },

    showDetails(q) {
        const qId = q.id || q.quest_id || '';
        const qTitle = q.name || qId || 'Quest';
        document.getElementById('quest-detail-title').textContent = qTitle;
        document.getElementById('quest-detail-desc').textContent = q.description || '';
        document.getElementById('quest-detail-obj').innerHTML = q.progress
            ? `<img src="/assets/icons/ui/target.webp" alt="" class="ui-ico">Progress: ${q.progress} (${q.status})`
            : `<img src="/assets/icons/ui/checkmark.png" alt="" class="ui-ico">Status: ${(q.status || 'Active').toUpperCase()}`;

        // Dynamically fetch rewards from world.json!
        let rewardText = q.reward || 'Armor Gear & Healing Consumables';
        if (GameState.worldQuests && GameState.worldQuests[qId]) {
            const rewIds = GameState.worldQuests[qId].reward_item_ids || [];
            if (rewIds.length > 0) {
                rewardText = rewIds.map(id => GameState.getItemMeta(id).name).join(' + ');
            }
        }
        document.getElementById('quest-detail-rew').innerHTML =
            `<img src="/assets/icons/ui/sparkles.webp" alt="" class="ui-ico">${rewardText}`;
    }
};