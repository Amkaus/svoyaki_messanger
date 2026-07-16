const API_URL = 'http://localhost:8080';
let socket = null;

window.onload = () => {
    const token = localStorage.getItem('token');
    if (token) {
        showChatScreen();
    }
};

async function register() {
    const usernameInput = document.getElementById('username').value;
    const passwordInput = document.getElementById('password').value;
    const errorEl = document.getElementById('auth-error');
    try {
        const response = await fetch(`${API_URL}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username: usernameInput, password: passwordInput })
        });

        if (response.ok) {
            errorEl.style.color = 'green';
            errorEl.textContent = 'Регистрация успешна';
        } else {
            
            errorEl.style.color = 'red';
            errorEl.innerText = 'Ошибка регистрации';
        }
    } catch (err) {
        errorEl.innerText = 'Ошибка соединения с сервером';
    
    }
}

async function login() {
    const usernameInput = document.getElementById('username').value;
    const passwordInput = document.getElementById('password').value;
    const errorEl = document.getElementById('auth-error');
    try {
        const response = await fetch(`${API_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username: usernameInput, password: passwordInput })
        });

        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('token', data.token);
            showChatScreen();
        } else {
            errorEl.style.color = 'red';
            errorEl.innerText = 'Ошибка входа';
        }
    } catch (err) {
        errorEl.innerText = 'Ошибка соединения с сервером';
    }
}

function logout() {
    localStorage.removeItem('token');
    document.getElementById('chat-screen').classList.add('hidden');
    document.getElementById('auth-screen').classList.remove('hidden');
}
function showChatScreen() {
    document.getElementById('auth-screen').classList.add('hidden');
    document.getElementById('chat-screen').classList.remove('hidden');
    loadChats();
    connectWebSocket();
}

async function loadChats() {
    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/chats`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.ok) {
            const chats = await response.json();
            renderChats(chats);
        } else if (response.status === 401) {
            logout();
        }
    } catch (err) {
        console.error('Ошибка загрузки чатов', err);
    }
}



let currentChatId = null;

function renderChats(chats) {
    const chatList = document.getElementById('chat-list');
    chatList.innerHTML = '';
    
    chats.forEach(chat => {
        const li = document.createElement('li');
        li.className = 'chat-item';
        li.innerText = chat.name;
        
        li.onclick = () => selectChat(chat.id, chat.name, chat.is_admin, chat.is_group);
        
        chatList.appendChild(li);
    });
}

function selectChat(chatId, chatName, isAdmin, isGroup) {
    currentChatId = chatId;
    
    document.getElementById('current-chat-name').innerText = chatName;  
    document.getElementById('message-text').disabled = false;
    document.getElementById('send-btn').disabled = false;
    document.getElementById('messages-container').innerHTML = '<p style="text-align:center; color:#888;">Чат открыт. История сообщений скоро появится</p>';
    document.getElementById('search-container').classList.remove('hidden');
    
    if (isAdmin && isGroup) {
        document.getElementById('add-user-btn').classList.remove('hidden');
        document.getElementById('remove-user-btn').classList.remove('hidden');
    } else {
        document.getElementById('add-user-btn').classList.add('hidden');
        document.getElementById('remove-user-btn').classList.add('hidden');
    }

    loadChatHistory(chatId).then(() => {
        const token = localStorage.getItem('token');
        fetch(`${API_URL}/messages/read-all?chat_id=${chatId}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        })
        .then(() => {
            document.querySelectorAll('[id^="meta-"]').forEach(el => {
                if (!el.innerText.includes("✓✓")) {
                el.innerText = el.innerText.replace("✓", "✓✓");
                }
            });
        });
    });
}

