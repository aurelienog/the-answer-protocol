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
            const isMe = name.toLowerCase() === GameState.playerName.toLowerCase();
            const alreadyInMyGroup = groupMembers.some(m => m.toLowerCase() === name.toLowerCase());

            let partyCell = '-';
            if (!isMe && GameState.activeHelpCombatId && GameState.activeHelpPlayer === name) {
                partyCell = `<button type="button" class="btn-gold-sm" data-cmd="COMBAT JOIN ${GameState.activeHelpCombatId}">⚔️ Aid</button>`;
            } else if (isMe) {
                partyCell = inGroup ? '<span class="badge">In Party</span>' : '-';
            } else if (alreadyInMyGroup) {
                partyCell = '<span class="badge">Member</span>';
            } else if (inGroup) {
                partyCell = `<button type="button" class="btn-gold-sm" data-cmd="GROUP INVITE ${name}"><img src="/assets/icons/ui/invite.png" alt="" class="tab-mini-icon">Invite</button>`;
            }

            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td>🟢 <strong>${name}</strong> ${isMe ? '<small style="color:var(--text-muted)">(You)</small>' : ''}</td>
                <td class="col-wide-only">Adventurer</td>
                <td class="col-wide-only">In Room</td>
                <td>${partyCell}</td>
            `;
            tbody.appendChild(tr);

            if (lPlayers) {
                const av = isMe
                    ? (ASSETS.avatars[GameState.gender] || ASSETS.avatars.male)
                    : ASSETS.avatars.male;
                const li = document.createElement('li');
                li.className = 'select-card static-card';
                li.innerHTML = `
                    <div class="card-left">
                        <div class="card-thumb"></div>
                        <div><strong class="card-title">${name} ${isMe ? '(You)' : ''}</strong><span class="card-sub">Adventurer</span></div>
                    </div>
                    <span class="badge">Online</span>
                `;
                UI.setArt(li.querySelector('.card-thumb'), av.iconImg || av.img, av.icon, name);
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

        gCount.textContent = grp.members.length;
        gBody.innerHTML = '';

        grp.members.forEach((member, idx) => {
          const isMe = member.toLowerCase() === GameState.playerName.toLowerCase();
          const icon = idx === 0 ? '👑' : '🛡️';
          const rank = idx === 0 ? 'Leader' : 'Member';
          const tr = document.createElement('tr');
          tr.innerHTML = `
              <td>${icon} <strong>${member}</strong> ${isMe ? '<small style="color:var(--text-muted)">(You)</small>' : ''}</td>
              <td class="col-wide-only">${rank}</td>
              <td><span class="badge">${grp.name}</span></td>
          `;
          gBody.appendChild(tr);
      });
    }
};