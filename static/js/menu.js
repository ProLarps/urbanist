const API = '';
const SESSION_KEY = 'urbanistSession';

const views = {
    login: document.querySelector('#login-view'),
    rooms: document.querySelector('#rooms-view'),
    lobby: document.querySelector('#lobby-view')
};

const elements = {
    loginForm: document.querySelector('#login-form'),
    playerName: document.querySelector('#player-name'),
    loginButton: document.querySelector('#login-button'),
    loginMessage: document.querySelector('#login-message'),
    roomsMessage: document.querySelector('#rooms-message'),
    lobbyMessage: document.querySelector('#lobby-message'),
    roomList: document.querySelector('#room-list'),
    createRoomButton: document.querySelector('#create-room-button'),
    refreshRoomsButton: document.querySelector('#refresh-rooms-button'),
    logoutButton: document.querySelector('#logout-button'),
    inviteCode: document.querySelector('#invite-code'),
    copyCodeButton: document.querySelector('#copy-code-button'),
    participantList: document.querySelector('#participant-list'),
    participantCount: document.querySelector('#participant-count'),
    roleBadge: document.querySelector('#role-badge'),
    lobbySubtitle: document.querySelector('#lobby-subtitle'),
    backToRoomsButton: document.querySelector('#back-to-rooms-button')
};

const state = {
    name: '',
    token: '',
    playerId: '',
    roomCode: '',
    ownerId: '',
    roomPoll: null,
    socket: null,
    isBusy: false
};


class ApiError extends Error {
    constructor(message, status, payload) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
        this.payload = payload;
    }
}

function showView(name) {
    for (const [key, element] of Object.entries(views)) {
        element.hidden = key !== name;
    }
}

function setMessage(element, message = '', kind = 'error') {
    element.textContent = message;
    element.classList.toggle('success', kind === 'success');
}

function saveSession() {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify({
        name: state.name,
        token: state.token,
        playerId: state.playerId
    }));
}

function clearSession() {
    sessionStorage.removeItem(SESSION_KEY);
    state.name = '';
    state.token = '';
    state.playerId = '';
    state.roomCode = '';
    state.ownerId = '';
    stopRoomPolling();
    closeLobbySocket();
}

function restoreSession() {
    try {
        const saved = JSON.parse(sessionStorage.getItem(SESSION_KEY) || 'null');
        if (!saved?.token || !saved?.name || !saved?.playerId) return false;
        state.name = saved.name;
        state.token = saved.token;
        state.playerId = String(saved.playerId);
        return true;
    } catch {
        sessionStorage.removeItem(SESSION_KEY);
        return false;
    }
}

async function apiRequest(path, { method = 'GET', body, auth = true } = {}) {
    const headers = { 'Accept': 'application/json' };
    if (body !== undefined) headers['Content-Type'] = 'application/json';
    if (auth && state.token) headers['Authorization'] = `Bearer ${state.token}`;

    let response;
    try {
        response = await fetch(`${API}${path}`, {
            method,
            headers,
            body: body === undefined ? undefined : JSON.stringify(body)
        });
    } catch (error) {
        throw new ApiError('Не удалось связаться с сервером. Проверь, запущен ли Go backend.', 0, null);
    }

    const raw = await response.text();
    let payload = {};
    if (raw) {
        try { payload = JSON.parse(raw); }
        catch { payload = { message: raw }; }
    }

    if (!response.ok) {
        const message = payload.message || payload.error || `Ошибка сервера (${response.status})`;
        throw new ApiError(message, response.status, payload);
    }
    return payload;
}

function explainApiError(error, context = '') {
    if (error.status === 0) return error.message;
    if (error.status === 401 || error.status === 403) {
        return 'Сессия недействительна или у тебя недостаточно прав. Войди заново.';
    }
    if (error.status === 404 && context === 'room') return 'Комната больше не существует.';
    return error.message || 'Произошла ошибка. Попробуй ещё раз.';
}

