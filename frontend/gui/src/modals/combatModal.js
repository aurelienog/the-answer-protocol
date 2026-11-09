// ==========================================================================
// MODAL MODULE: FULLSCREEN COMBAT ARENA & TURN-BY-TURN ANIMATION QUEUE
// ==========================================================================

const CombatModal = {
    enemyId: null,
    enemyName: "Enemy",
    enemyHp: 50,
    enemyMaxHp: 50,

    // Turn-by-turn animation queue so player attack & enemy counter-attack play sequentially!
    animQueue: [],
    isPlayingAnim: false,
    currentRound: 1,
    turnSeconds: 30,
    timerInterval: null,

    startTurnTimer(roundNum) {
        if (roundNum !== undefined && roundNum > 0) {
            this.currentRound = roundNum;
        }
        this.turnSeconds = 30;
        this.updateTimerUI();

        if (this.timerInterval) clearInterval(this.timerInterval);
        this.timerInterval = setInterval(() => {
            if (!GameState.inCombat) {
                clearInterval(this.timerInterval);
                return;
            }
            if (this.turnSeconds > 0) {
                this.turnSeconds--;
                this.updateTimerUI();
            }
        }, 1000);
    },

    updateTimerUI() {
        const rEl = document.getElementById('combat-round-label');
        const tEl = document.getElementById('combat-timer-label');
        if (rEl) rEl.textContent = `Round ${this.currentRound}`;
        if (tEl) tEl.textContent = `${this.turnSeconds}s`;
    },

    initListeners() {
        // 1. Attack Button -> sends COMBAT ATTACK <npc>
        document.getElementById('btn-combat-strike').addEventListener('click', () => {
            if (!this.enemyId || this.isPlayingAnim) return;
            App.sendCommand(`COMBAT ATTACK ${this.enemyId}`);
        });

        // 2. Defend Button -> sends COMBAT DEFEND (implemented on backend!)
        const defendBtn = document.getElementById('btn-combat-defend');
        if (defendBtn) {
            defendBtn.addEventListener('click', () => {
                if (this.isPlayingAnim) return;
                App.sendCommand('COMBAT DEFEND');
            });
        }

        // 3. Heal Button -> sends COMBAT USE <consumable_item>
        const healBtn = document.getElementById('btn-combat-heal');
        if (healBtn) {
            healBtn.addEventListener('click', () => {
                if (this.isPlayingAnim) return;
                const healOrder = ['item.healing_potion', 'item.wine', 'item.bread'];
                const itemToUse = healOrder.find(id => GameState.inventory.includes(id));
                if (!itemToUse) {
                    UI.addChatLine('ROOM', '🧪 No Consumables', 'You have no Healing Elixir, Wine, or Bread in your bag!', 'chat-err');
                    return;
                }
                App.sendCommand(`COMBAT USE ${itemToUse}`);
            });
        }

        // 4. Help Button -> sends COMBAT HELP (calls other players in room to join combat)
        const helpBtn = document.getElementById('btn-combat-help');
        if (helpBtn) {
            helpBtn.addEventListener('click', () => {
                if (this.isPlayingAnim) return;
                App.sendCommand('COMBAT HELP');
                UI.addChatLine('ROOM', '📯 Call for Help', 'You called nearby allies in the room to join the battle!', 'chat-sys');
            });
        }

        // 5. Flee Button -> opens Flee confirmation modal
        const fleeModal = document.getElementById('modal-flee');
        document.getElementById('btn-combat-flee').addEventListener('click', () => {
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
            App.sendCommand('COMBAT FLEE');
        });
    },

    // Sequential animation queue processor
    enqueueAnim(fn, durationMs = 750) {
        this.animQueue.push({ fn, durationMs });
        if (!this.isPlayingAnim) {
            this.processNextAnim();
        }
    },

    processNextAnim() {
        if (this.animQueue.length === 0) {
            this.isPlayingAnim = false;
            this.setButtonsDisabled(false);
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

    setButtonsDisabled(disabled) {
        ['btn-combat-strike', 'btn-combat-defend', 'btn-combat-heal', 'btn-combat-help'].forEach(id => {
            const btn = document.getElementById(id);
            if (btn) btn.disabled = disabled;
        });
    },

    open(npcIdentifier, initialHeroHp, initialEnemyHp) {
        this.startTurnTimer(1);
        UI.closeAllModals();
        UI.activeModalId = 'combat';
        GameState.inCombat = true;
        this.animQueue = [];
        this.isPlayingAnim = false;

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
        this.setButtonsDisabled(false);
        document.getElementById('combat-turn-text').textContent = 'YOUR TURN';

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

    // Queues a defend visual effect before the enemy counter-attacks!
    playDefendEffect(reductionPct) {
        this.enqueueAnim(() => {
            const turnText = document.getElementById('combat-turn-text');
            if (turnText) turnText.textContent = 'DEFENDING 🛡️';
            this.showFloatingDmg('dmg-on-hero', `🛡️ -${reductionPct}% DMG`, '#63b3ed');
        }, 650);
    },

    playHealEffect(healAmount) {
        this.enqueueAnim(() => {
            GameState.hp = Math.min(GameState.maxHp, GameState.hp + healAmount);
            this.showFloatingDmg('dmg-on-hero', `+${healAmount} HP`, '#48bb78');
            this.updateBars();
        }, 600);
    },

    // Queues player strike or enemy counter-attack with a clean delay between them!
    playHitEffect(isPlayerAttacking, damageAmount, remainingHp) {
        this.enqueueAnim(() => {
            const heroBox = document.getElementById('fighter-hero-box');
            const enemyBox = document.getElementById('fighter-enemy-box');
            const turnText = document.getElementById('combat-turn-text');

            if (isPlayerAttacking) {
                if (remainingHp !== undefined) this.enemyHp = remainingHp;
                if (turnText) turnText.textContent = 'YOUR STRIKE ⚔️';
                this.triggerAnim(heroBox, 'lunge-right');
                this.triggerAnim(enemyBox, 'hit-shake');
                this.showFloatingDmg('dmg-on-enemy', `-${damageAmount}`, '#ff4d4d');
            } else {
                if (remainingHp !== undefined) GameState.hp = remainingHp;
                if (turnText) turnText.textContent = `${this.enemyName.toUpperCase()} ATTACKS!`;
                this.triggerAnim(enemyBox, 'lunge-left');
                this.triggerAnim(heroBox, 'hit-shake');
                this.showFloatingDmg('dmg-on-hero', `-${damageAmount}`, '#ff4d4d');
            }

            this.updateBars();
        }, 800);
    },

    queueTurnBanner(text, roundNum) {
        this.enqueueAnim(() => {
            const turnEl = document.getElementById('combat-turn-text');
            if (turnEl) turnEl.textContent = text;
            this.startTurnTimer(roundNum);
        }, 150);
    },

    triggerAnim(el, className) {
        if (!el) return;
        el.classList.remove(className);
        void el.offsetWidth;
        el.classList.add(className);
        setTimeout(() => el.classList.remove(className), 500);
    },

    showFloatingDmg(elId, text, color = '#ff4d4d') {
        const el = document.getElementById(elId);
        if (!el) return;
        el.textContent = text;
        el.style.color = color;
        el.classList.remove('hidden');
        void el.offsetWidth;
        setTimeout(() => el.classList.add('hidden'), 750);
    },

    showOutcome(victory, subtitle) {
        this.enqueueAnim(() => {
            this.setButtonsDisabled(true);
            const banner = document.getElementById('combat-outcome-banner');
            document.getElementById('combat-outcome-title').textContent = victory ? '🏆 VICTORY!' : '💀 DEFEATED';
            document.getElementById('combat-outcome-sub').textContent = subtitle;
            banner.classList.remove('hidden');

            if (victory) {
                document.getElementById('fighter-enemy-box').style.opacity = '0.25';
            } else {
                // Immediately after defeat, we synchronize HP with the server (respawn with 25% HP).
                App.sendCommand('STATUS');
                App.sendCommand('LOOK');
            }

            setTimeout(() => {
                this.close();
                App.sendCommand('STATUS');
            }, 1800);
        }, 1800);
    },

    close() {
        sessionStorage.removeItem('tap_active_combat_enemy');
        sessionStorage.removeItem('tap_active_combat_enemy_hp');

        if (this.timerInterval) clearInterval(this.timerInterval);
        
        GameState.inCombat = false;
        this.animQueue = [];
        this.isPlayingAnim = false;
        document.getElementById('modal-flee').classList.add('hidden');
        document.getElementById('modal-combat').classList.add('hidden');
        if (UI.activeModalId === 'combat') UI.activeModalId = null;
    }
};