// ==========================================================================
// HUD MODULE: BOTTOM-LEFT CHAT & SYSTEM LOG PANEL
// ==========================================================================

const ChatView = {
    switchChatScope(scope) {
        GameState.activeChatScope = scope;
        document.querySelectorAll('.chat-panel .chat-tab').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.chatScope === scope);
        });
        const badgeScope = scope === 'COMBAT' ? 'ROOM' : scope;
        document.getElementById('chat-scope-badge').textContent = `[${badgeScope}]`;
        this.renderChatMessages();
    },

    addChatLine(scope, author, text, styleClass = '') {
        const now = new Date();
        const time = now.toTimeString().slice(0, 5);
        const entry = { time, author, text, styleClass };

        if (GameState.chats[scope]) {
            GameState.chats[scope].push(entry);
        }
        this.renderChatMessages();
    },

    renderChatMessages() {
        const box = document.getElementById('chat-messages');
        const list = GameState.chats[GameState.activeChatScope] || [];
        box.innerHTML = '';

        list.forEach(m => {
            const p = document.createElement('p');
            p.className = `chat-line ${m.styleClass}`;

            const timeEl = document.createElement('time');
            timeEl.className = 'chat-time';
            timeEl.textContent = `[${m.time}]`;

            const authorEl = document.createElement('strong');
            authorEl.className = 'chat-author';
            authorEl.textContent = `${m.author}:`;

            const textEl = document.createElement('span');
            textEl.textContent = m.text;

            p.append(timeEl, ' ', authorEl, ' ', textEl);
            box.appendChild(p);
        });
        box.scrollTop = box.scrollHeight;
    }
};