async function authenticate(name, roomCode = '') {
    const body = { name };
    if (roomCode) body.room = roomCode;

    const result = await apiRequest('/api/auth', {
        method: 'POST',
        body,
        auth: false
    });

    if (!result.token || !result.playerId) {
        throw new ApiError('Сервер не вернул token или playerId в ответе /api/auth.', 500, result);
    }

    state.name = result.name || name;
    state.token = result.token;
    state.playerId = String(result.playerId);
    saveSession();
    return result;
}

async function handleLogin(event) {
    event.preventDefault();
    if (state.isBusy) return;

    const name = elements.playerName.value.trim();
    if (name.length < 1 || name.length > 24) {
        setMessage(elements.loginMessage, 'Имя должно содержать от 1 до 24 символов.');
        return;
    }

    setBusy(true, elements.loginButton);
    setMessage(elements.loginMessage, 'Подключаемся…', 'success');

    try {
        // Первый запрос получает токен, необходимый для GET /api/rooms.
        await authenticate(name);
        showView('rooms');
        await loadRooms();
    } catch (error) {
        setMessage(elements.loginMessage, explainApiError(error));
    } finally {
        setBusy(false, elements.loginButton);
    }
}

async function loadRooms() {
    setMessage(elements.roomsMessage, '');
    elements.roomList.replaceChildren(makeTextElement('p', 'Загружаем комнаты…', 'muted'));

    try {
        const rooms = await apiRequest('/api/rooms');
        renderRooms(Array.isArray(rooms) ? rooms : []);
    } catch (error) {
        elements.roomList.replaceChildren(makeTextElement('p', explainApiError(error), 'muted'));
        setMessage(elements.roomsMessage, explainApiError(error));
    }
}

function renderRooms(rooms) {
    elements.roomList.replaceChildren();

    if (rooms.length === 0) {
        elements.roomList.append(makeTextElement('p', 'Пока нет открытых комнат. Создай первую!', 'empty-state'));
        return;
    }

    for (const room of rooms) {
        const card = document.createElement('article');
        card.className = 'room-card';

        const info = document.createElement('div');
        const title = makeTextElement('h3', room.name || `Комната ${room.code}`);
        const playersNow = Number(room.players ?? 0);
        const maxPlayers = Number(room.maxPlayers ?? 0);
        const playerText = maxPlayers > 0
            ? `${playersNow} / ${maxPlayers} игроков`
            : `${playersNow} игроков`;
        const subtitle = makeTextElement('p', `Код: ${room.code} · ${playerText}`);
        info.append(title, subtitle);

        const joinButton = makeButton('Войти', 'button button-primary');
        joinButton.addEventListener('click', () => joinRoom(room.code, joinButton));
        card.append(info, joinButton);
        elements.roomList.append(card);
    }
}

async function joinRoom(code, button) {
    if (state.isBusy) return;
    setBusy(true, button);
    setMessage(elements.roomsMessage, `Подключаемся к комнате ${code}…`, 'success');

    try {
        // Повторный /api/auth с room привязывает новую сессию игрока к выбранной комнате.
        await authenticate(state.name, code);
        await enterLobby(code);
    } catch (error) {
        setMessage(elements.roomsMessage, explainApiError(error));
    } finally {
        setBusy(false, button);
    }
}

async function createRoom() {
    if (state.isBusy) return;
    setBusy(true, elements.createRoomButton);
    setMessage(elements.roomsMessage, 'Создаём комнату…', 'success');

    try {
        const created = await apiRequest('/api/rooms', {
            method: 'POST',
            body: {}
        });

        if (!created.code) {
            throw new ApiError('Сервер не вернул уникальный код комнаты.', 500, created);
        }

        state.ownerId = String(created.ownerId ?? state.playerId);
        await enterLobby(created.code, created);
    } catch (error) {
        setMessage(elements.roomsMessage, explainApiError(error));
    } finally {
        setBusy(false, elements.createRoomButton);
    }
}

