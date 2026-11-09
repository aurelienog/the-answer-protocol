// ==========================================================================
// HUD MODULE: TOP BAR (Hero HP Bar & Server Population Counter)
// ==========================================================================

const TopHudView = {
    updateHUDStats() {
        const pct = Math.max(0, Math.min(100, Math.round((GameState.hp / GameState.maxHp) * 100)));
        document.getElementById('hud-hp-fill').style.width = pct + '%';
        document.getElementById('hud-hp-text').textContent = `${GameState.hp} / ${GameState.maxHp}`;
        document.getElementById('hud-server-count').textContent = GameState.serverPlayerCount;

        // Also keep Status Modal fields synced if open
        if (typeof StatusModal !== 'undefined') {
            StatusModal.render();
        }
    }
};