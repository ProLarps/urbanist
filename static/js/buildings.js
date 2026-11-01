import * as THREE from 'three';
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';

const loader = new GLTFLoader();

export async function createBuilding(scene, x, z) {

    // Загружаем модель
    const gltf = await loader.loadAsync(
        '/assets/models/buildings/cube.glb'
    );

    // Получаем объект модели
    const building = gltf.scene;

    // Устанавливаем положение
    building.position.set(x, 0, z);

    // Добавляем модель в сцену
    scene.add(building);

    return building;
}