async function enterLobby(code, createdRoom = null) {
    stopRoomPolling();
    state.roomCode = String(code);
    if (createdRoom?.ownerId !== undefined) state.ownerId = String(createdRoom.ownerId);

    elements.inviteCode.textContent = state.roomCode;
    elements.roleBadge.hidden = String(state.ownerId) !== String(state.playerId);
    elements.lobbySubtitle.textContent = 'Ожидаем подключения остальных игроков.';
    setMessage(elements.lobbyMessage, '');
    showView('lobby');

    await refreshRoom();
    connectLobbySocket();
    state.roomPoll = window.setInterval(() => {
        refreshRoom().catch(error => setMessage(elements.lobbyMessage, explainApiError(error, 'room')));
    }, 2500);
}

async function refreshRoom() {
    if (!state.roomCode) return;
    const room = await apiRequest(`/api/rooms/${encodeURIComponent(state.roomCode)}`);

    if (room.ownerId !== undefined && room.ownerId !== null) {
        state.ownerId = String(room.ownerId);
    }

    const isModerator = String(state.ownerId) === String(state.playerId);
    elements.roleBadge.hidden = !isModerator;
    elements.lobbySubtitle.textContent = isModerator
        ? 'Ты модератор этой комнаты. Можно исключать участников из списка.'
        : 'Ожидаем подключения остальных игроков.';

    renderParticipants(Array.isArray(room.players) ? room.players : [], isModerator);
    setMessage(elements.lobbyMessage, '');
}

function renderParticipants(players, isModerator) {
    elements.participantList.replaceChildren();
    elements.participantCount.textContent = String(players.length);

    if (players.length === 0) {
        elements.participantList.append(makeTextElement('li', 'Пока никого нет. Ожидаем игроков…', 'empty-state'));
        return;
    }

    for (const player of players) {
        const playerId = String(player.id ?? player.playerId ?? '');
        const playerName = player.name || 'Игрок';
        const row = document.createElement('li');
        row.className = 'participant-row';

        const identity = document.createElement('div');
        const name = document.createElement('span');
        name.className = 'participant-name';
        name.textContent = playerName;
        identity.append(name);

        if (playerId && playerId === String(state.ownerId)) {
            const badge = makeTextElement('span', 'модератор', 'participant-tag');
            identity.append(badge);
        }
        if (playerId && playerId === String(state.playerId)) {
            identity.append(makeTextElement('span', 'это вы', 'participant-tag'));
        }

        row.append(identity);

        const canKick = isModerator && playerId && playerId !== String(state.playerId);
        if (canKick) {
            const kickButton = makeButton('Исключить', 'kick-button');
            kickButton.setAttribute('aria-label', `Исключить игрока ${playerName}`);
            kickButton.addEventListener('click', () => removePlayer(playerId, playerName, kickButton));
            row.append(kickButton);
        }

        elements.participantList.append(row);
    }
}

async function removePlayer(playerId, playerName, button) {
    if (!confirm(`Исключить игрока «${playerName}» из комнаты?`)) return;
    button.disabled = true;

    try {
        // Этот DELETE-маршрут нужно добавить в Go backend — в текущей спецификации его пока нет.
        await apiRequest(`/api/rooms/${encodeURIComponent(state.roomCode)}/players/${encodeURIComponent(playerId)}`, {
            method: 'DELETE'
        });
        setMessage(elements.lobbyMessage, `Игрок «${playerName}» исключён.`, 'success');
        await refreshRoom();
    } catch (error) {
        if (error.status === 404 || error.status === 405) {
            setMessage(elements.lobbyMessage,
                'Кнопка готова, но backend пока не поддерживает исключение игроков. Добавь DELETE /api/rooms/{code}/players/{id}.');
        } else {
            setMessage(elements.lobbyMessage, explainApiError(error));
        }
    } finally {
        button.disabled = false;
    }
}

