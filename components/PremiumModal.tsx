import React, {useEffect, useMemo, useState} from 'react';
import {
    Alert,
    Modal,
    Platform,
    View,
    Text,
    TouchableOpacity,
    StyleSheet,
    ScrollView,
    ActivityIndicator,
} from 'react-native';
import {useTranslation} from 'react-i18next';
import {useSettings} from '../providers/SettingsProvider';
import {useAuth} from '../providers/AuthProvider';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {isApiError} from '../features/apiClient';
import {reportPurchaseRequest} from '../features/billingApi';
import {startYooKassaCheckout} from '../features/yookassaCheckout';
import {premiumGate, type PremiumGate} from '../features/premiumSource';
import {
    describePayError,
    FALLBACK_PREMIUM_PRICES,
    formatRub,
    getPremiumProducts,
    isPaymentsSupported,
    PREMIUM_PRODUCT_IDS,
    promptInstallRuStore,
    purchasePremium,
    PurchaseCancelledError,
    readStorePremiumStatus,
    RuStoreMissingError,
    type PremiumProductId,
} from '../features/payments';
import type {Product} from '../libs/RuStoreReactPay';

type PremiumModalProps = {
    visible: boolean;
    onClose: () => void;
    onActivated?: () => void;
};

const PLAN_COPY: Record<PremiumProductId, {name: string; period: string; hint: string}> = {
    premium_monthly: {name: 'plans.monthly.name', period: 'plans.monthly.period', hint: 'plans.monthly.hint'},
    premium_yearly: {name: 'plans.yearly.name', period: 'plans.yearly.period', hint: 'plans.yearly.hint'},
    premium_lifetime: {name: 'plans.lifetime.name', period: 'plans.lifetime.period', hint: 'plans.lifetime.hint'},
};

