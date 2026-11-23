// ==========================================================================
// HUD MODULE: COMPACT 2-COLUMN PLAYERS IN LOCATION & PARTY MEMBERS DOCK
// ==========================================================================

const PlayersDockView = {
    switchRightDock(tab) {
        document.getElementById('tab-btn-players').classList.toggle('active', tab === 'players');
        document.getElementById('tab-btn-group').classList.toggle('active', tab === 'group');
        document.getElementById('dock-players-view').classList.toggle('hidden', tab !== 'players');
        document.getElementById('dock-group-view').classList.toggle('hidden', tab !== 'group');
        if (tab === 'group' && GameState.currentGroup) {
            App.sendCommand('GROUP INFO');
        }
        this.renderGroupDock();
    },

    renderPlayersInRoom() {
        const players = GameState.playersInRoom.length > 0
            ? GameState.playersInRoom
            : [GameState.playerName];

        document.getElementById('hud-room-count').textContent = players.length;
        document.getElementById('dock-room-count').textContent = players.length;

        const tbody = document.getElementById('players-table-body');
        const lPlayers = document.getElementById('look-players-list');
        tbody.innerHTML = '';
        if (lPlayers) lPlayers.innerHTML = '';

        const inGroup = !!GameState.currentGroup;
        const groupMembers = (GameState.currentGroup && GameState.currentGroup.members) || [];

        players.forEach(name => {
            const safeName = UI.escapeHtml(name);
            const isMe = name.toLowerCase() === GameState.playerName.toLowerCase();
            const alreadyInMyGroup = groupMembers.some(m => m.toLowerCase() === name.toLowerCase());

            let partyCell = '-';
            let actionCmd = null;
            const isHelpTarget = !GameState.inCombat &&
                GameState.activeHelpCombatId &&
                GameState.activeHelpPlayer &&
                GameState.activeHelpPlayer.toLowerCase() === name.toLowerCase();
            if (!isMe && isHelpTarget) {
                partyCell = `<button type="button" class="btn-gold-sm js-player-act"><img src="/assets/icons/ui/sword.webp" alt="" class="tab-mini-icon">Aid</button>`;
                actionCmd = `COMBAT JOIN ${GameState.activeHelpCombatId}`;
            } else if (isMe) {
                partyCell = inGroup ? '<span class="badge">In Party</span>' : '-';
            } else if (alreadyInMyGroup) {
                partyCell = '<span class="badge">Member</span>';
            } else if (inGroup) {
                partyCell = `<button type="button" class="btn-gold-sm js-player-act"><img src="/assets/icons/ui/invite.png" alt="" class="tab-mini-icon">Invite</button>`;
                actionCmd = `GROUP INVITE ${name}`;
            }

            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td><img src="/assets/icons/ui/dot.webp" alt="" class="tab-mini-icon"> <strong>${safeName}</strong> ${isMe ? '<small style="color:var(--text-muted)">(You)</small>' : ''}</td>
                <td class="col-wide-only">Adventurer</td>
                <td class="col-wide-only">In Room</td>
                <td>${partyCell}</td>
            `;
            if (actionCmd) {
                const btn = tr.querySelector('.js-player-act');
                if (btn) btn.dataset.cmd = actionCmd;
            }
            tbody.appendChild(tr);

            if (lPlayers) {
                const li = document.createElement('li');
                li.className = 'select-card static-card';
                li.innerHTML = `
                    <div class="card-left">
                        <div class="card-thumb">
                            <img src="/assets/icons/ui/player.webp" alt="Player" class="art-img contain" style="padding:5px;">
                        </div>
                        <div><strong class="card-title">${safeName} ${isMe ? '(You)' : ''}</strong><span class="card-sub">Adventurer</span></div>
                    </div>
                    <span class="badge">Online</span>
                `;
                lPlayers.appendChild(li);
            }
        });

        this.renderGroupDock();
    },

    renderGroupDock() {
        const gBody = document.getElementById('group-table-body');
        const gCount = document.getElementById('dock-group-count');
        if (!gBody || !gCount) return;

        const grp = GameState.currentGroup;
        if (!grp || !grp.members || grp.members.length === 0) {
            gCount.textContent = '0';
            gBody.innerHTML = `<tr><td colspan="3" style="color:var(--text-muted);padding-top:10px;">Not in a party. Use GROUP menu to create or join.</td></tr>`;
            return;
        }

        const safeGroupName = UI.escapeHtml(grp.name);
        gCount.textContent = grp.members.length;
        gBody.innerHTML = '';

        grp.members.forEach((member, idx) => {
            const isMe = member.toLowerCase() === GameState.playerName.toLowerCase();
            const iconSrc = idx === 0 ? '/assets/icons/ui/crown.webp' : '/assets/icons/ui/shield.webp';
            const rank = idx === 0 ? 'Leader' : 'Member';
            const safeMember = UI.escapeHtml(member);

            const inSameRoom = (GameState.playersInRoom || []).some(p => p.toLowerCase() === member.toLowerCase());
            const canAid = !isMe && !GameState.inCombat && inSameRoom &&
                GameState.activeHelpCombatId &&
                GameState.activeHelpPlayer &&
                GameState.activeHelpPlayer.toLowerCase() === member.toLowerCase();

            const rightCell = canAid
                ? `<button type="button" class="btn-gold-sm js-party-aid"><img src="/assets/icons/ui/sword.webp" alt="" class="tab-mini-icon">Aid</button>`
                : `<span class="badge">${safeGroupName}</span>`;

            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td><img src="${iconSrc}" alt="" class="tab-mini-icon"> <strong>${safeMember}</strong> ${isMe ? '<small style="color:var(--text-muted)">(You)</small>' : ''}</td>
                <td class="col-wide-only">${rank}</td>
                <td>${rightCell}</td>
            `;
            if (canAid) {
                const btn = tr.querySelector('.js-party-aid');
                if (btn) btn.dataset.cmd = `COMBAT JOIN ${GameState.activeHelpCombatId}`;
            }
            gBody.appendChild(tr);
        });
    }
};