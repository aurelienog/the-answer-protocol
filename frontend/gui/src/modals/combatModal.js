// ==========================================================================
// MODAL MODULE: FULLSCREEN COMBAT ARENA & TURN-BY-TURN ANIMATION QUEUE
// ==========================================================================

const CombatModal = {
    enemyId: null,
    enemyName: "Enemy",
    enemyHp: 50,
    enemyMaxHp: 50,

    roundPlayers: [],
    allCombatPlayers: [],
    actedInRound: {},
    soleSurvivor: false,
    outcomeShown: false,

    animQueue: [],
    isPlayingAnim: false,
    isMyTurn: true,
    isFallen: false,
    turnEventCounter: 0,
    deathCheckTimer: null,
    activeTurnOwner: "",
    currentRound: 1,
    roundKnown: true,
    turnSeconds: 30,
    timerInterval: null,

    startTurnTimer(roundNum, resetClock = true) {
        if (roundNum !== undefined && roundNum > 0) {
            if (roundNum !== this.currentRound) {
                this.actedInRound = {};
                this.roundPlayers = this.allCombatPlayers.slice();
            }
            this.currentRound = roundNum;
            this.roundKnown = true;
            sessionStorage.setItem('tap_combat_round', String(this.currentRound));
        } else {
            const savedRound = parseInt(sessionStorage.getItem('tap_combat_round'), 10);
            if (!isNaN(savedRound) && savedRound > 0) {
                this.currentRound = savedRound;
            }
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
        if (rEl) rEl.textContent = this.roundKnown ? `Round ${this.currentRound}` : 'Joined';
        if (tEl) tEl.textContent = `${this.turnSeconds}s`;
    },

    // Sets the dedicated combat arena background (.webp) by enemyId or roomId!
    syncRoomBackground() {
        const bgMap = (ASSETS.combat && ASSETS.combat.backgrounds) || {};
        const roomMeta = ASSETS.rooms[GameState.room.id] || ASSETS.rooms.default;

        const bgUrl = bgMap[this.enemyId] || bgMap[GameState.room.id] || roomMeta.img;
        const combatBg = document.getElementById('combat-bg-img');
        if (combatBg && bgUrl) {
            combatBg.src = bgUrl;
            combatBg.alt = `Battle against ${this.enemyName}`;
        }
    },

    setInitialParticipants(playerList) {
        if (Array.isArray(playerList) && playerList.length > 0) {
            this.roundPlayers = playerList.slice();
            this.allCombatPlayers = playerList.slice();
            this.actedInRound = {};
            this.soleSurvivor = (playerList.length === 1 && playerList[0].toLowerCase() === GameState.playerName.toLowerCase());
            sessionStorage.setItem('tap_combat_participants', JSON.stringify(this.allCombatPlayers));
        }
    },

    addJoinedParticipant(playerName) {
        if (!playerName) return;
        if (!this.allCombatPlayers.some(p => p.toLowerCase() === playerName.toLowerCase())) {
            this.allCombatPlayers.push(playerName);
            sessionStorage.setItem('tap_combat_participants', JSON.stringify(this.allCombatPlayers));
        }
        if (this.allCombatPlayers.length > 1) {
            this.soleSurvivor = false;
        }
    },

    advanceWithinCurrentRound(actingPlayerName) {
        if (actingPlayerName) {
            this.actedInRound[actingPlayerName.toLowerCase()] = true;
        }

        if (this.roundPlayers.length < this.allCombatPlayers.length && this.currentRound > 1) {
            this.roundPlayers = this.allCombatPlayers.slice();
        }

        const nextInRound = this.roundPlayers.find(p => !this.actedInRound[p.toLowerCase()]);
        if (nextInRound) {
            this.activeTurnOwner = nextInRound;
            sessionStorage.setItem('tap_combat_turn_owner', nextInRound);
            sessionStorage.setItem('tap_combat_deadline', String(Date.now() + 30000));
        }
    },

    updateBannerFromOwner() {
        const turnEl = document.getElementById('combat-turn-text');
        if (!turnEl) return;

        if (this.isFallen) {
            this.isMyTurn = false;
            const owner = this.activeTurnOwner || 'ALLY';
            turnEl.innerHTML = `<img src="/assets/icons/ui/skull.webp" alt="" class="ui-ico">SPECTATING (${owner.toUpperCase()})`;
            return;
        }

        if (this.soleSurvivor) {
            this.activeTurnOwner = GameState.playerName;
            this.isMyTurn = true;
            turnEl.textContent = 'YOUR TURN';
            return;
        }

        const owner = this.activeTurnOwner || GameState.playerName;
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
                this.advanceWithinCurrentRound(GameState.playerName);
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

    enqueueAnim(fn, durationMs = 1100) {
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
        if (this.soleSurvivor && !this.isFallen) {
            this.isMyTurn = true;
        }
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

    open(npcIdentifier, initialHeroHp, initialEnemyHp, isRestoring = false, isJoiningMidCombat = false) {
        const alreadyOpen = GameState.inCombat && !document.getElementById('modal-combat').classList.contains('hidden');

        UI.closeAllModals();
        UI.activeModalId = 'combat';
        GameState.inCombat = true;
        this.outcomeShown = false;

        if (isJoiningMidCombat) {
            this.roundKnown = false;
        } else if (!alreadyOpen && !isRestoring) {
            this.roundKnown = true;
        }

        if (!alreadyOpen && !isRestoring) {
            this.actedInRound = {};
            this.soleSurvivor = false;
            if (this.allCombatPlayers.length === 0) {
                this.roundPlayers = [GameState.playerName];
                this.allCombatPlayers = [GameState.playerName];
            }
        } else if (isRestoring && this.allCombatPlayers.length === 0) {
            try {
                const savedParts = JSON.parse(sessionStorage.getItem('tap_combat_participants') || '[]');
                if (Array.isArray(savedParts) && savedParts.length > 0) {
                    this.allCombatPlayers = savedParts;
                    this.roundPlayers = savedParts.slice();
                }
            } catch (_) {}
        }

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
            this.startTurnTimer(isJoiningMidCombat ? undefined : 1, true);
        }

        this.syncRoomBackground();

        document.getElementById('combat-hero-name').textContent = GameState.playerName;
        document.getElementById('combat-enemy-name').textContent = this.enemyName;
        UI.setArt(document.getElementById('combat-enemy-avatar'), npcMeta.iconImg, npcMeta.icon);

        const heroSprite = document.getElementById('combat-hero-sprite');
        const heroFallback = document.getElementById('combat-hero-fallback');
        heroFallback.textContent = GameState.gender === 'male' ? '🧙‍♂️' : '🧝‍♀️';
        heroSprite.style.display = 'block';
        heroFallback.style.display = 'none';
        heroSprite.src = (ASSETS.combat && ASSETS.combat.heroes && ASSETS.combat.heroes[GameState.gender])
            || `/assets/combat/hero_${GameState.gender}.webp`;

        const enemySprite = document.getElementById('combat-enemy-sprite');
        const enemyFallback = document.getElementById('combat-enemy-fallback');
        enemyFallback.textContent = npcMeta.icon;
        enemySprite.style.display = 'block';
        enemyFallback.style.display = 'none';
        enemySprite.src = (ASSETS.combat && ASSETS.combat.enemies && ASSETS.combat.enemies[this.enemyId])
            || `/assets/combat/${this.enemyId}.webp`;

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
            if (turnText) turnText.innerHTML = `DEFENDING <img src="/assets/icons/ui/shield.webp" alt="" class="ui-ico">`;
            this.showFloatingDmg('dmg-on-hero', `+${reductionPct}% DEF`, '#63b3ed');
        }, 1250);
    },

    playHealEffect(healAmount) {
        this.enqueueAnim(() => {
            GameState.hp = Math.min(GameState.maxHp, GameState.hp + healAmount);
            this.showFloatingDmg('dmg-on-hero', `+${healAmount} HP`, '#48bb78');
            this.updateBars();
        }, 1250);
    },

    playHitEffect(mode, damageAmount, remainingHp, customBannerText = '', allyTargetName = '', attackerName = '') {
        if ((mode === 'enemy_hit' || mode === true) && attackerName) {
            this.advanceWithinCurrentRound(attackerName);
        }

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
        if (roundNum !== undefined && roundNum > 0) {
            if (roundNum !== this.currentRound) {
                this.actedInRound = {};
                this.roundPlayers = this.allCombatPlayers.slice();
            }
            this.roundKnown = true;
            sessionStorage.setItem('tap_combat_round', String(roundNum));
        }
        this.activeTurnOwner = cleanOwner;
        sessionStorage.setItem('tap_combat_turn_owner', cleanOwner);
        sessionStorage.setItem('tap_combat_deadline', String(Date.now() + 30000));

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
        if (fallenName) {
            this.roundPlayers = this.roundPlayers.filter(p => p.toLowerCase() !== fallenName.toLowerCase());
            this.allCombatPlayers = this.allCombatPlayers.filter(p => p.toLowerCase() !== fallenName.toLowerCase());
            sessionStorage.setItem('tap_combat_participants', JSON.stringify(this.allCombatPlayers));
        }

        if (diedIsMe) {
            this.isFallen = true;
            this.soleSurvivor = false;
            this.isMyTurn = false;
            GameState.hp = 0;
            this.updateBars();
            this.setButtonsDisabled(true);
            this.updateBannerFromOwner();
        } else if (!this.isFallen) {
            this.soleSurvivor = true;
            this.allCombatPlayers = [GameState.playerName];
            this.roundPlayers = [GameState.playerName];
            this.actedInRound = {};
            this.activeTurnOwner = GameState.playerName;
            this.isMyTurn = true;
            sessionStorage.setItem('tap_combat_turn_owner', GameState.playerName);
            sessionStorage.setItem('tap_combat_participants', JSON.stringify(this.allCombatPlayers));

            this.enqueueAnim(() => {
                this.soleSurvivor = true;
                this.activeTurnOwner = GameState.playerName;
                this.isMyTurn = true;
                this.updateBannerFromOwner();
                this.startTurnTimer(this.currentRound, false);
                this.syncActionButtonsState();
            }, 200);
        }

        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);
        this.deathCheckTimer = setTimeout(() => {
            if (GameState.inCombat && this.isFallen) {
                App.sendCommand('LOOK');
            }
        }, 450);
    },

    handleAllyFled(runnerName) {
        if (runnerName) {
            this.roundPlayers = this.roundPlayers.filter(p => p.toLowerCase() !== runnerName.toLowerCase());
            this.allCombatPlayers = this.allCombatPlayers.filter(p => p.toLowerCase() !== runnerName.toLowerCase());
            sessionStorage.setItem('tap_combat_participants', JSON.stringify(this.allCombatPlayers));
        }

        if (this.isFallen) {
            setTimeout(() => {
                if (GameState.inCombat) {
                    App.sendCommand('LOOK');
                }
            }, 350);
        } else {
            this.soleSurvivor = true;
            this.activeTurnOwner = GameState.playerName;
            this.isMyTurn = true;
            this.updateBannerFromOwner();
            this.syncActionButtonsState();
        }
    },

    showOutcome(victory, subtitle) {
        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);
        if (this.outcomeShown) return;
        this.outcomeShown = true;

        sessionStorage.removeItem('tap_active_combat_enemy');
        sessionStorage.removeItem('tap_active_combat_enemy_hp');
        sessionStorage.removeItem('tap_combat_deadline');
        sessionStorage.removeItem('tap_combat_round');
        sessionStorage.removeItem('tap_combat_turn_owner');
        sessionStorage.removeItem('tap_combat_participants');

        this.enqueueAnim(() => {
            this.isMyTurn = false;
            this.setButtonsDisabled(true);
            const banner = document.getElementById('combat-outcome-banner');
            document.getElementById('combat-outcome-title').innerHTML = victory
                ? `<img src="/assets/icons/ui/trophy.webp" alt="" class="ui-ico-lg">VICTORY!`
                : `<img src="/assets/icons/ui/skull.webp" alt="" class="ui-ico-lg">DEFEATED`;
            document.getElementById('combat-outcome-sub').textContent = subtitle;
            banner.classList.remove('hidden');

            if (victory) {
                document.getElementById('fighter-enemy-box').style.opacity = '0.25';
            }

            setTimeout(() => {
                this.close();
                App.sendCommand('STATUS');
                App.sendCommand('LOOK');
            }, 3200);
        }, 3300);
    },

    close() {
        sessionStorage.removeItem('tap_active_combat_enemy');
        sessionStorage.removeItem('tap_active_combat_enemy_hp');
        sessionStorage.removeItem('tap_combat_deadline');
        sessionStorage.removeItem('tap_combat_round');
        sessionStorage.removeItem('tap_combat_turn_owner');
        sessionStorage.removeItem('tap_combat_participants');

        if (this.timerInterval) clearInterval(this.timerInterval);
        if (this.deathCheckTimer) clearTimeout(this.deathCheckTimer);

        GameState.inCombat = false;
        this.outcomeShown = false;
        this.soleSurvivor = false;
        this.roundKnown = true;
        this.roundPlayers = [];
        this.allCombatPlayers = [];
        this.actedInRound = {};
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