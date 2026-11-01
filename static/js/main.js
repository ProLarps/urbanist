import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

import { createWorld } from './createWorld.js';
import '../css/menu.css';

let gameStarted = false;

// Основная функция запуска игры
export async function startGame(options = {}) {

    if (gameStarted) return;

    const gameContainer = document.getElementById('game-container');

    if (!gameContainer) {
        throw new Error('Не найден game-container!');
    }

    // Проверяем режим запуска
    if (options.mode === 'test') {

        console.log('Запуск тестового режима');

    }

    else if (options.mode === 'create') {

        console.log('Создание нового мира');

    }

    else if (options.mode === 'join') {

        console.log('Подключение к миру:', options.worldKey);

    }

    else {

        throw new Error('Неизвестный режим запуска.');

    }

    // Очищаем контейнер
    gameContainer.replaceChildren();

    // Создаём сцену
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x87ceeb);

    // Создаём камеру
    const camera = new THREE.PerspectiveCamera(
        60,
        window.innerWidth / window.innerHeight,
        0.1,
        1000
    );

    camera.position.set(20, 20, 20);

    // Создаём рендерер
    const renderer = new THREE.WebGLRenderer({
        antialias: true
    });

    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

    // Добавляем canvas в контейнер игры
    gameContainer.appendChild(renderer.domElement);

    // Временное управление камерой
    const controls = new OrbitControls(camera, renderer.domElement);

    controls.target.set(0, 0, 0);
    controls.update();

    // Создаём игровой мир
    createWorld(scene);

    // Обработка изменения размера окна
    function resize() {

        camera.aspect = window.innerWidth / window.innerHeight;
        camera.updateProjectionMatrix();

        renderer.setSize(window.innerWidth, window.innerHeight);

    }

    window.addEventListener('resize', resize);

    // Скрываем меню после успешного запуска
    document.querySelector('#menu')?.style.setProperty('display', 'none');

    gameStarted = true;

    // Игровой цикл
    function animate() {

        requestAnimationFrame(animate);

        controls.update();

        renderer.render(scene, camera);

    }

    animate();

    console.log('Игра успешно запущена!');
}


// Получаем событие от menu.js
window.addEventListener('urbanist:start-game', (event) => {

    startGame(event.detail).catch(error => {
        console.error('Ошибка запуска игры:', error);
    });

});