import React, {useCallback, useEffect, useRef, useState} from 'react';
import {ActivityIndicator, Modal, StyleSheet, Text, View} from 'react-native';
import {useVideoPlayer, VideoView} from 'expo-video';
import {useTranslation} from 'react-i18next';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {beginAdSession} from '../features/dailyCard';
import {isApiError} from '../features/apiClient';

const UNLOCK_VIDEO = require('../assets/splah_video.mp4');

type Props = {
    visible: boolean;
    onClose: () => void;
    onUnlocked: (adToken: string) => void;
    onQuota: () => void;
};

export const RewardedUnlockModal = ({visible, onClose, onUnlocked, onQuota}: Props) => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [secondsLeft, setSecondsLeft] = useState(15);
    const onUnlockedRef = useRef(onUnlocked);
    const onQuotaRef = useRef(onQuota);
    onUnlockedRef.current = onUnlocked;
    onQuotaRef.current = onQuota;
    const tokenRef = useRef<string | null>(null);
    const doneRef = useRef(false);

    const player = useVideoPlayer(UNLOCK_VIDEO, (instance) => {
        instance.loop = true;
        instance.muted = false;
        instance.pause();
    });

    const finish = useCallback(() => {
        if (doneRef.current || !tokenRef.current) {
            return;
        }
        doneRef.current = true;
        player.pause();
        onUnlockedRef.current(tokenRef.current);
    }, [player]);

    useEffect(() => {
        if (!visible) {
            player.pause();
            tokenRef.current = null;
            doneRef.current = false;
            setLoading(true);
            setError(null);
            return;
        }

        let cancelled = false;
        let tick: ReturnType<typeof setInterval> | undefined;
        (async () => {
            setLoading(true);
            setError(null);
            try {
                const session = await beginAdSession();
                if (cancelled) {
                    return;
                }
                tokenRef.current = session.adToken;
                const wait = Math.max(5, session.minWatchSeconds);
                setSecondsLeft(wait);
                setLoading(false);
                player.play();
                let left = wait;
                tick = setInterval(() => {
                    left -= 1;
                    setSecondsLeft(Math.max(0, left));
                    if (left <= 0 && tick) {
                        clearInterval(tick);
                        finish();
                    }
                }, 1000);
            } catch (err) {
                if (cancelled) {
                    return;
                }
                setLoading(false);
                if (isApiError(err) && err.code === 'quota_exceeded') {
                    onQuotaRef.current();
                    return;
                }
                if (isApiError(err) && err.code === 'rate_limited') {
                    setError(t('dailyCard.needWait'));
                    return;
                }
                setError(t('dailyCard.unlockError'));
            }
        })();

        return () => {
            cancelled = true;
            if (tick) {
                clearInterval(tick);
            }
            player.pause();
        };
    }, [finish, player, t, visible]);

    return (
        <Modal visible={visible} animationType="fade" transparent onRequestClose={() => {}}>
            <View style={styles.overlay}>
                <View
                    style={[
                        styles.box,
                        {backgroundColor: colors.tabBar, borderColor: hexAlpha(colors.accent, 0.35)},
                    ]}
                >
                    <Text style={[styles.title, {color: colors.gold}]}>{t('dailyCard.watching')}</Text>
                    {loading ? <ActivityIndicator color={colors.accent} /> : null}
                    {!loading && !error ? (
                        <>
                            <VideoView
                                player={player}
                                style={styles.video}
                                contentFit="cover"
                                nativeControls={false}
                            />
                            <Text style={[styles.timer, {color: colors.text}]}>
                                {t('dailyCard.secondsLeft', {count: secondsLeft})}
                            </Text>
                        </>
                    ) : null}
                    {error ? (
                        <Text style={[styles.error, {color: colors.danger}]} onPress={onClose}>
                            {error}
                        </Text>
                    ) : null}
                </View>
            </View>
        </Modal>
    );
};

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.72)',
        justifyContent: 'center',
        padding: 24,
    },
    box: {
        borderRadius: 20,
        borderWidth: 1,
        padding: 20,
        gap: 16,
    },
    title: {
        fontSize: 18,
        fontWeight: '700',
        textAlign: 'center',
    },
    video: {
        width: '100%',
        height: 180,
        borderRadius: 12,
        overflow: 'hidden',
    },
    timer: {
        textAlign: 'center',
        fontSize: 16,
        fontWeight: '600',
    },
    error: {
        textAlign: 'center',
        fontSize: 15,
        textDecorationLine: 'underline',
    },
});
