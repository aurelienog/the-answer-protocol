// ==========================================================================
// MODAL MODULE: FULLSCREEN COMBAT ARENA & TURN-BY-TURN ANIMATION QUEUE
// ==========================================================================

const CombatModal = {
    enemyId: null,
    enemyName: "Enemy",
    enemyHp: 50,
    enemyMaxHp: 50,

    animQueue: [],
    isPlayingAnim: false,
    isMyTurn: true,
    isFallen: false,
    turnEventCounter: 0,
    deathCheckTimer: null,
    activeTurnOwner: "",
    currentRound: 1,
    turnSeconds: 30,
    timerInterval: null,

    startTurnTimer(roundNum, resetClock = true) {
        if (roundNum !== undefined && roundNum > 0) {
            this.currentRound = roundNum;
            sessionStorage.setItem('tap_combat_round', String(this.currentRound));
        } else {
            const savedRound = parseInt(sessionStorage.getItem('tap_combat_round'), 10);
            if (!isNaN(savedRound) && savedRound > 0) this.currentRound = savedRound;
        }

        let deadline = parseInt(sessionStorage.getItem('tap_combat_deadline'), 10);
        if (resetClock || isNaN(deadline) || deadline <= Date.now()) {
            deadline = Date.now() + 30000;
            sessionStorage.setItem('tap_combat_deadline', String(deadline));
        }

        const tick = () => {
            if (!GameState.inCombat) {
                clearInterval(this.timerInterval);
                return;
            }
            const remainingMs = deadline - Date.now();
            this.turnSeconds = Math.max(0, Math.ceil(remainingMs / 1000));
            this.updateTimerUI();
        };

        tick();
        if (this.timerInterval) clearInterval(this.timerInterval);
        this.timerInterval = setInterval(tick, 250);
    },

    updateTimerUI() {
        const rEl = document.getElementById('combat-round-label');
        const tEl = document.getElementById('combat-timer-label');
        if (rEl) rEl.textContent = `Round ${this.currentRound}`;
        if (tEl) tEl.textContent = `${this.turnSeconds}s`;
    },

    updateBannerFromOwner() {
        const turnEl = document.getElementById('combat-turn-text');
        if (!turnEl) return;

        const owner = this.activeTurnOwner || GameState.playerName;
        if (this.isFallen) {
            this.isMyTurn = false;
            turnEl.textContent = `💀 SPECTATING (${owner.toUpperCase()})`;
            return;
        }

        this.isMyTurn = (owner.toLowerCase() === GameState.playerName.toLowerCase());
        turnEl.textContent = this.isMyTurn ? 'YOUR TURN' : `${owner.toUpperCase()}'S TURN`;
    },

    initListeners() {
        document.getElementById('btn-combat-strike').addEventListener('click', () => {
            if (!this.enemyId || this.isPlayingAnim || !this.isMyTurn || this.isFallen) return;
            this.isMyTurn = false;
            this.setButtonsDisabled(true);
            App.sendCommand(`COMBAT ATTACK ${this.enemyId}`);
        });

        const defendBtn = document.getElementById('btn-combat-defend');
        if (defendBtn) {
            defendBtn.addEventListener('click', () => {
                if (this.isPlayingAnim || !this.isMyTurn || this.isFallen) return;
                this.isMyTurn = false;
                this.setButtonsDisabled(true);
                App.sendCommand('COMBAT DEFEND');
            });
        }

        const healBtn = document.getElementById('btn-combat-heal');
        if (healBtn) {
            healBtn.addEventListener('click', () => {
                if (this.isPlayingAnim || !this.isMyTurn || this.isFallen) return;
                const healOrder = ['item.healing_potion', 'item.wine', 'item.bread'];
                const itemToUse = healOrder.find(id => GameState.inventory.includes(id));
                if (!itemToUse) {
                    UI.addChatLine('ROOM', '🧪 No Consumables', 'You have no Healing Elixir, Wine, or Bread in your bag!', 'chat-err');
                    return;
                }
                App.sendCommand(`COMBAT USE ${itemToUse}`);
            });
        }

        const helpBtn = document.getElementById('btn-combat-help');
        if (helpBtn) {
            helpBtn.addEventListener('click', () => {
                if (this.isPlayingAnim || this.isFallen) return;
                App.sendCommand('COMBAT HELP');
            });
        }

        const fleeModal = document.getElementById('modal-flee');
        document.getElementById('btn-combat-flee').addEventListener('click', () => {
            if (this.isPlayingAnim || !this.isMyTurn || this.isFallen) return;
            fleeModal.classList.remove('hidden');
        });
        document.getElementById('btn-close-flee').addEventListener('click', () => {
            fleeModal.classList.add('hidden');
        });
        document.getElementById('btn-cancel-flee').addEventListener('click', () => {
            fleeModal.classList.add('hidden');
        });
        document.getElementById('btn-confirm-flee').addEventListener('click', () => {
            fleeModal.classList.add('hidden');
            this.isMyTurn = false;
            this.setButtonsDisabled(true);
            App.sendCommand('COMBAT FLEE');
        });
    },

    enqueueAnim(fn, durationMs = 1200) {
        this.animQueue.push({ fn, durationMs });
        if (!this.isPlayingAnim) {
            this.processNextAnim();
        }
    },

    processNextAnim() {
        if (this.animQueue.length === 0) {
            this.isPlayingAnim = false;
            this.updateBannerFromOwner();
            this.syncActionButtonsState();
            return;
        }

        this.isPlayingAnim = true;
        this.setButtonsDisabled(true);

        const step = this.animQueue.shift();
        step.fn();

        setTimeout(() => {
            this.processNextAnim();
        }, step.durationMs);
    },

    syncActionButtonsState() {
        const lockTurnActions = this.isPlayingAnim || !this.isMyTurn || this.isFallen;
        ['btn-combat-strike', 'btn-combat-defend', 'btn-combat-heal', 'btn-combat-flee'].forEach(id => {
            const btn = document.getElementById(id);
            if (btn) btn.disabled = lockTurnActions;
        });
        const helpBtn = document.getElementById('btn-combat-help');
        if (helpBtn) helpBtn.disabled = this.isPlayingAnim || this.isFallen;
    },

    setButtonsDisabled(disabled) {
        ['btn-combat-strike', 'btn-combat-defend', 'btn-combat-heal', 'btn-combat-help', 'btn-combat-flee'].forEach(id => {
            const btn = document.getElementById(id);
            if (btn) btn.disabled = disabled;
        });
    },

    open(npcIdentifier, initialHeroHp, initialEnemyHp, isRestoring = false) {
        const alreadyOpen = GameState.inCombat && !document.getElementById('modal-combat').classList.contains('hidden');

        UI.closeAllModals();
        UI.activeModalId = 'combat';
        GameState.inCombat = true;

        let npcMeta = GameState.getNpcMeta(npcIdentifier);
        if (npcMeta.id === 'default') {
            const saved = sessionStorage.getItem('tap_active_combat_enemy');
            if (saved && saved !== 'default') {
                npcMeta = GameState.getNpcMeta(saved);
            }
        }

        this.enemyId = npcMeta.id;
        this.enemyName = npcMeta.name;
        if (this.enemyId && this.enemyId !== 'default') {
            sessionStorage.setItem('tap_active_combat_enemy', this.enemyId);
        }

        const defaultMax = npcMeta.maxHp || npcMeta.hp || 60;
        const savedEnemyHp = parseInt(sessionStorage.getItem('tap_active_combat_enemy_hp'), 10);

        this.enemyMaxHp = (initialEnemyHp !== undefined && initialEnemyHp > defaultMax)
            ? initialEnemyHp
            : defaultMax;

        this.enemyHp = initialEnemyHp !== undefined
            ? initialEnemyHp
            : (!isNaN(savedEnemyHp) && savedEnemyHp > 0 ? savedEnemyHp : this.enemyMaxHp);

        if (initialHeroHp !== undefined) {
            GameState.hp = initialHeroHp;
        }

        this.isFallen = isRestoring ? (GameState.hp <= 0) : false;

        const savedTurnOwner = sessionStorage.getItem('tap_combat_turn_owner');
        if (savedTurnOwner) {
            this.activeTurnOwner = savedTurnOwner;
            this.updateBannerFromOwner();
            this.startTurnTimer(undefined, !isRestoring && !alreadyOpen);
        } else {
            this.activeTurnOwner = GameState.playerName;
            this.updateBannerFromOwner();
            this.startTurnTimer(1, true);
        }

        const roomMeta = ASSETS.rooms[GameState.room.id] || ASSETS.rooms.default;
        const combatBg = document.getElementById('combat-bg-img');
        if (combatBg && roomMeta.img) {
            combatBg.src = roomMeta.img;
            combatBg.alt = `Battle in ${GameState.room.name}`;
        }

        document.getElementById('combat-hero-name').textContent = GameState.playerName;
        document.getElementById('combat-enemy-name').textContent = this.enemyName;
        UI.setArt(document.getElementById('combat-enemy-avatar'), npcMeta.iconImg, npcMeta.icon);

        const heroSprite = document.getElementById('combat-hero-sprite');
        const heroFallback = document.getElementById('combat-hero-fallback');
        heroFallback.textContent = GameState.gender === 'male' ? '🧙‍♂️' : '🧝‍♀️';
        heroSprite.style.display = 'block';
        heroFallback.style.display = 'none';
        heroSprite.src = `/assets/combat/hero_${GameState.gender}.png`;

        const enemySprite = document.getElementById('combat-enemy-sprite');
        const enemyFallback = document.getElementById('combat-enemy-fallback');
        enemyFallback.textContent = npcMeta.icon;
        enemySprite.style.display = 'block';
        enemyFallback.style.display = 'none';
        enemySprite.src = `/assets/combat/${this.enemyId}.png`;

        document.getElementById('combat-outcome-banner').classList.add('hidden');
        document.getElementById('fighter-enemy-box').style.opacity = '1';
        this.syncActionButtonsState();

        this.updateBars();
        FrameView.mountButtons();
        FrameView.mountHpBarFrames();
        document.getElementById('modal-combat').classList.remove('hidden');
    },

    updateBars() {
        const heroPct = Math.max(0, Math.min(100, Math.round((GameState.hp / GameState.maxHp) * 100)));
        document.getElementById('combat-hero-hp-fill').style.width = heroPct + '%';
        document.getElementById('combat-hero-hp-text').textContent = `${GameState.hp} / ${GameState.maxHp}`;

        const enemyPct = Math.max(0, Math.min(100, Math.round((this.enemyHp / this.enemyMaxHp) * 100)));
        document.getElementById('combat-enemy-hp-fill').style.width = enemyPct + '%';
        document.getElementById('combat-enemy-hp-text').textContent = `${this.enemyHp} / ${this.enemyMaxHp}`;

        if (GameState.inCombat && this.enemyHp > 0) {
            sessionStorage.setItem('tap_active_combat_enemy_hp', String(this.enemyHp));
        }
        TopHudView.updateHUDStats();
    },

    playDefendEffect(reductionPct) {
        this.enqueueAnim(() => {
            const turnText = document.getElementById('combat-turn-text');
            if (turnText) turnText.textContent = 'DEFENDING 🛡️';
            this.showFloatingDmg('dmg-on-hero', `🛡️ +${reductionPct}% DEF`, '#63b3ed');
        }, 1250);
    },

    playHealEffect(healAmount) {
        this.enqueueAnim(() => {
            GameState.hp = Math.min(GameState.maxHp, GameState.hp + healAmount);
            this.showFloatingDmg('dmg-on-hero', `+${healAmount} HP`, '#48bb78');
            this.updateBars();
        }, 1250);
    },

    playHitEffect(mode, damageAmount, remainingHp, customBannerText = '', allyTargetName = '') {
        this.enqueueAnim(() => {
            const heroBox = document.getElementById('fighter-hero-box');
            const enemyBox = document.getElementById('fighter-enemy-box');
            const turnText = document.getElementById('combat-turn-text');

            if (mode === 'enemy_hit' || mode === true) {
                if (remainingHp !== undefined) this.enemyHp = remainingHp;
                if (turnText) turnText.textContent = customBannerText || 'STRIKE ⚔️';
                this.triggerAnim(heroBox, 'lunge-right');
                this.triggerAnim(enemyBox, 'hit-shake');
                this.showFloatingDmg('dmg-on-enemy', `-${damageAmount}`, '#ff4d4d');
            } else if (mode === 'me_hit' || mode === false) {
                if (remainingHp !== undefined) GameState.hp = remainingHp;
                if (turnText) turnText.textContent = `${this.enemyName.toUpperCase()} ATTACKS YOU!`;
                this.triggerAnim(enemyBox, 'lunge-left');
                this.triggerAnim(heroBox, 'hit-shake');
                this.showFloatingDmg('dmg-on-hero', `-${damageAmount}`, '#ff4d4d');
            } else if (mode === 'ally_hit') {
                if (turnText) turnText.textContent = `${this.enemyName.toUpperCase()} HITS ${allyTargetName.toUpperCase()}!`;
                this.triggerAnim(enemyBox, 'lunge-left');
                this.showFloatingDmg('dmg-on-hero', `🛡️ ${allyTargetName}: -${damageAmount} HP`, '#63b3ed');
            }

            this.updateBars();
        }, 1350);
    },

    queueTurnBanner(participantName, roundNum) {
        this.turnEventCounter = (this.turnEventCounter || 0) + 1;
        const cleanOwner = (participantName || '').trim();
        this.activeTurnOwner = cleanOwner;
        sessionStorage.setItem('tap_combat_turn_owner', cleanOwner);
        sessionStorage.setItem('tap_combat_deadline', String(Date.now() + 30000));
        if (roundNum !== undefined && roundNum > 0) {
            sessionStorage.setItem('tap_combat_round', String(roundNum));
        }

        this.enqueueAnim(() => {
            this.activeTurnOwner = cleanOwner;
            this.updateBannerFromOwner();
            this.startTurnTimer(roundNum, false);
            this.syncActionButtonsState();
        }, 250);
    },

    triggerAnim(el, className) {
        if (!el) return;
        el.classList.remove(className);
        void el.offsetWidth;
        el.classList.add(className);
        setTimeout(() => el.classList.remove(className), 550);
    },

    showFloatingDmg(elId, text, color = '#ff4d4d') {
        const el = document.getElementById(elId);
        if (!el) return;
        el.textContent = text;
        el.style.color = color;
        el.classList.remove('hidden');
        void el.offsetWidth;
        setTimeout(() => el.classList.add('hidden'), 1300);
    },

    handlePlayerDeath(diedIsMe, fallenName) {
        if (diedIsMe) {
            this.isFallen = true;
            this.isMyTurn = false;
            GameState.hp = 0;
            this.updateBars();
            this.setButtonsDisabled(true);
            this.updateBannerFromOwner();
        }

        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);
        const turnCountBefore = this.turnEventCounter || 0;

        this.deathCheckTimer = setTimeout(() => {
            if (!GameState.inCombat) return;
            const turnCountAfter = this.turnEventCounter || 0;

            if (turnCountAfter === turnCountBefore) {
                this.showOutcome(false, 'Your party was defeated! Respawning at Harbor Square...');
                App.sendCommand('LOOK');
                App.sendCommand('STATUS');
            } else if (diedIsMe) {
                UI.addChatLine('ROOM', '💀 Spectating', 'You fell in battle! Spectating your ally until the fight ends...', 'chat-err');
            }
        }, 450);
    },

    showOutcome(victory, subtitle) {
        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);

        this.enqueueAnim(() => {
            this.isMyTurn = false;
            this.setButtonsDisabled(true);
            const banner = document.getElementById('combat-outcome-banner');
            document.getElementById('combat-outcome-title').textContent = victory ? '🏆 VICTORY!' : '💀 DEFEATED';
            document.getElementById('combat-outcome-sub').textContent = subtitle;
            banner.classList.remove('hidden');

            if (victory) {
                document.getElementById('fighter-enemy-box').style.opacity = '0.25';
            } else {
                App.sendCommand('STATUS');
                App.sendCommand('LOOK');
            }

            setTimeout(() => {
                this.close();
                App.sendCommand('STATUS');
            }, 3500);
        }, 3600);
    },

    close() {
        sessionStorage.removeItem('tap_active_combat_enemy');
        sessionStorage.removeItem('tap_active_combat_enemy_hp');
        sessionStorage.removeItem('tap_combat_deadline');
        sessionStorage.removeItem('tap_combat_round');
        sessionStorage.removeItem('tap_combat_turn_owner');

        if (this.timerInterval) clearInterval(this.timerInterval);
        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);

        GameState.inCombat = false;
        this.animQueue = [];
        this.isPlayingAnim = false;
        this.isMyTurn = true;
        this.isFallen = false;
        this.activeTurnOwner = "";
        document.getElementById('modal-flee').classList.add('hidden');
        document.getElementById('modal-combat').classList.add('hidden');
        if (UI.activeModalId === 'combat') UI.activeModalId = null;
    }
};