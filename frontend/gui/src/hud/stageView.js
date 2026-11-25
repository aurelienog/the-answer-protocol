// ==========================================================================
// HUD MODULE: WORLD STAGE, LEFT NPC BAR (Talk/Attack) & COMPASS WIDGET
// ==========================================================================

const StageView = {
    renderStage() {
        const room = GameState.room;
        const meta = GameState.getRoomMeta(room.id);

        const stageIconEl = document.getElementById('stage-icon');
        if (meta.iconImg) {
            stageIconEl.innerHTML = `<img src="${meta.iconImg}" alt="${room.name} icon" class="stage-banner-icon-img">`;
        } else {
            stageIconEl.textContent = meta.icon;
        }

        document.getElementById('stage-room-title').textContent = room.name;
        document.getElementById('stage-room-region').textContent = meta.region;

        const bgImg = document.getElementById('stage-bg-img');
        const worldStage = document.getElementById('world-stage');
        if (bgImg && meta.img) {
            bgImg.src = meta.img;
            bgImg.alt = `${room.name} — ${meta.region}`;
        } else if (worldStage && meta.img) {
            worldStage.style.backgroundImage = `url('${meta.img}')`;
        }

        const npcDock = document.getElementById('left-npcs-dock');
        if (npcDock) {
            npcDock.innerHTML = '';
            GameState.npcsInRoom.forEach(npcId => {
                const m = GameState.getNpcMeta(npcId);
                const isHostile = !!m.hostile;
                const actionIcon = isHostile
                    ? `/assets/icons/command/attack.png`
                    : `/assets/icons/command/talk.png`;
                const actionLabel = isHostile ? 'ATTACK' : 'TALK';

                const btn = document.createElement('button');
                btn.type = 'button';
                btn.className = `npc-quick-card ${isHostile ? 'npc-hostile' : 'npc-friendly'}`;
                btn.title = `${actionLabel}: ${m.name} (${m.role})`;
                btn.innerHTML = `
                    <div class="card-thumb ${isHostile ? 'hostile-glow' : 'friendly-glow'}">
                        ${m.iconImg ? `<img src="${m.iconImg}" alt="${m.name}" class="art-img cover">` : m.icon}
                    </div>
                    <div class="npc-quick-info">
                        <strong>${m.name}</strong>
                        <span><img src="${actionIcon}" alt="" class="tab-mini-icon">${actionLabel}</span>
                    </div>
                `;
                btn.addEventListener('click', () => {
                    if (isHostile) {
                        CombatModal.enemyId = m.id;
                        UI.closeAllModals();
                        App.sendCommand(`ATTACK ${m.id}`);
                    } else {
                        DialogueModal.open(m.id);
                    }
                });
                npcDock.appendChild(btn);
            });
        }

        const shortRoomNames = {
            "loc.central_harbor_square":  "Harbor",
            "loc.market_street":          "Market",
            "loc.royal_blacksmith_forge": "Forge",
            "loc.apothecary_shop":        "Apothecary",
            "loc.the_salty_siren_tavern": "Tavern",
            "loc.the_great_pier":         "Pier",
            "loc.abandoned_warehouse":    "Warehouse",
            "loc.north_town_gate":        "Gate",
            "loc.whispering_forest_edge": "Forest"
        };

        const exits = room.exits || {};
        ['north', 'east', 'south', 'west'].forEach(dir => {
            const btn = document.getElementById(`comp-btn-${dir}`);
            if (!btn) return;
            const targetId = exits[dir];
            if (targetId) {
                const tMeta = GameState.getRoomMeta(targetId);
                const shortLabel = shortRoomNames[targetId] || targetId.replace(/^[^.]+\./, '').split('_')[0];
                const capLabel = shortLabel.charAt(0).toUpperCase() + shortLabel.slice(1);
                const pinIcon = tMeta.iconImg
                    ? `<img src="${tMeta.iconImg}" alt="" class="comp-sq-icon">`
                    : `<span style="font-size:18px;">${tMeta.icon || '🧭'}</span>`;

                btn.classList.remove('hidden');
                btn.dataset.cmd = `MOVE ${dir}`;
                btn.title = `${dir.toUpperCase()} → ${tMeta.name}`;
                btn.innerHTML = `
                    ${pinIcon}
                    <span class="comp-sq-label">${capLabel}</span>
                `;
            } else {
                btn.classList.add('hidden');
                delete btn.dataset.cmd;
            }
        });
    }
};