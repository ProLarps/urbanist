import * as THREE from 'three';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';

const loader = new GLTFLoader();

export async function createRoadCrossroadPath(scene, x, z) {

    // Загружаем модель
    const gltf = await loader.loadAsync(
        '/assets/models/environment/road-crossroad-path.glb'
    );

    // Получаем объект модели
    const road = gltf.scene;

    // Устанавливаем положение
    road.position.set(x, 0, z);

    // Добавляем модель в сцену
    scene.add(road);
}

export async function createRoadStraight(scene, x, z) {

    // Загружаем модель
    const gltf = await loader.loadAsync(
        '/assets/models/environment/road-straight.glb'
    );

    // Получаем объект модели
    const road = gltf.scene;

    // Устанавливаем положение
    road.position.set(x, 0, z);

    // Добавляем модель в сцену
    scene.add(road);
}

export async function createRoadStraight90(scene, x, z) {

    // Загружаем модель
    const gltf = await loader.loadAsync(
        '/assets/models/environment/road-straight.glb'
    );

    // Получаем объект модели
    const road = gltf.scene;

    // Устанавливаем положение
    road.rotation.y = Math.PI / 2;
    road.position.set(x, 0, z);

    // Добавляем модель в сцену
    scene.add(road);
}
