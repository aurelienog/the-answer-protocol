// ==========================================================================
// MODAL MODULE: LOOK — LOCATION (5 Tabs)
// ==========================================================================

const LookModal = {
    switchTab(tabId) {
        document.querySelectorAll('#modal-look .tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.lookTab === tabId);
        });
        document.querySelectorAll('#modal-look .look-tab-panel').forEach(panel => {
            panel.classList.toggle('hidden', panel.id !== tabId);
        });
    },

    render() {
        const room = GameState.room;
        const meta = ASSETS.rooms[room.id] || ASSETS.rooms.default;
        const exits = room.exits || {};

        const lookImg = meta.cardImg || meta.img;
        UI.setArt(document.getElementById('look-art-box'), lookImg, meta.icon);

        document.getElementById('look-sub-id').textContent = `ID: ${room.id}`;
        document.getElementById('look-room-title').textContent = room.name;
        document.getElementById('look-room-region').textContent = meta.region;
        document.getElementById('look-room-desc').textContent = room.description;

        // Exits Tab
        const lExits = document.getElementById('look-exits-list');
        lExits.innerHTML = '';
        Object.keys(exits).forEach(dir => {
            const targetId = exits[dir];
            const tMeta = ASSETS.rooms[targetId] || ASSETS.rooms.default;
            const li = document.createElement('li');
            li.className = 'select-card';
            li.dataset.cmd = `MOVE ${dir}`;
            li.innerHTML = `
                <div class="card-left">
                    <div class="card-thumb"></div>
                    <div><strong class="card-title">${dir.toUpperCase()}</strong><span class="card-sub">Leads to: ${targetId}</span></div>
                </div>
                <img src="/assets/icons/ui/next.png" alt="❯" class="inline-cmd-icon">
            `;
            UI.setArt(li.querySelector('.card-thumb'), tMeta.iconImg, '🧭');
            lExits.appendChild(li);
        });

        // Items Tab
        const lItems = document.getElementById('look-items-list');
        lItems.innerHTML = GameState.itemsInRoom.length ? '' : '<li class="card-sub">No items in this location.</li>';
        GameState.itemsInRoom.forEach(itemId => {
            const m = GameState.getItemMeta(itemId);
            const qty = (GameState.roomItemCounts && GameState.roomItemCounts[itemId]) || 1;
            const li = document.createElement('li');
            li.className = 'select-card';
            li.innerHTML = `
                <div class="card-left">
                    <div class="card-thumb"></div>
                    <div><strong class="card-title">${qty > 1 ? `${m.name} (×${qty})` : m.name}</strong><span class="card-sub">${m.desc}</span></div>
                </div>
                <button type="button" class="btn-gold-sm" data-cmd="TAKE ${itemId}">Take</button>
            `;
            UI.setArt(li.querySelector('.card-thumb'), m.iconImg || m.img, m.icon, m.name);
            lItems.appendChild(li);
        });

        // NPCs Tab
        const lNpcs = document.getElementById('look-npcs-list');
        lNpcs.innerHTML = GameState.npcsInRoom.length ? '' : '<li class="card-sub">No NPCs in this location.</li>';
        GameState.npcsInRoom.forEach(npcId => {
            const m = GameState.getNpcMeta(npcId);
            const li = document.createElement('li');
            li.className = 'select-card';
            li.innerHTML = `
                <div class="card-left">
                    <div class="card-thumb"></div>
                    <div><strong class="card-title">${m.name}</strong><span class="card-sub">${m.role}</span></div>
                </div>
                <span class="badge">ID: ${m.id}</span>
            `;
            UI.setArt(li.querySelector('.card-thumb'), m.iconImg, m.icon);
            lNpcs.appendChild(li);
        });

        FrameView.mountSlotFrames();
    }
};