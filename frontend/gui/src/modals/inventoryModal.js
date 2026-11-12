// ==========================================================================
// MODAL MODULE: INVENTORY (4x4 Slots Grid with .webp Icons & Card Preview!)
// ==========================================================================

const InventoryModal = {
    render() {
        const grid = document.getElementById('inventory-slots');
        grid.innerHTML = '';

        if (!GameState.selectedInventoryItem && GameState.inventory.length > 0) {
            GameState.selectedInventoryItem = GameState.inventory[0];
        }

        if (!GameState.selectedInventoryItem || !GameState.inventory.includes(GameState.selectedInventoryItem)) {
            this.clearInspector();
        } else {
            this.updateInspector(GameState.selectedInventoryItem);
        }

        for (let i = 0; i < 16; i++) {
            const itemId = GameState.inventory[i];
            const slot = document.createElement('li');
            slot.className = 'inv-slot' + (itemId ? ' filled' : '');

            if (itemId) {
                const meta = GameState.getItemMeta(itemId);
                const qty = (GameState.inventoryCounts && GameState.inventoryCounts[itemId]) || 1;
                if (GameState.selectedInventoryItem === itemId) slot.classList.add('selected');

                const iconUrl = meta.iconImg || meta.img;
                slot.title = meta.name;
                slot.innerHTML = `
                    ${iconUrl
                        ? `<img src="${iconUrl}" alt="${meta.name}" class="inv-slot-img" onerror="this.style.display='none';this.nextElementSibling.style.display='block';">
                           <span style="display:none;">${meta.icon}</span>`
                        : `<span>${meta.icon}</span>`
                    }
                    ${qty > 1 ? `<span class="inv-slot-qty">${qty}</span>` : ''}
                `;
                slot.addEventListener('click', () => this.selectItem(itemId));
            } else {
                slot.innerHTML = `<span style="color:rgba(212,163,70,0.15);font-size:14px;">✦</span>`;
            }
            grid.appendChild(slot);
        }
    },

    clearInspector() {
        GameState.selectedInventoryItem = null;
        UI.setArt(document.getElementById('inv-item-icon'), '/assets/ui/card/look-items.jpeg', '🗡️');
        document.getElementById('inv-item-name').textContent = 'Select an Item';
        document.getElementById('inv-stat-type').textContent = '-';
        document.getElementById('inv-stat-weight').textContent = '-';
        document.getElementById('inv-stat-value').textContent = '-';
        document.getElementById('inv-item-desc').textContent = 'Click on any item in your bag to inspect, use, or drop it in the current room.';
        document.getElementById('btn-drop-selected').disabled = true;
        document.getElementById('btn-use-selected').disabled = true;
    },

    updateInspector(itemId) {
        const meta = GameState.getItemMeta(itemId);
        const itemType = (meta.type || 'common').toLowerCase();
        const qty = (GameState.inventoryCounts && GameState.inventoryCounts[itemId]) || 1;

        // On the right, in the description window, we display a large illustration from /assets/items/card/*.webp!
        UI.setArt(document.getElementById('inv-item-icon'), meta.img || meta.iconImg, meta.icon, meta.name);
        document.getElementById('inv-item-name').textContent = qty > 1 ? `${meta.name} (×${qty})` : meta.name;
        document.getElementById('inv-stat-type').textContent = itemType.toUpperCase();
        document.getElementById('inv-stat-weight').textContent = meta.stats || (itemType === 'common' ? 'Droppable' : 'Bound to Player');
        document.getElementById('inv-stat-value').textContent = itemId;
        document.getElementById('inv-item-desc').textContent = meta.desc;

        const canDrop = (itemType === 'common' || itemType === 'consumable');
        const dropBtn = document.getElementById('btn-drop-selected');
        dropBtn.disabled = !canDrop;
        dropBtn.title = canDrop ? 'Drop item in current room' : 'Equipment, Unique and Quest items cannot be dropped';

        const useBtn = document.getElementById('btn-use-selected');
        useBtn.disabled = (itemType !== 'consumable');
        useBtn.title = itemType === 'consumable' ? 'Consume item to restore HP' : 'Passive or Quest item';
    },

    selectItem(itemId) {
        GameState.selectedInventoryItem = itemId;
        this.render();
    },

    useSelected() {
        if (!GameState.selectedInventoryItem) return;
        const item = GameState.selectedInventoryItem;
        App.sendCommand(`USE ${item}`);
    },

    dropSelected() {
        if (!GameState.selectedInventoryItem) return;
        const item = GameState.selectedInventoryItem;
        App.sendCommand(`DROP ${item}`);
    }
};