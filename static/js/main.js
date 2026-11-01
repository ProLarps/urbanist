import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

import { createWorld } from './createWorld.js';
import '../css/style_main.css';

export async function startGame(options) {

    const gameContainer = document.getElementById('game-container');

    // Очищаем контейнер
    gameContainer.replaceChildren();


    // Проверяем режим запуска
    if (options.mode === 'create') {

        console.log('Создание нового мира');

        // Здесь будет логика создания мира

    }

    else if (options.mode === 'join') {

        console.log('Подключение к миру:', options.worldKey);

        // Здесь будет проверка ключа и подключение к миру

    }

    else {

        throw new Error('Неизвестный режим запуска.');

    }


    // Здесь будет инициализация Three.js
    // Например:
    // const renderer = new THREE.WebGLRenderer();
    // gameContainer.appendChild(renderer.domElement);

}

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

document.body.appendChild(renderer.domElement);

// Временное управление камерой для разработки
const controls = new OrbitControls(camera, renderer.domElement);
controls.target.set(0, 0, 0);
controls.update();

// Создаём игровой мир
createWorld(scene);

// Обработка изменения размера окна
window.addEventListener('resize', () => {
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();

    renderer.setSize(window.innerWidth, window.innerHeight);
});

// Игровой цикл
function animate() {
    requestAnimationFrame(animate);

    controls.update();
    renderer.render(scene, camera);
}

animate();