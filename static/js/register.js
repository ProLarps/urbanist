const form = document.querySelector('#register-form');
const nameInput = document.querySelector('#register-name');
const passwordInput = document.querySelector('#register-password');
const repeatInput = document.querySelector('#register-password-repeat');
const button = document.querySelector('#register-button');
const message = document.querySelector('#register-message');

function showMessage(text, success = false) {
    message.textContent = text;
    message.classList.toggle('success', success);
}

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const name = nameInput.value.trim();
    const password = passwordInput.value;
    const passwordRepeat = repeatInput.value;

    if (name.length < 1 || name.length > 24) {
        showMessage('Имя должно содержать от 1 до 24 символов.');
        return;
    }
    if (password.length < 8) {
        showMessage('Пароль должен содержать не менее 8 символов.');
        return;
    }
    if (password !== passwordRepeat) {
        showMessage('Пароли не совпадают.');
        return;
    }

    button.disabled = true;
    showMessage('Создаём аккаунт…', true);

    try {
        // Предлагаемый маршрут: его нужно реализовать на Go backend.
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
            body: JSON.stringify({ name, password })
        });

        const raw = await response.text();
        let result = {};
        if (raw) {
            try { result = JSON.parse(raw); }
            catch { result = { message: raw }; }
        }

        if (!response.ok) {
            if (response.status === 404 || response.status === 405) {
                throw new Error('Регистрация ещё не подключена на сервере. Backend должен добавить POST /api/register.');
            }
            if (response.status === 409) {
                throw new Error('Это имя уже занято. Выбери другое.');
            }
            throw new Error(result.message || result.error || `Ошибка регистрации (${response.status}).`);
        }

        window.location.href = `./index.html?registered=1&name=${encodeURIComponent(name)}`;
    } catch (error) {
        showMessage(error.message || 'Не удалось связаться с сервером.');
        button.disabled = false;
    }
});