async function createNewChat() {
    const isGroup = confirm("Хотите создать групповой чат?\n\n(ОК - Группа, Отмена - Личный чат 1-на-1)");
    
    let chatName = "";
    let targetUser = ""; 
    if (isGroup) {
        chatName = prompt("Введите название нового чата:");
        if (!chatName) return; 
    } else {
        targetUser = prompt("Введите логин пользователя, с которым хотите начать переписку:");
        if (!targetUser) return;
        chatName = targetUser; 
    }

    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/chats/create`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ name: chatName, is_group: isGroup })
        });

        if (response.ok) {
            if (!isGroup) {
                const data = await response.json();
                currentChatId = data.chat_id; 
                await addUserToChat(targetUser); 
            }
            loadChats();
        } else {
            alert("Ошибка при создании чата");
        }
    } catch (err) {
        console.error('Ошибка:', err);
    }
}

function connectWebSocket() {
    const token = localStorage.getItem('token');
    if (!token) return;

    if (socket) {
        socket.close();
    }

    socket = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

    socket.onopen = () => {
        console.log("WebSocket подключен!");
    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Входящее событие WS:", data); 

        if (data.type === "read_receipt" || data.type === "read") {
            updateCheckmarks(data.message_id);
            return;
        }

        let msg = null;
        if (data.type === "message" && data.message) {
            msg = data.message;
        } else if (data.chat_id) {
            msg = data;
        }

        if (msg) {
            if (String(msg.chat_id) === String(currentChatId)) {
                
                const myUserId = getMyUserId();
                
                if (String(msg.sender_id) !== String(myUserId)) {
                    displayMessage(msg);
                    markAsRead(msg.id);
                }
            } else {
                console.log("Сообщение пришло для другого чата:", msg.chat_id, "А мы сидим в:", currentChatId);
            }
        }
    };

    socket.onclose = () => {
        console.log("WebSocket отключен");
    };
}

function sendMessage() {
    const input = document.getElementById('message-text');
    const text = input.value.trim();

    if (!text || !currentChatId) return;

    const msgId = crypto.randomUUID(); 

    const messageData = {
        type: "send",
        chat_id: currentChatId,
        text: text,
        client_msg_id: msgId
    };

    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(messageData));
        input.value = ''; 
        
        const localMsg = {
            id: msgId, 
            chat_id: currentChatId,
            sender_id: getMyUserId(),
            content: text,
            created_at: new Date().toISOString(), 
            is_read: false
        };
        displayMessage(localMsg);
        
    } else {
        alert("Нет подключения к серверу. Проверьте консоль.");
    }
}

function displayMessage(msg) {
    const container = document.getElementById('messages-container');
    
    if (container.innerHTML.includes('История сообщений') || container.innerHTML.includes('Здесь пока нет')) {
        container.innerHTML = '';
    }

    const myUserId = getMyUserId();
    const isMyMsg = msg.sender_id === myUserId;

    const msgWrapper = document.createElement('div');
    msgWrapper.style.display = "flex";
    msgWrapper.style.width = "100%";
    msgWrapper.style.marginBottom = "10px";
    
    msgWrapper.style.justifyContent = isMyMsg ? "flex-end" : "flex-start";

    const msgBubble = document.createElement('div');
    msgBubble.style.padding = "10px 15px";
    msgBubble.style.borderRadius = "15px";
    msgBubble.style.maxWidth = "70%"; 
    msgBubble.style.boxShadow = "0 1px 2px rgba(0,0,0,0.1)";
    msgBubble.style.wordBreak = "break-word";
    
    if (isMyMsg) {
        msgBubble.style.background = "#dcf8c6"; 
        msgBubble.style.borderBottomRightRadius = "2px";
    } else {
        msgBubble.style.background = "white"; 
        msgBubble.style.borderBottomLeftRadius = "2px";
    }

    const textSpan = document.createElement('span');
    textSpan.innerText = msg.content || msg.text || "Пустое сообщение"; 
    msgBubble.appendChild(textSpan);

    const metaSpan = document.createElement('span');
    if (msg.id) metaSpan.id = `meta-${msg.id}`;
    metaSpan.style.fontSize = "10px";
    metaSpan.style.color = isMyMsg ? "#4fc3f7" : "#999"; 
    metaSpan.style.marginLeft = "12px";
    metaSpan.style.float = "right"; 
    metaSpan.style.marginTop = "8px";

    let timeString = "";
    if (msg.created_at) {
        const dateObj = new Date(msg.created_at);
        timeString = dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    let checkmarks = "";
    if (isMyMsg) {
        checkmarks = msg.is_read ? " ✓✓" : " ✓";
    }

    metaSpan.innerText = `${timeString}${checkmarks}`;
    msgBubble.appendChild(metaSpan);
    
    msgWrapper.appendChild(msgBubble);
    container.appendChild(msgWrapper);
    
  
    container.scrollTop = container.scrollHeight;
}

async function loadChatHistory(chatId) {
    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/messages/history?chat_id=${chatId}`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.ok) {
            const messages = await response.json();
            const container = document.getElementById('messages-container');
            container.innerHTML = ''; 

            if (!messages || messages.length === 0) {
                container.innerHTML = '<p style="text-align:center; color:#888;">Здесь пока нет сообщений</p>';
                return;
            }

            messages.reverse().forEach(msg => displayMessage(msg));
        } else {
            console.error("Не удалось загрузить историю");
        }
    } catch (err) {
        console.error("Ошибка сети при загрузке истории:", err);
    }
}

