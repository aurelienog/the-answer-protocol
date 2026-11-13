// ==========================================================================
// MODAL MODULE: PARTY & GROUP MANAGEMENT
// ==========================================================================

const GroupModal = {
    handleAction(type) {
        if (type === 'CREATE' || type === 'JOIN') {
            const nameInput = document.getElementById('group-name-input');
            const groupName = nameInput ? nameInput.value.trim() : '';
            if (!groupName) return;
            App.sendCommand(`GROUP ${type} ${groupName}`);
            return;
        }
        if (type === 'INVITE') {
            const playerInput = document.getElementById('group-modal-input');
            const target = playerInput ? playerInput.value.trim() : '';
            if (!target) return;
            App.sendCommand(`GROUP INVITE ${target}`);
            playerInput.value = '';
        }
    },

    renderInvitations(invitations) {
        GameState.groupInvitations = invitations || [];
        this.renderList();
    },

    renderCurrentGroup(group) {
        if (!group) {
            GameState.currentGroup = null;
        } else {
            GameState.currentGroup = {
                id: group.ID || group.id || '',
                name: group.Name || group.name || 'Party',
                members: group.Members || group.members || [GameState.playerName]
            };
        }
        this.renderList();
        PlayersDockView.renderPlayersInRoom();
    },

    renderList() {
        const box = document.getElementById('group-invitations-list');
        if (!box) return;
        box.innerHTML = '';

        if (GameState.currentGroup) {
            const grp = GameState.currentGroup;
            const li = document.createElement('li');
            li.className = 'select-card static-card selected';
            li.innerHTML = `
                <div class="card-left">
                    <div class="card-thumb">
                        <img src="/assets/icons/ui/crown.webp" alt="Leader" class="art-img contain" style="padding:5px;">
                    </div>
                    <div>
                        <strong class="card-title">Active Party: ${grp.name}</strong>
                        <span class="card-sub">Members (${grp.members.length}): ${grp.members.join(', ')}</span>
                    </div>
                </div>
                <span class="badge">Active</span>
            `;
            box.appendChild(li);
        }

        const invs = (GameState.groupInvitations || []).filter(inv => {
            const invName = (inv.Name || inv.name || '').toLowerCase();
            const curName = (GameState.currentGroup && GameState.currentGroup.name || '').toLowerCase();
            return invName !== curName;
        });

        if (invs.length === 0 && !GameState.currentGroup) {
            box.innerHTML = '<li class="card-sub" style="padding:6px 4px;">No active group or pending invitations.</li>';
        } else {
            invs.forEach(inv => {
                const invName = inv.Name || inv.name;
                const li = document.createElement('li');
                li.className = 'select-card static-card';
                li.innerHTML = `
                    <div class="card-left">
                        <div class="card-thumb">
                            <img src="/assets/icons/ui/shield.webp" alt="Invite" class="art-img contain" style="padding:5px;">
                        </div>
                        <div>
                            <strong class="card-title">Invite to: ${invName}</strong>
                            <span class="card-sub">Group ID: ${inv.ID || inv.id}</span>
                        </div>
                    </div>
                    <button type="button" class="btn-gold-sm btn-take-look" data-cmd="GROUP JOIN ${invName}">Accept & Join</button>
                `;
                box.appendChild(li);
            });
        }

        FrameView.mountButtons();
        FrameView.mountSlotFrames();
    }
};