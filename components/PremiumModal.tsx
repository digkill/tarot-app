import React, {useState} from 'react';
import {
    Modal,
    View,
    Text,
    TouchableOpacity,
    StyleSheet,
    ScrollView,
    ActivityIndicator,
    Alert,
} from 'react-native';
import {useTranslation} from 'react-i18next';
import {useSettings} from '../providers/SettingsProvider';
import {
    describePayError,
    hasActivePremiumSubscription,
    isPaymentsSupported,
    purchasePremium,
    PurchaseCancelledError,
} from '../features/payments';

type PremiumModalProps = {
    visible: boolean;
    onClose: () => void;
};

export const PremiumModal = ({visible, onClose}: PremiumModalProps) => {
    const {t} = useTranslation();
    const {setSetting} = useSettings();
    const [processing, setProcessing] = useState(false);

    const activatePremium = async () => {
        await setSetting('hasPremium', true);
        Alert.alert(t('premium.purchaseSuccess'), '', [{text: 'OK', onPress: onClose}]);
    };

    const handleSubscribe = async () => {
        if (processing) return;

        if (!isPaymentsSupported()) {
            if (__DEV__) {
                // В dev-окружении (Expo Go, iOS-симулятор) активируем премиум без оплаты
                await activatePremium();
            } else {
                Alert.alert(t('premium.paymentsUnavailable'));
            }
            return;
        }

        setProcessing(true);
        try {
            await purchasePremium('DARK');
            await activatePremium();
        } catch (error) {
            if (!(error instanceof PurchaseCancelledError)) {
                console.warn('[payments] purchase failed', error);
                const details = describePayError(error);
                Alert.alert(t('premium.purchaseError'), details ?? undefined);
            }
        } finally {
            setProcessing(false);
        }
    };

    const handleRestore = async () => {
        if (processing) return;

        if (!isPaymentsSupported()) {
            Alert.alert(t('premium.paymentsUnavailable'));
            return;
        }

        setProcessing(true);
        try {
            if (await hasActivePremiumSubscription()) {
                await setSetting('hasPremium', true);
                Alert.alert(t('premium.restoreSuccess'), '', [{text: 'OK', onPress: onClose}]);
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
                <View style={styles.content}>
                    <ScrollView contentContainerStyle={styles.scrollContent}>
                        <Text style={styles.title}>{t('premium.title')}</Text>
                        <View style={styles.priceBox}>
                            <Text style={styles.price}>{t('premium.price')}</Text>
                        </View>

                        <Text style={styles.benefitsText}>{t('premium.benefits')}</Text>

                        <View style={styles.featuresBox}>
                            <Text style={styles.featuresTitle}>{t('premium.features.title')}</Text>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>✨</Text>
                                <Text style={styles.featureText}>{t('premium.features.aiInterpretations')}</Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>🔮</Text>
                                <Text style={styles.featureText}>{t('premium.features.deeperInsights')}</Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>♾️</Text>
                                <Text style={styles.featureText}>{t('premium.features.unlimitedReadings')}</Text>
                            </View>
                            <View style={styles.feature}>
                                <Text style={styles.featureBullet}>🎯</Text>
                                <Text style={styles.featureText}>{t('premium.features.prioritySupport')}</Text>
                            </View>
                        </View>

                        <TouchableOpacity
                            style={styles.subscribeButton}
                            onPress={handleSubscribe}
                            disabled={processing}
                        >
                            {processing ? (
                                <ActivityIndicator color="#fff" />
                            ) : (
                                <Text style={styles.subscribeButtonText}>{t('premium.subscribe')}</Text>
                            )}
                        </TouchableOpacity>

                        <TouchableOpacity
                            style={styles.restoreButton}
                            onPress={handleRestore}
                            disabled={processing}
                        >
                            <Text style={styles.restoreButtonText}>{t('premium.restore')}</Text>
                        </TouchableOpacity>

                        <TouchableOpacity style={styles.cancelButton} onPress={onClose}>
                            <Text style={styles.cancelButtonText}>{t('premium.cancel')}</Text>
                        </TouchableOpacity>

                        <Text style={styles.disclaimer}>{t('premium.disclaimer')}</Text>
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
        gap: 20,
    },
    title: {
        color: '#f4d386',
        fontSize: 28,
        fontWeight: '700',
        textAlign: 'center',
    },
    priceBox: {
        alignItems: 'center',
        paddingVertical: 16,
    },
    price: {
        color: '#6c5ce7',
        fontSize: 36,
        fontWeight: '700',
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
