import React, {useState} from 'react';
import {ActivityIndicator, Modal, StyleSheet, Text, TouchableOpacity, View} from 'react-native';
import {useTranslation} from 'react-i18next';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {PremiumModal} from './PremiumModal';
import {RewardedUnlockModal} from './RewardedUnlockModal';
import {consumeAdOneCard} from '../features/dailyCard';
import {isApiError} from '../features/apiClient';

type Props = {
    visible: boolean;
    todayReadingId?: string;
    onClose: () => void;
    onOpenToday: () => void;
    onUnlocked: (result: {consumed: boolean; kind: 'daily' | 'bonus'}) => void;
};

export const DailyLimitModal = ({visible, todayReadingId, onClose, onOpenToday, onUnlocked}: Props) => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const [premiumOpen, setPremiumOpen] = useState(false);
    const [videoOpen, setVideoOpen] = useState(false);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleAdToken = async (adToken: string) => {
        setVideoOpen(false);
        setBusy(true);
        setError(null);
        try {
            const kind = await consumeAdOneCard(adToken);
            onUnlocked({consumed: true, kind});
        } catch (err) {
            if (isApiError(err) && err.code === 'quota_exceeded') {
                setError(t('dailyCard.adMax'));
            } else if (isApiError(err) && err.code === 'ad_watch_incomplete') {
                setError(t('dailyCard.watchIncomplete'));
            } else {
                setError(t('dailyCard.unlockError'));
            }
        } finally {
            setBusy(false);
        }
    };

    return (
        <>
            <Modal visible={visible && !premiumOpen && !videoOpen} animationType="slide" transparent onRequestClose={onClose}>
                <View style={styles.overlay}>
                    <View
                        style={[
                            styles.box,
                            {backgroundColor: colors.tabBar, borderColor: hexAlpha(colors.accent, 0.35)},
                        ]}
                    >
                        <Text style={[styles.title, {color: colors.gold}]}>{t('dailyCard.limitTitle')}</Text>
                        <Text style={[styles.body, {color: colors.text}]}>{t('dailyCard.limitBody')}</Text>
                        {error ? <Text style={[styles.error, {color: colors.danger}]}>{error}</Text> : null}
                        {todayReadingId ? (
                            <TouchableOpacity
                                style={[styles.secondary, {borderColor: hexAlpha(colors.gold, 0.4)}]}
                                onPress={onOpenToday}
                            >
                                <Text style={[styles.secondaryText, {color: colors.gold}]}>
                                    {t('dailyCard.ctaOpen')}
                                </Text>
                            </TouchableOpacity>
                        ) : null}
                        <TouchableOpacity
                            style={[styles.primary, {backgroundColor: colors.accent}]}
                            onPress={() => setVideoOpen(true)}
                            disabled={busy}
                        >
                            {busy ? (
                                <ActivityIndicator color="#fff" />
                            ) : (
                                <Text style={styles.primaryText}>{t('dailyCard.watch')}</Text>
                            )}
                        </TouchableOpacity>
                        <TouchableOpacity
                            style={[styles.primary, {backgroundColor: colors.gold}]}
                            onPress={() => setPremiumOpen(true)}
                            disabled={busy}
                        >
                            <Text style={[styles.primaryDark]}>{t('dailyCard.subscribe')}</Text>
                        </TouchableOpacity>
                        <TouchableOpacity onPress={onClose} disabled={busy}>
                            <Text style={[styles.cancel, {color: colors.muted}]}>{t('premium.notNow')}</Text>
                        </TouchableOpacity>
                    </View>
                </View>
            </Modal>
            <PremiumModal
                visible={premiumOpen}
                onClose={() => setPremiumOpen(false)}
                onActivated={() => {
                    setPremiumOpen(false);
                    onUnlocked({consumed: false, kind: 'daily'});
                }}
            />
            <RewardedUnlockModal
                visible={videoOpen}
                onClose={() => setVideoOpen(false)}
                onUnlocked={(token) => {
                    handleAdToken(token).catch(() => {});
                }}
                onQuota={() => {
                    setVideoOpen(false);
                    setError(t('dailyCard.adMax'));
                }}
            />
        </>
    );
};

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.65)',
        justifyContent: 'flex-end',
    },
    box: {
        borderTopLeftRadius: 24,
        borderTopRightRadius: 24,
        borderWidth: 1,
        padding: 24,
        gap: 14,
        paddingBottom: 36,
    },
    title: {
        fontSize: 22,
        fontWeight: '700',
    },
    body: {
        fontSize: 15,
        lineHeight: 22,
        opacity: 0.9,
    },
    error: {
        fontSize: 14,
    },
    primary: {
        borderRadius: 20,
        paddingVertical: 14,
        alignItems: 'center',
    },
    primaryText: {
        color: '#fff',
        fontWeight: '700',
        fontSize: 16,
    },
    primaryDark: {
        color: '#08070f',
        fontWeight: '700',
        fontSize: 16,
    },
    secondary: {
        borderWidth: 1,
        borderRadius: 20,
        paddingVertical: 12,
        alignItems: 'center',
    },
    secondaryText: {
        fontWeight: '600',
        fontSize: 16,
    },
    cancel: {
        textAlign: 'center',
        marginTop: 4,
        fontSize: 15,
    },
});
