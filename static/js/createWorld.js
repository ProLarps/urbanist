import * as THREE from 'three';
import { createBuilding } from './buildings.js';
import { createRoadStraight, createRoadStraight90, createRoadCrossroadPath } from './environment.js';

const LenChunk = 16;
export function createWorld(scene) {
    let x_blocks = 2;
    for (let i = -x_blocks; i <= x_blocks; i++) {
        for (let j = -x_blocks; j <= x_blocks; j++){
            createRoadCrossroadPath(scene, i*LenChunk, j*LenChunk);
            for (let t = i*LenChunk - LenChunk/2; t < i*LenChunk + LenChunk/2; t++){
                if (t != i*LenChunk) {
                    createRoadStraight(scene, t, j*LenChunk);
                }
            }
            for (let t = j*LenChunk - LenChunk/2; t < j*LenChunk + LenChunk/2; t++){
                if (t != i*LenChunk) {
                    createRoadStraight90(scene, i*LenChunk, t);
                }
            }
        }
    }
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
        new THREE.PlaneGeometry(1000, 1000),
        new THREE.MeshStandardMaterial({
            color: 0x75a85b
        })
    );

    ground.rotation.x = -Math.PI / 2;

    scene.add(ground);
}