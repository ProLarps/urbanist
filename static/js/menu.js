
const menuRoot = document.getElementById('menu-root');
const gameScreen = document.getElementById('game-screen');


// ==========================================
// ПЕРЕКЛЮЧЕНИЕ ЭКРАНОВ
// ==========================================

function showScreen(screenId) {

    // Скрываем все экраны
    document.querySelectorAll('.screen').forEach(screen => {
        screen.classList.remove('active');
    });

    // Показываем нужный экран
    const screen = document.getElementById(screenId);

    if (screen) {
        screen.classList.add('active');
    }

    // Очищаем сообщения об ошибках
    document.querySelectorAll('.form-error').forEach(error => {
        error.textContent = '';
    });
}


// ==========================================
// ОБРАБОТКА ОШИБОК
// ==========================================

function showError(elementId, message) {

    document.getElementById(elementId).textContent = message;

}


// ==========================================
// ЗАПУСК ИГРЫ
// ==========================================

async function launchGame(options, button, errorElementId) {

    const originalText = button.textContent;

    try {

        button.disabled = true;
        button.textContent = 'Загрузка...';

        // Загружаем main.js только при запуске игры
        const { startGame } = await import('./main.js');

        await startGame(options);

        menuRoot.hidden = true;
        gameScreen.hidden = false;

    } catch (error) {

        console.error(error);

        showError(
            errorElementId,
            error.message || 'Не удалось запустить игру.'
        );

    } finally {

        button.disabled = false;
        button.textContent = originalText;

    }
}


// ==========================================
// ГЛАВНОЕ МЕНЮ → АВТОРИЗАЦИЯ
// ==========================================

document
    .getElementById('play-button')
    .addEventListener('click', () => {

        console.log('Кнопка Играть нажата!');

        showScreen('auth-screen');

    });


// ==========================================
// АВТОРИЗАЦИЯ → ОСНОВНОЕ МЕНЮ
// ==========================================

document
    .getElementById('auth-form')
    .addEventListener('submit', (event) => {

        event.preventDefault();

        const login = document.getElementById('login').value.trim();

        const password = document.getElementById('password').value;


        // Проверка логина
        if (!login) {

            showError('auth-error', 'Введите логин.');

            return;
        }

        if (login.length < 3) {

            showError('auth-error', 'Логин должен содержать минимум 3 символа.');

            return;
        }


        // Проверка пароля
        if (!password) {

            showError('auth-error', 'Введите пароль.');

            return;
        }


        // Пока пропускаем пользователя дальше
        // Здесь позднее появится настоящая авторизация

        showScreen('main-menu-screen');

    });


// ==========================================
// ОСНОВНОЕ МЕНЮ → СОЗДАНИЕ МИРА
// ==========================================

document
    .getElementById('create-world')
    .addEventListener('click', async (event) => {

        const button = event.currentTarget;

        await launchGame(
            {
                mode: 'create'
            },
            button,
            'main-menu-error'
        );

    });


// ==========================================
// ОСНОВНОЕ МЕНЮ → ВВОД КЛЮЧА МИРА
// ==========================================

document
    .getElementById('join-world')
    .addEventListener('click', () => {

        showScreen('world-key-screen');

    });


// ==========================================
// ВХОД В МИР ПО КЛЮЧУ
// ==========================================

document
    .getElementById('world-key-form')
    .addEventListener('submit', async (event) => {

        event.preventDefault();

        const worldKey = document
            .getElementById('world-key')
            .value
            .trim();


        // Проверка ключа
        if (!worldKey) {

            showError('world-key-error', 'Введите ключ мира.');

            return;
        }


        const button = event.currentTarget.querySelector('button[type="submit"]');


        // Передаём ключ в main.js
        await launchGame(
            {
                mode: 'join',
                worldKey: worldKey
            },
            button,
            'world-key-error'
        );

    });


// ==========================================
// КНОПКИ НАЗАД
// ==========================================

document.querySelectorAll('[data-back]').forEach(button => {

    button.addEventListener('click', () => {

        const targetScreen = button.dataset.back;

        showScreen(targetScreen);

    });

});