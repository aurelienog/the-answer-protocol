// ==========================================================================
// MODAL MODULE: UNIFIED SELECTOR (MOVE, TAKE, TALK, ATTACK)
// ==========================================================================

const SelectorModal = {
    mode: null,
    selectedValue: null,

    open(mode) {
        this.mode = mode;
        this.selectedValue = null;

        const modal = document.getElementById('modal-selector');
        const icon = document.getElementById('sel-header-icon');
        const title = document.getElementById('sel-header-title');
        const sub = document.getElementById('sel-header-sub');
        const heading = document.getElementById('sel-list-heading');
        const listEl = document.getElementById('sel-options-list');
        const confirmBtn = document.getElementById('btn-sel-confirm');

        icon.innerHTML = `<img src="/assets/icons/command/${mode}.png" alt="" class="modal-header-icon-img">`;

        listEl.innerHTML = '';
        confirmBtn.disabled = true;
        confirmBtn.classList.toggle('btn-attack-mode', mode === 'attack');

        let options = [];

        if (mode === 'move') {
            title.textContent = 'MOVE — Travel';
            sub.textContent = 'Choose a destination from the available exits.';
            confirmBtn.textContent = 'Move';

            const exits = GameState.room.exits || {};
            const dirs = Object.keys(exits);
            heading.textContent = `🧭 Available exits (${dirs.length})`;

            options = dirs.map(dir => {
                const targetId = exits[dir];
                const rMeta = ASSETS.rooms[targetId] || ASSETS.rooms.default;
                return {
                    value: dir,
                    icon: rMeta.icon || '🚪',
                    img: rMeta.cardImg || rMeta.img || '',     // <-- On the left, we preview a compact image from /rooms/cards/!
                    iconImg: rMeta.iconImg || rMeta.img || '', // <-- On the right, we display a small icon from /rooms/icons/!
                    title: dir.charAt(0).toUpperCase() + dir.slice(1),
                    subtitle: targetId.replace(/_/g, ' '),
                    desc: `Travel ${dir} towards ${targetId.replace(/_/g, ' ')}.`,
                    stats: [['Direction', dir.toUpperCase()], ['Destination', targetId], ['Region', rMeta.region]]
                };
            });
        } else if (mode === 'take') {
            title.textContent = 'TAKE — Items in this location';
            sub.textContent = 'Select an item to take it into your inventory.';
            confirmBtn.textContent = 'Take';
            heading.textContent = `🧰 Available items (${GameState.itemsInRoom.length})`;

            options = GameState.itemsInRoom.map(itemId => {
                const m = GameState.getItemMeta(itemId);
                const qty = (GameState.roomItemCounts && GameState.roomItemCounts[itemId]) || 1;
                const typeLabel = (m.type || 'common').toUpperCase();
                return {
                    value: itemId,
                    icon: m.icon,
                    img: m.img || '',
                    iconImg: m.iconImg || m.img || '',
                    title: qty > 1 ? `${m.name} (×${qty})` : m.name,
                    subtitle: m.desc,
                    desc: m.desc,
                    stats: [['Class / Type', typeLabel], ['Available in Room', `${qty}`], ['Item ID', itemId]]
                };
            });
        } else if (mode === 'talk') {
            title.textContent = 'TALK — Choose a character';
            sub.textContent = 'Select an NPC in this location to start a conversation.';
            confirmBtn.textContent = 'Talk';
            heading.textContent = `👥 Available characters (${GameState.npcsInRoom.length})`;

            options = GameState.npcsInRoom.map(npcId => {
                const m = GameState.getNpcMeta(npcId);
                return {
                    value: m.id,
                    icon: m.icon,
                    img: m.img || '',
                    iconImg: m.iconImg || m.img || '',
                    title: m.name,
                    subtitle: m.role,
                    desc: `Approach ${m.name} (${m.role}) and strike up a conversation.`,
                    stats: [['Role', m.role], ['Disposition', m.hostile ? 'Hostile' : 'Friendly'], ['Location', GameState.room.name]]
                };
            });
        } else if (mode === 'attack') {
            title.textContent = 'ATTACK';
            sub.textContent = 'Select a target NPC to initiate combat.';
            confirmBtn.textContent = 'Attack';
            heading.textContent = `🪖 Targets in this location (${GameState.npcsInRoom.length})`;

            options = GameState.npcsInRoom.map(npcId => {
                const m = GameState.getNpcMeta(npcId);
                return {
                    value: m.id,
                    icon: m.icon,
                    img: m.img || '',
                    iconImg: m.iconImg || m.img || '',
                    title: m.name,
                    subtitle: m.hostile ? '💀 Hostile Enemy' : m.role,
                    hostile: m.hostile,
                    desc: `Engage ${m.name} in turn-based combat.`,
                    stats: [['❤️ Health', m.hostile ? '70 HP' : '60 HP'], ['🗡️ Damage', '10 - 12'], ['🛡️ Defense', '5'], ['🪙 Loot', 'Quest Evidence']]
                };
            });
        }

        if (options.length === 0) {
            listEl.innerHTML = `<li class="select-card"><span class="card-sub">Nothing available for ${mode.toUpperCase()} in this room.</span></li>`;
            UI.setArt(document.getElementById('sel-preview-art'), '', icon.textContent);
            document.getElementById('sel-preview-title').textContent = 'None Available';
            document.getElementById('sel-preview-desc').textContent = 'Explore other locations to find more interactions.';
            document.getElementById('sel-preview-stats').innerHTML = '';
        } else {
            options.forEach((opt, idx) => {
                const li = document.createElement('li');
                li.className = 'select-card';
                li.innerHTML = `
                    <div class="card-left">
                        <div class="card-thumb"></div>
                        <div>
                            <strong class="card-title">${opt.title}</strong>
                            <span class="${opt.hostile ? 'hostile-tag' : 'card-sub'}">${opt.subtitle}</span>
                        </div>
                    </div>
                    <div class="check-circle">✔</div>
                `;
                // Let's place a small square icon on the right side of the list!
                UI.setArt(li.querySelector('.card-thumb'), opt.iconImg, opt.icon);

                li.addEventListener('click', () => this.selectCard(li, opt));
                listEl.appendChild(li);
                if (idx === 0) this.selectCard(li, opt);
            });
        }

        FrameView.mountButtons();
        FrameView.mountSlotFrames();
        modal.classList.remove('hidden');
    },

    selectCard(cardEl, opt) {
        document.querySelectorAll('#sel-options-list .select-card').forEach(c => c.classList.remove('selected'));
        cardEl.classList.add('selected');
        this.selectedValue = opt.value;

        // Let's place a LARGE portrait on the left in the preview!
        const artEl = document.getElementById('sel-preview-art');
        artEl.classList.toggle('tall', this.mode === 'talk' || this.mode === 'attack');
        UI.setArt(artEl, opt.img, opt.icon, opt.title);

        document.getElementById('sel-preview-title').textContent = opt.title;
        document.getElementById('sel-preview-desc').textContent = opt.desc;

        const statsBox = document.getElementById('sel-preview-stats');
        statsBox.innerHTML = (opt.stats || [])
            .map(([k, v]) => `<li><span>${k}:</span> <strong>${v}</strong></li>`)
            .join('');

        document.getElementById('btn-sel-confirm').disabled = false;
        FrameView.mountSlotFrames();
    },

    confirmAction() {
        if (!this.selectedValue) return;
        const val = this.selectedValue;
        const mode = this.mode;

        if (mode === 'move') {
            UI.closeAllModals();
            App.sendCommand(`MOVE ${val}`);
        } else if (mode === 'take') {
            App.sendCommand(`TAKE ${val}`);
        } else if (mode === 'attack') {
            CombatModal.enemyId = val;
            UI.closeAllModals();
            App.sendCommand(`ATTACK ${val}`);
        } else if (mode === 'talk') {
            DialogueModal.open(val);
        }
    }
};