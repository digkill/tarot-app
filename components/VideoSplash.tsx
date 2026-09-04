import React, {useCallback, useEffect, useRef, useState} from 'react';
import {Image, Pressable, StyleSheet, Text, View} from 'react-native';
import {useTranslation} from 'react-i18next';
import {useVideoPlayer, VideoView} from 'expo-video';
import * as SplashScreen from 'expo-splash-screen';

const SPLASH_VIDEO = require('../assets/splah_video.mp4');
const SPLASH_ICON = require('../assets/icon.png');
const BRAND_MS = 2200;
const MAX_MS = 12000;

type Props = {
    onDone: () => void;
};

export const VideoSplash = ({onDone}: Props) => {
    const {t} = useTranslation();
    const finished = useRef(false);
    const [showBrand, setShowBrand] = useState(true);

    const finish = useCallback(() => {
        if (finished.current) {
            return;
        }
        finished.current = true;
        SplashScreen.hideAsync().catch(() => {});
        onDone();
    }, [onDone]);

    const player = useVideoPlayer(SPLASH_VIDEO, (instance) => {
        instance.loop = false;
        instance.muted = false;
        instance.pause();
    });

    useEffect(() => {
        SplashScreen.hideAsync().catch(() => {});
    }, []);

    useEffect(() => {
        const ended = player.addListener('playToEnd', finish);
        const failed = player.addListener('statusChange', ({status}) => {
            if (status === 'error') {
                finish();
            }
        });
        const brandTimer = setTimeout(() => {
            setShowBrand(false);
            player.play();
        }, BRAND_MS);
        const timeout = setTimeout(finish, MAX_MS);
        return () => {
            ended.remove();
            failed.remove();
            clearTimeout(brandTimer);
            clearTimeout(timeout);
        };
    }, [finish, player]);

    return (
        <Pressable style={styles.root} onPress={finish}>
            <VideoView
                player={player}
                style={styles.video}
                contentFit="cover"
                nativeControls={false}
                pointerEvents="none"
            />
            {showBrand ? (
                <View style={styles.brand} pointerEvents="none">
                    <Image source={SPLASH_ICON} style={styles.icon} />
                    <Text style={styles.powered}>{t('splash.poweredBy')}</Text>
                </View>
            ) : null}
            <Text style={styles.skip}>{t('splash.skip')}</Text>
        </Pressable>
    );
};

const styles = StyleSheet.create({
    root: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        width: '100%',
        height: '100%',
        backgroundColor: '#0B1220',
        zIndex: 100,
        overflow: 'hidden',
    },
    video: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        width: '100%',
        height: '100%',
    },
    brand: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#0B1220',
        paddingHorizontal: 32,
    },
    icon: {
        width: 196,
        height: 196,
        borderRadius: 40,
    },
    powered: {
        marginTop: 28,
        color: '#f7f4ea',
        opacity: 0.78,
        fontSize: 15,
        fontWeight: '600',
        letterSpacing: 1.2,
        textAlign: 'center',
    },
    skip: {
        position: 'absolute',
        right: 24,
        bottom: 48,
        color: '#f7f4ea',
        opacity: 0.75,
        fontSize: 15,
        fontWeight: '600',
    },
});
