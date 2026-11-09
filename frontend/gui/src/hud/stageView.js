// ==========================================================================
// HUD MODULE: WORLD STAGE, LEFT NPC BAR (Talk/Attack) & COMPASS WIDGET
// ==========================================================================

const StageView = {
    renderStage() {
        const room = GameState.room;
        const meta = ASSETS.rooms[room.id] || ASSETS.rooms.default;

        const hudRoomEl = document.getElementById('hud-room-name');
        if (hudRoomEl) hudRoomEl.textContent = room.name;

        // 1. Location icon in the top banner
        const stageIconEl = document.getElementById('stage-icon');
        if (meta.iconImg) {
            stageIconEl.innerHTML = `<img src="${meta.iconImg}" alt="${room.name} icon" class="stage-banner-icon-img">`;
        } else {
            stageIconEl.textContent = meta.icon;
        }

        document.getElementById('stage-room-title').textContent = room.name;
        document.getElementById('stage-room-region').textContent = meta.region;

        // 2. Main scene background
        const bgImg = document.getElementById('stage-bg-img');
        const worldStage = document.getElementById('world-stage');
        if (bgImg && meta.img) {
            bgImg.src = meta.img;
            bgImg.alt = `${room.name} — ${meta.region}`;
        } else if (worldStage && meta.img) {
            worldStage.style.backgroundImage = `url('${meta.img}')`;
        }

        // 3. LEFT NPC DOCK (Under Hero Avatar: Green = TALK, Red = ATTACK!)
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

        // 4. COMPASS NAVIGATION (Square Right-Menu style buttons: Icon on top + 1 Short Word below!)
        const shortRoomNames = {
            "harbor_square":    "Harbor",
            "market_street":    "Market",
            "blacksmith_forge": "Forge",
            "apothecary":       "Apothecary",
            "tavern":           "Tavern",
            "docks":            "Docks",
            "warehouse":        "Warehouse",
            "town_gate":        "Gate",
            "forest_edge":      "Forest"
        };

        const exits = room.exits || {};
        ['north', 'east', 'south', 'west'].forEach(dir => {
            const btn = document.getElementById(`comp-btn-${dir}`);
            if (!btn) return;
            const targetId = exits[dir];
            if (targetId) {
                const tMeta = ASSETS.rooms[targetId] || ASSETS.rooms.default;
                const shortLabel = shortRoomNames[targetId] || targetId.split('_')[0];
                const capLabel = shortLabel.charAt(0).toUpperCase() + shortLabel.slice(1);
                const pinIcon = tMeta.iconImg
                    ? `<img src="${tMeta.iconImg}" alt="" class="comp-sq-icon">`
                    : `<span style="font-size:18px;">${tMeta.icon || '🧭'}</span>`;

                btn.classList.remove('hidden');
                btn.dataset.cmd = `MOVE ${dir}`;
                btn.title = `${dir.toUpperCase()} → ${targetId.replace(/_/g, ' ')}`;
                btn.innerHTML = `
                    ${pinIcon}
                    <span class="comp-sq-label">${capLabel}</span>
                `;
            } else {
                btn.classList.add('hidden');
                delete btn.dataset.cmd;
            }
        });

        // 5. Urgent Aid Button if an ally called COMBAT HELP
        const aidSlot = document.getElementById('compass-aid-slot');
        if (aidSlot) {
            aidSlot.innerHTML = '';
            if (GameState.activeHelpCombatId && !GameState.inCombat) {
                const helpBtn = document.createElement('button');
                helpBtn.type = 'button';
                helpBtn.className = 'btn-gold-sm';
                helpBtn.dataset.cmd = `COMBAT JOIN ${GameState.activeHelpCombatId}`;
                helpBtn.innerHTML = `⚔️ Aid ${GameState.activeHelpPlayer || 'Ally'}`;
                aidSlot.appendChild(helpBtn);
            }
        }
    }
};