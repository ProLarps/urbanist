import * as THREE from 'three';

export function createWorld(scene) {

    // Освещение
    const ambientLight = new THREE.HemisphereLight(
        0xffffff,
        0x667755,
        2
    );

    scene.add(ambientLight);

    const sun = new THREE.DirectionalLight(
        0xffffff,
        2
    );

    sun.position.set(10, 20, 10);
    scene.add(sun);


    // Земля
    const ground = new THREE.Mesh(
        new THREE.PlaneGeometry(50, 50),
        new THREE.MeshStandardMaterial({
            color: 0x75a85b
        })
    );

    ground.rotation.x = -Math.PI / 2;

    scene.add(ground);


    // Дорога
    const road = new THREE.Mesh(
        new THREE.BoxGeometry(40, 0.1, 6),
        new THREE.MeshStandardMaterial({
            color: 0x444444
        })
    );

    road.position.set(0, 0.05, 0);

    scene.add(road);
}