export const PremiumModal = ({visible, onClose, onActivated}: PremiumModalProps) => {
    const {t} = useTranslation();
    const {setSetting, settings} = useSettings();
    const {user, refreshUser} = useAuth();
    const colors = useAppColors();
    const [processing, setProcessing] = useState(false);
    const [selected, setSelected] = useState<PremiumProductId>('premium_monthly');
    const [catalog, setCatalog] = useState<Product[]>([]);
    const gate = premiumGate(Boolean(user?.hasPremium || settings.hasPremium), user?.premiumSource);

    const managedCopy = (current: PremiumGate): {title: string; body: string} => {
        if (current.kind !== 'managed') {
            return {title: t('premium.title'), body: t('premium.benefits')};
        }
        if (current.provider === 'support') {
            return {title: t('premium.managedBySupportTitle'), body: t('premium.managedBySupportBody')};
        }
        if (current.provider === 'rustore') {
            return {
                title: t('premium.managedInRuStoreTitle'),
                body: current.local ? t('premium.managedInRuStoreLocal') : t('premium.managedInRuStoreOther'),
            };
        }
        return {
            title: t('premium.managedInYooKassaTitle'),
            body: current.local ? t('premium.managedInYooKassaLocal') : t('premium.managedInYooKassaOther'),
        };
    };

    useEffect(() => {
        if (!visible || !isPaymentsSupported()) {
            return;
        }
        getPremiumProducts()
            .then(setCatalog)
            .catch(() => {
                setCatalog([]);
            });
    }, [visible]);

    const priceById = useMemo(() => {
        const next: Record<string, string> = {};
        for (const id of PREMIUM_PRODUCT_IDS) {
            const fromStore = catalog.find((item) => item.productId === id);
            next[id] = fromStore?.amountLabel || formatRub(FALLBACK_PREMIUM_PRICES[id]);
        }
        return next;
    }, [catalog]);

    const syncPurchase = async (input: Parameters<typeof reportPurchaseRequest>[0]) => {
        if (!user) {
            return;
        }
        try {
            await reportPurchaseRequest(input);
            await refreshUser();
        } catch (error) {
            console.warn('[billing] report purchase failed', error);
        }
    };

    const activatePremium = async () => {
        await setSetting('hasPremium', true);
        Alert.alert(t('premium.purchaseSuccess'), '', [
            {
                text: 'OK',
                onPress: () => {
                    onActivated?.();
                    onClose();
                },
            },
        ]);
    };

    const handleSubscribe = async () => {
        if (processing) {
            return;
        }

        if (gate.kind === 'managed') {
            const copy = managedCopy(gate);
            Alert.alert(copy.title, copy.body);
            return;
        }

        if (Platform.OS !== 'android') {
            setProcessing(true);
            try {
                const status = await startYooKassaCheckout(selected);
                if (status === null) {
                    return;
                }
                await refreshUser();
                if (status.hasPremium || status.status === 'paid') {
                    await activatePremium();
                    return;
                }
                Alert.alert(t('premium.checkoutPending'));
            } catch (error) {
                if (isApiError(error) && error.code === 'premium_already_active') {
                    const copy = managedCopy(premiumGate(true, user?.premiumSource));
                    Alert.alert(copy.title, copy.body);
                    await refreshUser();
                    return;
                }
                if (__DEV__ && isApiError(error) && error.code === 'payments_unavailable') {
                    await syncPurchase({productId: selected, source: 'dev'});
                    await activatePremium();
                    return;
                }
                console.warn('[payments] yookassa checkout failed', error);
                Alert.alert(t('premium.purchaseError'));
            } finally {
                setProcessing(false);
            }
            return;
        }

        if (!isPaymentsSupported()) {
            if (__DEV__) {
                await syncPurchase({productId: selected, source: 'dev'});
                await activatePremium();
            } else {
                Alert.alert(t('premium.paymentsUnavailable'));
            }
            return;
        }

        setProcessing(true);
        try {
            const result = await purchasePremium(selected, 'DARK');
            await syncPurchase({
                productId: selected,
                invoiceId: result.invoiceId,
                purchaseId: result.purchaseId,
                orderId: result.orderId,
                source: 'rustore',
                sandbox: result.sandbox,
            });
            await activatePremium();
        } catch (error) {
            if (error instanceof PurchaseCancelledError) {
                return;
            }
            if (error instanceof RuStoreMissingError) {
                Alert.alert(t('premium.rustoreMissingTitle'), t('premium.rustoreMissing'), [
                    {text: t('premium.cancel'), style: 'cancel'},
                    {
                        text: t('premium.installRuStore'),
                        onPress: () => {
                            promptInstallRuStore().catch(() => {});
                        },
                    },
                ]);
                return;
            }
            console.warn('[payments] purchase failed', error);
            const details = describePayError(error);
            Alert.alert(t('premium.purchaseError'), details ?? undefined);
        } finally {
            setProcessing(false);
        }
    };

    const handleRestore = async () => {
        if (processing) {
            return;
        }

        if (Platform.OS !== 'android') {
            setProcessing(true);
            try {
                const me = await refreshUser();
                if (me?.hasPremium) {
                    await setSetting('hasPremium', true);
                    Alert.alert(t('premium.restoreSuccess'), '', [
                        {
                            text: 'OK',
                            onPress: () => {
                                onActivated?.();
                                onClose();
                            },
                        },
                    ]);
                } else {
                    Alert.alert(t('premium.restoreNone'));
                }
            } catch {
                Alert.alert(t('premium.purchaseError'));
            } finally {
                setProcessing(false);
            }
            return;
        }

        if (gate.kind === 'managed' && gate.provider !== 'rustore') {
            setProcessing(true);
            try {
                const me = await refreshUser();
                if (me?.hasPremium) {
                    await setSetting('hasPremium', true);
                    Alert.alert(t('premium.restoreSuccess'));
                } else {
                    Alert.alert(t('premium.restoreNone'));
                }
            } catch {
                Alert.alert(t('premium.purchaseError'));
            } finally {
                setProcessing(false);
            }
            return;
        }

        if (!isPaymentsSupported()) {
            Alert.alert(t('premium.paymentsUnavailable'));
            return;
        }

        setProcessing(true);
        try {
            const status = await readStorePremiumStatus();
            if (status.active && status.productId) {
                await syncPurchase({
                    productId: status.productId,
                    source: 'restore',
                    expiresAt: status.expiresAt,
                });
                await setSetting('hasPremium', true);
                Alert.alert(t('premium.restoreSuccess'), '', [
                    {
                        text: 'OK',
                        onPress: () => {
                            onActivated?.();
                            onClose();
                        },
                    },
                ]);
            } else {
                Alert.alert(t('premium.restoreNone'));
            }
        } catch (error) {
            console.warn('[payments] restore failed', error);
            const details = describePayError(error);
            Alert.alert(t('premium.purchaseError'), details ?? undefined);
        } finally {
            setProcessing(false);
        }
    };

    return (
        <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
            <View style={styles.overlay}>
                <View
                    style={[
                        styles.content,
                        {backgroundColor: colors.tabBar, borderColor: hexAlpha(colors.accent, 0.3)},
                    ]}
                >
                    <ScrollView contentContainerStyle={styles.scrollContent}>
                        <Text style={[styles.title, {color: colors.gold}]}>{t('premium.title')}</Text>
                        {gate.kind === 'managed' ? (
                            <>
                                <Text style={[styles.benefitsText, {color: colors.text}]}>
                                    {managedCopy(gate).body}
                                </Text>
                                <TouchableOpacity style={styles.cancelButton} onPress={onClose}>
                                    <Text style={[styles.cancelButtonText, {color: colors.text}]}>
                                        {t('premium.cancel')}
                                    </Text>
                                </TouchableOpacity>
                            </>
                        ) : (
                            <>
                        <Text style={[styles.benefitsText, {color: colors.text}]}>{t('premium.benefits')}</Text>
                        <Text style={[styles.selectLabel, {color: colors.gold}]}>{t('premium.selectPlan')}</Text>

                        {PREMIUM_PRODUCT_IDS.map((id) => {
                            const copy = PLAN_COPY[id];
                            const active = selected === id;
                            const titleFromStore = catalog.find((item) => item.productId === id)?.title;
                            return (
                                <TouchableOpacity
                                    key={id}
                                    style={[
                                        styles.plan,
                                        {borderColor: hexAlpha(colors.gold, 0.2)},
                                        active && {
                                            borderColor: colors.accent,
                                            backgroundColor: hexAlpha(colors.accent, 0.16),
                                        },
                                    ]}
                                    onPress={() => setSelected(id)}
                                    disabled={processing}
                                >
                                    <View style={styles.planHeader}>
                                        <Text style={[styles.planName, {color: colors.text}]}>
                                            {titleFromStore || t(`premium.${copy.name}`)}
                                        </Text>
                                        <Text style={[styles.planPrice, {color: colors.accent}]}>{priceById[id]}</Text>
                                    </View>
                                    <Text style={[styles.planPeriod, {color: colors.gold}]}>
                                        {t(`premium.${copy.period}`)}
                                    </Text>
                                    <Text style={[styles.planHint, {color: colors.text}]}>
                                        {t(`premium.${copy.hint}`)}
                                    </Text>
                                </TouchableOpacity>
                            );
                        })}

                        <View style={[styles.featuresBox, {backgroundColor: hexAlpha(colors.accent, 0.1)}]}>
                            <Text style={[styles.featuresTitle, {color: colors.gold}]}>
                                {t('premium.features.title')}
                            </Text>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>✨</Text>
                                <Text style={[styles.featureText, {color: colors.text}]}>{t('premium.features.aiInterpretations')}</Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>🔮</Text>
                                <Text style={[styles.featureText, {color: colors.text}]}>
                                    {t('premium.features.deeperInsights')}
                                </Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>☀️</Text>
                                <Text style={[styles.featureText, {color: colors.text}]}>
                                    {t('premium.features.dailyCards')}
                                </Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>♾️</Text>
                                <Text style={[styles.featureText, {color: colors.text}]}>
                                    {t('premium.features.unlimitedReadings')}
                                </Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>🎯</Text>
                                <Text style={[styles.featureText, {color: colors.text}]}>{t('premium.features.prioritySupport')}</Text>
                            </View>
                        </View>

                        <TouchableOpacity
                            style={[styles.subscribeButton, {backgroundColor: colors.accent}]}
                            onPress={() => {
                                handleSubscribe().catch(() => {});
                            }}
                            disabled={processing}
                        >
                            {processing ? (
                                <ActivityIndicator color="#fff" />
                            ) : (
                                <Text style={styles.subscribeButtonText}>
                                    {t(Platform.OS === 'android' ? 'premium.subscribe' : 'premium.subscribeYooKassa')}
                                </Text>
                            )}
                        </TouchableOpacity>

                        <TouchableOpacity
                            style={styles.restoreButton}
                            onPress={() => {
                                handleRestore().catch(() => {});
                            }}
                            disabled={processing}
                        >
                            <Text style={[styles.restoreButtonText, {color: colors.accent}]}>{t('premium.restore')}</Text>
                        </TouchableOpacity>

                        <TouchableOpacity style={styles.cancelButton} onPress={onClose}>
                            <Text style={[styles.cancelButtonText, {color: colors.text}]}>{t('premium.cancel')}</Text>
                        </TouchableOpacity>

                        <Text style={[styles.disclaimer, {color: colors.muted}]}>
                            {t(Platform.OS === 'android' ? 'premium.disclaimer' : 'premium.disclaimerIos')}
                        </Text>
                            </>
                        )}
                    </ScrollView>
                </View>
            </View>
        </Modal>
    );
};

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.8)',
        justifyContent: 'flex-end',
    },
    content: {
        backgroundColor: '#0c0a14',
        borderTopLeftRadius: 30,
        borderTopRightRadius: 30,
        maxHeight: '90%',
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.3)',
    },
    scrollContent: {
        padding: 24,
        gap: 16,
    },
    title: {
        color: '#f4d386',
        fontSize: 28,
        fontWeight: '700',
        textAlign: 'center',
    },
    selectLabel: {
        color: '#f4d386',
        fontSize: 16,
        fontWeight: '600',
        marginTop: 4,
    },
    plan: {
        backgroundColor: 'rgba(247,244,234,0.06)',
        borderRadius: 18,
        padding: 16,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.2)',
        gap: 4,
    },
    planActive: {
        borderColor: '#6c5ce7',
        backgroundColor: 'rgba(108,92,231,0.16)',
    },
    planHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: 12,
    },
    planName: {
        color: '#f7f4ea',
        fontSize: 16,
        fontWeight: '700',
        flex: 1,
    },
    planPrice: {
        color: '#6c5ce7',
        fontSize: 18,
        fontWeight: '700',
    },
    planPeriod: {
        color: '#f4d386',
        fontSize: 13,
        fontWeight: '600',
    },
    planHint: {
        color: '#f7f4ea',
        opacity: 0.7,
        fontSize: 13,
        lineHeight: 18,
    },
    benefitsText: {
        color: '#f7f4ea',
        fontSize: 16,
        lineHeight: 24,
        textAlign: 'center',
    },
    featuresBox: {
        backgroundColor: 'rgba(108,92,231,0.1)',
        borderRadius: 20,
        padding: 20,
        gap: 16,
    },
    featuresTitle: {
        color: '#f4d386',
        fontSize: 18,
        fontWeight: '600',
        marginBottom: 8,
    },
    feature: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 12,
    },
    featureBullet: {
        fontSize: 24,
    },
    featureText: {
        color: '#f7f4ea',
        fontSize: 16,
        flex: 1,
    },
    subscribeButton: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 18,
        borderRadius: 20,
        alignItems: 'center',
        marginTop: 8,
    },
    subscribeButtonText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: '700',
    },
    restoreButton: {
        paddingVertical: 12,
        alignItems: 'center',
    },
    restoreButtonText: {
        color: '#6c5ce7',
        fontSize: 15,
        fontWeight: '600',
    },
    cancelButton: {
        paddingVertical: 14,
        alignItems: 'center',
    },
    cancelButtonText: {
        color: '#f7f4ea',
        fontSize: 16,
        opacity: 0.7,
    },
    disclaimer: {
        color: '#f7f4ea',
        opacity: 0.5,
        fontSize: 12,
        textAlign: 'center',
        fontStyle: 'italic',
    },
});
