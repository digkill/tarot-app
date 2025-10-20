// components/TiledBackground.tsx
import React, {useRef, useEffect} from "react";
import {GLView} from "expo-gl";
import type {ExpoWebGLRenderingContext} from "expo-gl";
import {Renderer} from "expo-three";
import * as THREE from "three";
import {Asset} from "expo-asset";

export default function TiledBackground() {
    const animationRef = useRef<number | null>(null);

    useEffect(() => {
        return () => {
            if (animationRef.current) cancelAnimationFrame(animationRef.current);
        };
    }, []);

    const onContextCreate = async (gl: ExpoWebGLRenderingContext) => {
        const scene = new THREE.Scene();
        const camera = new THREE.OrthographicCamera(
            -1, 1, 1, -1, 0.1, 10
        );
        camera.position.z = 1;

        const renderer = new Renderer({gl}) as unknown as THREE.WebGLRenderer;
        renderer.setSize(gl.drawingBufferWidth, gl.drawingBufferHeight);

        // Загрузка текстуры (паттерна)
        const asset = Asset.fromModule(require("../assets/pattern2.png"));
        await asset.downloadAsync();
        const texture = new THREE.TextureLoader().load(asset.uri);
        texture.wrapS = THREE.RepeatWrapping;
        texture.wrapT = THREE.RepeatWrapping;
        texture.repeat.set(4, 4); // масштаб повторения
        (texture as any).colorSpace = THREE.SRGBColorSpace;

        const geometry = new THREE.PlaneGeometry(2, 2, 1, 1);
        const material = new THREE.MeshBasicMaterial({map: texture});

        const mesh = new THREE.Mesh(geometry, material);
        scene.add(mesh);
        scene.background = null;

        const animate = () => {
            animationRef.current = requestAnimationFrame(animate);

            // медленное движение текстуры (анимация)
            texture.offset.x += 0.00005;
            texture.offset.y += 0.00005;

            renderer.render(scene, camera);
            gl.endFrameEXP();
        };

        animate();
    };

    return (
        <GLView
            style={{
                position: "absolute",
                width: "100%",
                height: "100%",
                zIndex: -1,
            }}
            onContextCreate={onContextCreate}
        />
    );
}
