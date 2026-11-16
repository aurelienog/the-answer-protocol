// ==========================================================================
// MODULE: COMBAT SPRITE ANIMATOR (With Automatic Action Timing & Sync!)
// ==========================================================================

const CombatAnimator = {

    // ======================================================================
    // 🔧 MANUAL TUNING PANEL — Edit frames, FPS, scale & timing here!
    // ======================================================================
    CONFIG: {
        cropCornerPercent: 0,

        // Default step durations (in ms) when an actor uses static image (no frames)
        defaultStepMs: 1350,
        defaultImpactDelayMs: 250,

        enemies: {
            "dire_wolf": {
                enabled: true,
                basePath: "/assets/combat/anims/dire_wolf",
                scale: 1.0,
                offsetY: 10,
                states: {
                    idle: {
                        frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18],
                        fps: 10,
                        loop: true
                    },
                    attack: {
                        frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18],
                        fps: 10,                        loop: false,
                        impactDelayMs: 650,
                        extraHoldMs: 200
                    },
                    hit: {
                        frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18],
                        fps: 10,
                        loop: false
                    },
                    death: {
                        frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18],
                        fps: 10,
                        loop: false
                    }
                }
            },
            "smuggler": {
                enabled: true,
                basePath: "/assets/combat/anims/smuggler",
                scale: 1.2,
                offsetY: 0,
                states: {
                    idle:   { frames: [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17], fps: 10,  loop: true },
                    attack: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, impactDelayMs: 400, extraHoldMs: 200 },
                    hit:    { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false },
                    death:  { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 8,  loop: false }
                }
            },
            "dock_warden": {
                enabled: true,
                basePath: "/assets/combat/anims/dock_warden",
                scale: 1.2,
                offsetY: 0,
                states: {
                    idle:   { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10,  loop: true },
                    attack: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, impactDelayMs: 400, extraHoldMs: 200 },
                    hit:    { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false },
                    death:  { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 8,  loop: false }
                }
            }
        },

        heroes: {
            "male": {
                enabled: true,
                basePath: "/assets/combat/anims/hero_male",
                scale: 1.2,
                offsetY: 0,
                states: {
                    idle:   { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10,  loop: true },
                    attack: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, impactDelayMs: 400, extraHoldMs: 200 },
                    defend: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, extraHoldMs: 200 },
                    heal:   { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, extraHoldMs: 200 },
                    hit:    { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17], fps: 10, loop: false },
                    death:  { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 8,  loop: false }
                }
            },
            "female": {
                enabled: true,
                basePath: "/assets/combat/anims/hero_female",
                scale: 1.2,
                offsetY: 0,
                states: {
                    idle:   { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10,  loop: true },
                    attack: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, impactDelayMs: 400, extraHoldMs: 200 },
                    defend: { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, extraHoldMs: 200 },
                    heal:   { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false, extraHoldMs: 200 },
                    hit:    { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 10, loop: false },
                    death:  { frames: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18], fps: 8,  loop: false }
                }
            }
        }
    },

    imageCache: {},
    slots: {
        hero:  { timer: null, actorKey: null, state: 'idle', seqIndex: 0, sequence: [] },
        enemy: { timer: null, actorKey: null, state: 'idle', seqIndex: 0, sequence: [] }
    },

    preloadActor(actorCfg) {
        if (!actorCfg || !actorCfg.enabled || !actorCfg.states) return;
        Object.keys(actorCfg.states).forEach(stateName => {
            const st = actorCfg.states[stateName];
            if (!st || !Array.isArray(st.frames)) return;
            st.frames.forEach(num => {
                const url = `${actorCfg.basePath}/${stateName}/${num}.webp`;
                if (!this.imageCache[url]) {
                    const img = new Image();
                    img.src = url;
                    this.imageCache[url] = img;
                }
            });
        });
    },

    startCombat(heroGender, enemyId) {
        this.stopAll();
        this.applySpriteTransforms(heroGender, enemyId);

        const heroCfg = this.CONFIG.heroes[heroGender];
        if (heroCfg && heroCfg.enabled) {
            this.preloadActor(heroCfg);
            this.playState('hero', heroGender, 'idle');
        }

        const enemyCfg = this.CONFIG.enemies[enemyId];
        if (enemyCfg && enemyCfg.enabled) {
            this.preloadActor(enemyCfg);
            this.playState('enemy', enemyId, 'idle');
        }
    },

    // Calculates exact total duration (ms) & impact delay (ms) for an action state
    // so CombatModal's animation queue waits until all frames finish playing!
    getStepTiming(slotName, stateName) {
        const key = slotName === 'hero' ? GameState.gender : CombatModal.enemyId;
        const group = slotName === 'hero' ? this.CONFIG.heroes : this.CONFIG.enemies;
        const actorCfg = group[key];

        if (!actorCfg || !actorCfg.enabled || !actorCfg.states || !actorCfg.states[stateName]) {
            return { totalMs: this.CONFIG.defaultStepMs, impactDelayMs: 0, hasFrames: false };
        }

        const st = actorCfg.states[stateName];
        if (!Array.isArray(st.frames) || st.frames.length === 0) {
            return { totalMs: this.CONFIG.defaultStepMs, impactDelayMs: 0, hasFrames: false };
        }

        const fps = st.fps || 8;
        const animDurationMs = Math.round((st.frames.length / fps) * 1000);
        const extraHold = st.extraHoldMs !== undefined ? st.extraHoldMs : 200;
        const totalMs = Math.max(this.CONFIG.defaultStepMs, animDurationMs + extraHold);
        const impactDelayMs = st.impactDelayMs !== undefined
            ? st.impactDelayMs
            : Math.round(animDurationMs * 0.4);

        return { totalMs, impactDelayMs, hasFrames: true };
    },

    playState(slotName, actorKey, stateName = 'idle') {
        const group = slotName === 'hero' ? this.CONFIG.heroes : this.CONFIG.enemies;
        const actorCfg = group[actorKey];
        if (!actorCfg || !actorCfg.enabled) return false;

        const st = actorCfg.states[stateName];
        if (!st || !Array.isArray(st.frames) || st.frames.length === 0) {
            if (stateName !== 'idle') {
                return this.playState(slotName, actorKey, 'idle');
            }
            return false;
        }

        this.stopSlot(slotName);

        const imgEl = document.getElementById(
            slotName === 'hero' ? 'combat-hero-sprite' : 'combat-enemy-sprite'
        );
        if (!imgEl) return false;

        const sequence = st.frames.map(n => `${actorCfg.basePath}/${stateName}/${n}.webp`);

        const slot = this.slots[slotName];
        slot.actorKey = actorKey;
        slot.state = stateName;
        slot.sequence = sequence;
        slot.seqIndex = 0;

        imgEl.src = sequence[0];

        if (sequence.length === 1) return true;

        const intervalMs = Math.max(30, Math.round(1000 / (st.fps || 8)));
        slot.timer = setInterval(() => {
            if (!GameState.inCombat) {
                this.stopAll();
                return;
            }

            slot.seqIndex++;

            if (slot.seqIndex >= slot.sequence.length) {
                if (st.loop) {
                    slot.seqIndex = 0;
                } else {
                    this.stopSlot(slotName);
                    if (stateName !== 'death') {
                        this.playState(slotName, actorKey, 'idle');
                    }
                    return;
                }
            }

            imgEl.src = slot.sequence[slot.seqIndex];
        }, intervalMs);

        return true;
    },

    triggerAction(slotName, stateName) {
        const slot = this.slots[slotName];
        const key = (slot && slot.actorKey) || (slotName === 'hero' ? GameState.gender : CombatModal.enemyId);
        if (!key) return false;
        return this.playState(slotName, key, stateName);
    },

    applySpriteTransforms(heroGender, enemyId) {
        const pct = this.CONFIG.cropCornerPercent || 0;
        const clip = pct > 0 ? `inset(${pct}% ${pct}% ${pct}% ${pct}%)` : '';

        const heroEl = document.getElementById('combat-hero-sprite');
        const heroCfg = this.CONFIG.heroes[heroGender];
        if (heroEl) {
            const s = (heroCfg && heroCfg.enabled && heroCfg.scale) ? heroCfg.scale : 1;
            const y = (heroCfg && heroCfg.enabled && heroCfg.offsetY) ? heroCfg.offsetY : 0;
            heroEl.style.clipPath = (heroCfg && heroCfg.enabled) ? clip : '';
            heroEl.style.transform = `translateY(${y}px) scale(${s})`;
            heroEl.style.transformOrigin = 'bottom center';
        }

        const enemyEl = document.getElementById('combat-enemy-sprite');
        const enemyCfg = this.CONFIG.enemies[enemyId];
        if (enemyEl) {
            const s = (enemyCfg && enemyCfg.enabled && enemyCfg.scale) ? enemyCfg.scale : 1;
            const y = (enemyCfg && enemyCfg.enabled && enemyCfg.offsetY) ? enemyCfg.offsetY : 0;
            enemyEl.style.clipPath = (enemyCfg && enemyCfg.enabled) ? clip : '';
            enemyEl.style.transform = `translateY(${y}px) scale(${s})`;
            enemyEl.style.transformOrigin = 'bottom center';
        }
    },

    stopSlot(slotName) {
        const slot = this.slots[slotName];
        if (slot && slot.timer) {
            clearInterval(slot.timer);
            slot.timer = null;
        }
    },

    stopAll() {
        this.stopSlot('hero');
        this.stopSlot('enemy');
    }
};