function connectLobbySocket() {
    closeLobbySocket();
    if (!state.token || !state.roomCode) return;

    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws?token=${encodeURIComponent(state.token)}`;
    const socket = new WebSocket(wsUrl);
    state.socket = socket;

    socket.addEventListener('open', () => {
        // Типы сообщений взяты из backend-спецификации.
        // Формат конверта { type: ..., ... } нужно сверить с shared/protocol.js.
        socket.send(JSON.stringify({
            type: 'auth',
            token: state.token,
            name: state.name
        }));
    });    

    socket.addEventListener('message', (event) => {
        let message;
        try { message = JSON.parse(event.data); }
        catch { return; }

        const type = message.type || message.event;
        if (type === 'authOk') {
            socket.send(JSON.stringify({ type: 'subscribe', channels: ['players'] }));
        } else if (type === 'authFail') {
            setMessage(elements.lobbyMessage, `WebSocket: ${message.reason || 'не удалось авторизоваться'}`);
        } else if (type === 'players') {
            const players = Array.isArray(message.players) ? message.players : [];
            const isModerator = String(state.ownerId) === String(state.playerId);
            renderParticipants(players, isModerator);
        } else if (type === 'playerLeave') {
            // REST-запрос подтянет актуальные данные комнаты.
            refreshRoom().catch(error => setMessage(elements.lobbyMessage, explainApiError(error, 'room')));
        } else if (type === 'toast') {
            setMessage(elements.lobbyMessage, message.text || 'Сообщение сервера');
        }
    });

    socket.addEventListener('close', (event) => {
        if (state.socket !== socket) return;
        state.socket = null;
        if (event.code === 4404) {
            setMessage(elements.lobbyMessage, 'Комната была удалена модератором сервера.');
        } else if (event.code === 4401 || event.code === 4400) {
            setMessage(elements.lobbyMessage, 'WebSocket отклонил токен. Попробуй войти заново.');
        }
        // REST-опрос продолжит обновлять список, даже если WS недоступен.
    });

    socket.addEventListener('error', () => {
        setMessage(elements.lobbyMessage, 'Не удалось открыть WebSocket. Список участников пока обновляется через HTTP.');
    });
}

function closeLobbySocket() {
    if (state.socket) {
        const socket = state.socket;
        state.socket = null;
        if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
            socket.close(1000, 'Leaving lobby');
        }
    }
}

function stopRoomPolling() {
    if (state.roomPoll !== null) {
        window.clearInterval(state.roomPoll);
        state.roomPoll = null;
    }
}

async function copyRoomCode() {
    try {
        await navigator.clipboard.writeText(state.roomCode);
        setMessage(elements.lobbyMessage, 'Код комнаты скопирован.', 'success');
    } catch {
        setMessage(elements.lobbyMessage, `Не удалось скопировать автоматически. Код комнаты: ${state.roomCode}`);
    }
}

function logout() {
    clearSession();
    elements.playerName.value = '';
    showView('login');
    setMessage(elements.loginMessage, '');
}

function setBusy(busy, button) {
    state.isBusy = busy;
    if (button) button.disabled = busy;
}

function makeTextElement(tag, text, className = '') {
    const element = document.createElement(tag);
    element.textContent = text;
    if (className) element.className = className;
    return element;
}

function makeButton(text, className) {
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = text;
    button.className = className;
    return button;
}


const testButton = document.querySelector('#dev-test-button');

testButton?.addEventListener('click', () => {

    console.log('Нажата тестовая кнопка');

    window.dispatchEvent(new CustomEvent('urbanist:start-game', {
        detail: {
            mode: 'test'
        }
    }));

});

// Обработчики интерфейса.
elements.loginForm.addEventListener('submit', handleLogin);
elements.createRoomButton.addEventListener('click', createRoom);
elements.refreshRoomsButton.addEventListener('click', loadRooms);
elements.logoutButton.addEventListener('click', logout);
elements.copyCodeButton.addEventListener('click', copyRoomCode);
elements.backToRoomsButton.addEventListener('click', async () => {
    stopRoomPolling();
    closeLobbySocket();
    state.roomCode = '';
    showView('rooms');
    await loadRooms();
});

// После регистрации подставляем имя из query-параметра.
const params = new URLSearchParams(window.location.search);
const registeredName = params.get('name');
if (registeredName) elements.playerName.value = registeredName.slice(0, 24);
if (params.get('registered') === '1') {
    setMessage(elements.loginMessage, 'Регистрация завершена. Теперь войди в игру.', 'success');
}

// Восстанавливаем сессию после перезагрузки вкладки, если токен ещё действителен.
if (restoreSession()) {
    showView('rooms');
    loadRooms();
} else {
    showView('login');
}