async function addUserToChat(prefilledUsername = null) {
    const username = prefilledUsername || prompt("Введите логин пользователя, которого хотите добавить:");
    if (!username) return;

    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/chats/add_member`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ chat_id: currentChatId, username: username.trim() }) 
        });

        if (response.ok) {
            if (!prefilledUsername) {
                alert(`Пользователь ${username} добавлен в чат!`);
            }
        } else {
            alert("Не удалось добавить пользователя. Проверь консоль.");
            console.error("Ошибка добавления:", await response.text());
        }
    } catch (err) {
        console.error('Ошибка сети:', err);
    }
}

async function removeUserFromChat() {
    const username = prompt("Введите логин пользователя, которого хотите удалить:");
    if (!username) return;

    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/chats/remove_member`, {
            method: 'POST', 
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ chat_id: currentChatId, username: username.trim() }) 
        });

        if (response.ok) {
            alert(`Пользователь ${username} успешно удален из чата!`);
        } else {
            const errText = await response.text();
            alert(`Ошибка: ${errText}`);
        }
    } catch (err) {
        console.error('Ошибка сети:', err);
    }
}

function getMyUserId() {
    const token = localStorage.getItem('token');
    if (!token) return null;
    
    try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        return payload.user_id; 
    } catch (e) {
        console.error("Ошибка расшифровки токена", e);
        return null;
    }
}

async function searchMessages() {
    const query = document.getElementById('search-input').value.trim();
    if (!query || !currentChatId) return;

    const token = localStorage.getItem('token');
    try {
        const response = await fetch(`${API_URL}/messages/search?chat_id=${currentChatId}&text=${encodeURIComponent(query)}`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.ok) {
            const messages = await response.json();
            const container = document.getElementById('messages-container');
            container.innerHTML = '';
            
            if (!messages || messages.length === 0) {
                container.innerHTML = '<p style="text-align:center; color:#888;">Ничего не найдено</p>';
                return;
            }
            
            messages.reverse().forEach(msg => displayMessage(msg));
        } else {
            alert("Ошибка при поиске. Возможно, параметры запроса не совпадают.");
        }
    } catch (err) {
        console.error("Ошибка поиска:", err);
    }
}

function clearSearch() {
    document.getElementById('search-input').value = '';
    loadChatHistory(currentChatId); 
}
function markAsRead(msgId) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
            type: "read",
            message_id: msgId
        }));
    }
}

function updateCheckmarks(msgId) {
    const metaSpan = document.getElementById(`meta-${msgId}`);
    
    if (!metaSpan) {
        return; 
    }

    if (!metaSpan.innerText.includes("✓✓")) {
        metaSpan.innerText = metaSpan.innerText.replace("✓", "✓✓");
    }
}