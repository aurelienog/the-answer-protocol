// ==========================================================================
// HUD MODULE: BOTTOM-LEFT CHAT & SYSTEM LOG PANEL
// ==========================================================================

const ChatView = {
    switchChatScope(scope) {
        GameState.activeChatScope = scope;
        document.querySelectorAll('.chat-panel .chat-tab').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.chatScope === scope);
        });
        document.getElementById('chat-scope-badge').textContent = `[${scope}]`;
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
            p.innerHTML = `<time class="chat-time">[${m.time}]</time> <strong class="chat-author">${m.author}:</strong> <span>${m.text}</span>`;
            box.appendChild(p);
        });
        box.scrollTop = box.scrollHeight;
    }
};