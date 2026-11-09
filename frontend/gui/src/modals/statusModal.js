// ==========================================================================
// MODAL MODULE: CHARACTER STATUS (Real Combat Stats + Equipment Bonuses!)
// ==========================================================================

const StatusModal = {
    gearBonuses: {
        "item.iron_sword":        { atk: 6, def: 0, spd: 0 },
        "item.oak_shield":        { atk: 0, def: 4, spd: 0 },
        "item.leather_boots":     { atk: 0, def: 2, spd: 5 },
        "item.iron_gauntlets":    { atk: 3, def: 3, spd: 0 },
        "item.steel_breastplate": { atk: 0, def: 6, spd: 0 },
        "item.royal_helmet":      { atk: 2, def: 5, spd: 0 }
    },

    render() {
        const pct = Math.max(0, Math.min(100, Math.round((GameState.hp / GameState.maxHp) * 100)));
        document.getElementById('st-hp-fill').style.width = pct + '%';
        document.getElementById('st-hp-val').textContent = `${GameState.hp} / ${GameState.maxHp}`;
        document.getElementById('st-cond-val').textContent = GameState.status;

        // Calculate real equipment bonuses from current inventory
        let bonusAtk = 0, bonusDef = 0, bonusSpd = 0, gearCount = 0;
        (GameState.inventory || []).forEach(itemId => {
            const b = this.gearBonuses[itemId];
            if (b) {
                bonusAtk += b.atk;
                bonusDef += b.def;
                bonusSpd += b.spd;
                gearCount++;
            }
        });

        const baseAtk = 10, baseDef = 5, baseSpd = 10;
        document.getElementById('st-atk-val').textContent = bonusAtk > 0 ? `${baseAtk + bonusAtk} (${baseAtk} + ${bonusAtk})` : `${baseAtk}`;
        document.getElementById('st-def-val').textContent = bonusDef > 0 ? `${baseDef + bonusDef} (${baseDef} + ${bonusDef})` : `${baseDef}`;
        document.getElementById('st-spd-val').textContent = bonusSpd > 0 ? `${baseSpd + bonusSpd} (${baseSpd} + ${bonusSpd})` : `${baseSpd}`;

        // Miscellaneous world info
        document.getElementById('st-loc-val').textContent = GameState.room.name;
        document.getElementById('st-gear-val').textContent = `${gearCount} / 6 equipped`;

        let totalItems = 0;
        (GameState.inventory || []).forEach(id => {
            totalItems += (GameState.inventoryCounts && GameState.inventoryCounts[id]) || 1;
        });
        document.getElementById('st-items-val').textContent = `${totalItems} in bag`;

        const completedQuests = (GameState.quests || []).filter(q => q.status === 'completed').length;
        document.getElementById('st-quests-val').textContent = `${completedQuests} completed`;

        document.getElementById('st-party-val').textContent = GameState.currentGroup
            ? `${GameState.currentGroup.name} (${GameState.currentGroup.members.length})`
            : 'Solo';
        document.getElementById('st-pop-val').textContent = `${GameState.serverPlayerCount} online`;
    }
};