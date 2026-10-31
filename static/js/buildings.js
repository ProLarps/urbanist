import * as THREE from 'three';

export function createBuilding(scene, x, z) {

    const geometry = new THREE.BoxGeometry(5, 8, 5);

    const material = new THREE.MeshStandardMaterial({
        color: 0xd5c4a1
    });

    const building = new THREE.Mesh(
        geometry,
        material
    );

    building.position.set(x, 4, z);

    scene.add(building);

    return building;
}