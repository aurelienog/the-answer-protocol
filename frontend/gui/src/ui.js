// ==========================================================================
// MASTER UI CONTROLLER — Coordinates clicks and delegates to specific modules
// ==========================================================================

const UI = {
    activeModalId: null,

    initEventListeners() {
        FrameView.mountFrames();
        CombatModal.initListeners();
        AudioManager.init();
        document.querySelectorAll('.js-sound-toggle').forEach(btn => {
            btn.addEventListener('click', () => AudioManager.toggleMute());
        });
        const btnNextTrack = document.getElementById('btn-next-track');
        if (btnNextTrack) {
            btnNextTrack.addEventListener('click', () => AudioManager.nextTrack());
        }

        document.getElementById('login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            App.connect();
        });

        this.applyGenderAvatar();

        document.querySelectorAll('input[name="gender"]').forEach(radio => {
            radio.checked = (radio.value === GameState.gender);
            radio.addEventListener('change', (e) => {
                GameState.gender = e.target.value;
                localStorage.setItem('tap_gender', GameState.gender);
                this.applyGenderAvatar();
            });
        });

        document.getElementById('chat-form').addEventListener('submit', (e) => {
            e.preventDefault();
            App.sendChatMessage();
        });

        // Dock controls: Expand Chat (+30%) & Independent Minimize/Restore for Chat and Players panels
        const chatPanel = document.getElementById('dock-chat-panel');
        const playersPanel = document.getElementById('dock-players-panel');
        const btnChatTall = document.getElementById('btn-chat-tall');
        const btnChatCollapse = document.getElementById('btn-chat-collapse');
        const btnPlayersCollapse = document.getElementById('btn-players-collapse');

        if (btnChatTall && chatPanel) {
            btnChatTall.addEventListener('click', () => {
                chatPanel.classList.remove('dock-collapsed');
                if (btnChatCollapse) btnChatCollapse.textContent = '▾';
                const isTall = chatPanel.classList.toggle('chat-tall');
                btnChatTall.classList.toggle('active', isTall);
            });
        }

        if (btnChatCollapse && chatPanel) {
            btnChatCollapse.addEventListener('click', () => {
                const collapsed = chatPanel.classList.toggle('dock-collapsed');
                btnChatCollapse.textContent = collapsed ? '▴' : '▾';
            });
        }

        if (btnPlayersCollapse && playersPanel) {
            btnPlayersCollapse.addEventListener('click', () => {
                const collapsed = playersPanel.classList.toggle('dock-collapsed');
                btnPlayersCollapse.textContent = collapsed ? '▴' : '▾';
            });
        }

        document.getElementById('btn-sel-confirm').addEventListener('click', () => {
            SelectorModal.confirmAction();
        });

        document.getElementById('btn-drop-selected').addEventListener('click', () => InventoryModal.dropSelected());
        document.getElementById('btn-use-selected').addEventListener('click', () => InventoryModal.useSelected());

        document.getElementById('dlg-btn-talk-again').addEventListener('click', () => {
            if (DialogueModal.activeNpcId) App.sendCommand(`TALK ${DialogueModal.activeNpcId}`);
        });
        document.getElementById('dlg-btn-quest').addEventListener('click', () => {
            if (!DialogueModal.activeNpcId) return;
            if (DialogueModal.activeQuestStatus === 'available') {
                App.sendCommand(`QUEST ${DialogueModal.activeNpcId}`);
            } else {
                // If quest is active, talking to the NPC checks progress or turns in the item!
                App.sendCommand(`TALK ${DialogueModal.activeNpcId}`);
            }
        });

        // Global Event Delegation
        document.addEventListener('click', (e) => {
            if (e.target.closest('[data-close-modals]')) {
                this.closeAllModals();
                return;
            }

            const dockTabBtn = e.target.closest('[data-dock-tab]');
            if (dockTabBtn) {
                PlayersDockView.switchRightDock(dockTabBtn.dataset.dockTab);
                return;
            }

            const winBtn = e.target.closest('[data-open-window]');
            if (winBtn) {
                this.openWindow(winBtn.dataset.openWindow);
                return;
            }

            const lookTabBtn = e.target.closest('[data-look-tab]');
            if (lookTabBtn) {
                LookModal.switchTab(lookTabBtn.dataset.lookTab);
                return;
            }

            const cmdBtn = e.target.closest('[data-cmd]');
            if (cmdBtn) {
                const cmd = cmdBtn.dataset.cmd || '';
                if (GameState.inCombat && cmd.toUpperCase().startsWith('MOVE ')) {
                    UI.addChatLine('ROOM', '⚔️ In Combat', 'You cannot leave the room during combat! Use "🏃 Flee" to retreat.', 'chat-err');
                    return;
                }
                App.sendCommand(cmd);
                return;
            }

            const chatScopeBtn = e.target.closest('[data-chat-scope]');
            if (chatScopeBtn) {
                ChatView.switchChatScope(chatScopeBtn.dataset.chatScope);
                return;
            }

            const groupActBtn = e.target.closest('[data-group-act]');
            if (groupActBtn) {
                GroupModal.handleAction(groupActBtn.dataset.groupAct);
            }

            const questTabBtn = e.target.closest('[data-quest-tab]');
            if (questTabBtn) {
                QuestsModal.switchTab(questTabBtn.dataset.questTab);
                return;
            }
        });
    },

    closeAllModals() {
        document.querySelectorAll('.rpg-modal').forEach(m => m.classList.add('hidden'));
        document.querySelectorAll('.right-action-bar .action-btn').forEach(b => b.classList.remove('active'));
        this.activeModalId = null;
    },

    openWindow(type) {
        if (this.activeModalId === type) {
            this.closeAllModals();
            return;
        }

        this.closeAllModals();
        this.activeModalId = type;

        const sideBtn = document.querySelector(`.right-action-bar [data-open-window="${type}"]`);
        if (sideBtn) sideBtn.classList.add('active');

        switch (type) {
            case 'look':
                document.getElementById('modal-look').classList.remove('hidden');
                App.sendCommand('LOOK');
                this.renderRoom();
                break;
            case 'move':
            case 'take':
            case 'talk':
            case 'attack':
                SelectorModal.open(type);
                break;
            case 'inventory':
                document.getElementById('modal-inventory').classList.remove('hidden');
                App.sendCommand('INVENTORY');
                InventoryModal.render();
                break;
            case 'status':
                document.getElementById('modal-status').classList.remove('hidden');
                App.sendCommand('STATUS');
                App.sendCommand('WHO');
                StatusModal.render();
                break;
            case 'quests':
                document.getElementById('modal-quests').classList.remove('hidden');
                App.sendCommand('QUESTS');
                QuestsModal.render();
                break;
            case 'group':
                document.getElementById('modal-group').classList.remove('hidden');
                if (GameState.currentGroup) App.sendCommand('GROUP INFO');
                App.sendCommand('GROUP INVITATIONS');
                GroupModal.renderList();
                break;
            case 'quit':
                document.getElementById('modal-quit').classList.remove('hidden');
                break;
        }
    },

    // A versatile helper: it displays an image (img or iconImg), or shows an emoji if none is available!
    setArt(el, imgUrl, fallbackEmoji = '', altText = 'Game illustration') {
        if (!el) return;
        const src = imgUrl || '/assets/ui/card/look-items.jpeg';

        if (el.tagName === 'IMG') {
            el.src = src;
            el.alt = altText || 'Illustration';
            el.classList.toggle('contain-img', src.endsWith('.png'));
            return;
        }

        // If it is a container of the .card-thumb type within a list, place a clean <img> element inside it, without any inline styles.
        if (imgUrl) {
            const fitClass = imgUrl.endsWith('.png') ? 'art-img contain' : 'art-img cover';
            el.innerHTML = `<img src="${imgUrl}" alt="${altText}" class="${fitClass}">`;
        } else {
            el.textContent = fallbackEmoji;
        }
    },

    applyGenderAvatar() {
        const av = ASSETS.avatars[GameState.gender] || ASSETS.avatars.male;
        // 1. Round icon in the top-left corner of the HUD and in Battle -> use iconImg!
        this.setArt(document.getElementById('hud-avatar'), av.iconImg, av.icon);
        this.setArt(document.getElementById('combat-hero-avatar'), av.iconImg, av.icon);

        // 2. Large portrait in the STATUS window on the left -> let's take the large image (img)!
        this.setArt(document.getElementById('status-hero-art'), av.img, av.icon);
    },

    showLoginError(msg) {
        const el = document.getElementById('login-error');
        if (!msg) {
            el.classList.add('hidden');
            return;
        }
        el.textContent = msg;
        el.classList.remove('hidden');
    },

    enterGameScreen() {
        this.applyGenderAvatar();
        
        document.getElementById('login-screen').classList.add('hidden');
        document.getElementById('game-screen').classList.remove('hidden');
        document.getElementById('hud-player-name').textContent = GameState.playerName;
        document.getElementById('status-char-name').textContent = GameState.playerName;
    },

    returnToLogin(reason) {
        this.closeAllModals();
        document.getElementById('game-screen').classList.add('hidden');
        document.getElementById('login-screen').classList.remove('hidden');
        if (reason) this.showLoginError(reason);
    },

    // Unified Facade methods called by network.js (App)
    renderRoom() {
        StageView.renderStage();
        LookModal.render();
        PlayersDockView.renderPlayersInRoom();

        if (this.activeModalId === 'take') {
            if (GameState.itemsInRoom.length > 0) {
                SelectorModal.open('take');
            } else {
                this.closeAllModals();
            }
        }
    },
    renderPlayersInRoom()   { PlayersDockView.renderPlayersInRoom(); },
    updateHUDStats()        { TopHudView.updateHUDStats(); },
    addChatLine(s, a, t, c) { ChatView.addChatLine(s, a, t, c); },
    renderInventory()       { InventoryModal.render(); },
    renderQuests()          { QuestsModal.render(); }
};

document.addEventListener('DOMContentLoaded', () => {
    UI.initEventListeners();
    App.restoreSessionIfActive();
});