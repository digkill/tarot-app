import React, {useState} from 'react';
import {
    Alert,
    Platform,
    ScrollView,
    StyleSheet,
    Switch,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation} from '@react-navigation/native';
import type {CompositeNavigationProp} from '@react-navigation/native';
import type {BottomTabNavigationProp} from '@react-navigation/bottom-tabs';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {useSettings} from '../providers/SettingsProvider';
import {useAuth} from '../providers/AuthProvider';
import type {ThemePreference} from '../entities';
import {SUPPORTED_LANGUAGES} from '../utils/locale';
import {PremiumModal} from '../components/PremiumModal';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {openRuStoreSubscriptions, RuStoreMissingError} from '../features/payments';
import type {AppTabsParamList, RootStackParamList} from '../navigation/types';

type SettingsNav = CompositeNavigationProp<
    BottomTabNavigationProp<AppTabsParamList, 'Settings'>,
    NativeStackNavigationProp<RootStackParamList>
>;

const THEMES: ThemePreference[] = ['system', 'light', 'dark'];

export const SettingsScreen = () => {
    const {settings, setSetting} = useSettings();
    const {user, logout, deleteAccount} = useAuth();
    const navigation = useNavigation<SettingsNav>();
    const {t} = useTranslation();
    const colors = useAppColors();
    const [showPremiumModal, setShowPremiumModal] = useState(false);

    const toggle = (key: 'disableAnimations' | 'disableSounds' | 'showMysticMode') => {
        setSetting(key, !settings[key]).catch(() => {});
    };

    const adjustReversedChance = (delta: number) => {
        const next = Math.min(1, Math.max(0, settings.reversedChance + delta));
        setSetting('reversedChance', parseFloat(next.toFixed(2))).catch(() => {});
    };

    const confirmLogout = () => {
        Alert.alert(t('auth.logoutTitle'), t('auth.logoutDescription'), [
            {text: t('history.cancel'), style: 'cancel'},
            {
                text: t('auth.logoutCta'),
                onPress: () => {
                    logout().catch(() => {});
                },
            },
        ]);
    };

    const confirmDelete = () => {
        Alert.alert(t('auth.deleteTitle'), t('auth.deleteDescription'), [
            {text: t('history.cancel'), style: 'cancel'},
            {
                text: t('auth.deleteCta'),
                style: 'destructive',
                onPress: () => {
                    deleteAccount().catch(() => {
                        Alert.alert(t('auth.deleteTitle'), t('auth.errors.generic'));
                    });
                },
            },
        ]);
    };

    return (
        <SafeAreaView style={[styles.safe, {backgroundColor: colors.bg}]}>
            <ScrollView contentContainerStyle={styles.container}>
                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.accountSection')}</Text>
                <View
                    style={[
                        styles.accountCard,
                        {backgroundColor: hexAlpha(colors.gold, 0.08), borderColor: hexAlpha(colors.gold, 0.25)},
                    ]}
                >
                    <Text style={[styles.accountEmail, {color: colors.text}]}>{user?.email ?? ''}</Text>
                    <TouchableOpacity
                        style={styles.linkRow}
                        onPress={() => navigation.navigate('LegalDocument', {doc: 'terms'})}
                    >
                        <Text style={[styles.linkRowText, {color: colors.accent}]}>{t('legal.termsTitle')}</Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={styles.linkRow}
                        onPress={() => navigation.navigate('LegalDocument', {doc: 'privacy'})}
                    >
                        <Text style={[styles.linkRowText, {color: colors.accent}]}>{t('legal.privacyTitle')}</Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={[styles.manageButton, {borderColor: hexAlpha(colors.gold, 0.4)}]}
                        onPress={confirmLogout}
                    >
                        <Text style={[styles.manageButtonText, {color: colors.gold}]}>{t('auth.logoutCta')}</Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={[styles.deleteButton, {borderColor: hexAlpha(colors.danger, 0.45)}]}
                        onPress={confirmDelete}
                    >
                        <Text style={[styles.deleteButtonText, {color: colors.danger}]}>{t('auth.deleteCta')}</Text>
                    </TouchableOpacity>
                </View>

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.premiumSection')}</Text>
                <View
                    style={[
                        styles.premiumCard,
                        {backgroundColor: hexAlpha(colors.accent, 0.15), borderColor: hexAlpha(colors.accent, 0.3)},
                    ]}
                >
                    <View style={styles.premiumHeader}>
                        <View>
                            <Text style={[styles.premiumTitle, {color: colors.gold}]}>{t('premium.title')}</Text>
                            <Text style={[styles.premiumStatus, {color: colors.text}]}>
                                {settings.hasPremium
                                    ? t('premium.status.active')
                                    : t('premium.status.inactive')}
                            </Text>
                            {settings.hasPremium ? (
                                <Text style={[styles.premiumStatus, {color: colors.muted}]}>
                                    {user?.premiumExpiresAt
                                        ? t('premium.expiresOn', {
                                              date: new Date(user.premiumExpiresAt).toLocaleDateString(),
                                          })
                                        : t('premium.lifetimeAccess')}
                                </Text>
                            ) : null}
                            {settings.hasPremium && user?.premiumSource ? (
                                <Text style={[styles.premiumStatus, {color: colors.muted}]}>
                                    {t(`premium.source.${user.premiumSource}`, {
                                        defaultValue: user.premiumSource,
                                    })}
                                </Text>
                            ) : null}
                        </View>
                        <View
                            style={[
                                styles.statusBadge,
                                settings.hasPremium ? styles.statusActive : styles.statusInactive,
                            ]}
                        >
                            <Text style={styles.statusBadgeText}>
                                {settings.hasPremium ? '✓' : '✕'}
                            </Text>
                        </View>
                    </View>
                    {!settings.hasPremium ? (
                        <>
                            <Text style={[styles.premiumDescription, {color: colors.text}]}>{t('premium.benefits')}</Text>
                            <TouchableOpacity
                                style={[styles.premiumButton, {backgroundColor: colors.accent}]}
                                onPress={() => setShowPremiumModal(true)}
                            >
                                <Text style={styles.premiumButtonText}>{t('premium.subscribe')}</Text>
                            </TouchableOpacity>
                        </>
                    ) : (
                        <TouchableOpacity
                            style={[styles.manageButton, {borderColor: hexAlpha(colors.gold, 0.4)}]}
                            onPress={() => {
                                if (Platform.OS !== 'android' || user?.premiumSource === 'yookassa') {
                                    setShowPremiumModal(true);
                                    return;
                                }
                                openRuStoreSubscriptions().catch((error) => {
                                    if (error instanceof RuStoreMissingError) {
                                        Alert.alert(t('premium.rustoreMissingTitle'), t('premium.rustoreMissing'));
                                        return;
                                    }
                                    Alert.alert(t('premium.purchaseError'));
                                });
                            }}
                        >
                            <Text style={[styles.manageButtonText, {color: colors.gold}]}>{t('premium.manage')}</Text>
                        </TouchableOpacity>
                    )}
                </View>

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.languageTitle')}</Text>
                <View style={styles.row}>
                    {SUPPORTED_LANGUAGES.map((language) => (
                        <TouchableOpacity
                            key={language}
                            style={[
                                styles.option,
                                {borderColor: hexAlpha(colors.gold, 0.3), backgroundColor: hexAlpha(colors.gold, 0.12)},
                                settings.language === language && {
                                    backgroundColor: colors.accent,
                                    borderColor: colors.accent,
                                },
                            ]}
                            onPress={() => setSetting('language', language).catch(() => {})}
                        >
                            <Text
                                style={[
                                    styles.optionText,
                                    {color: colors.text},
                                    settings.language === language && styles.optionTextActive,
                                ]}
                            >
                                {t(`settings.language.${language}`)}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.themeTitle')}</Text>
                <View style={styles.row}>
                    {THEMES.map((theme) => (
                        <TouchableOpacity
                            key={theme}
                            style={[
                                styles.option,
                                {borderColor: hexAlpha(colors.gold, 0.3), backgroundColor: hexAlpha(colors.gold, 0.12)},
                                settings.theme === theme && {
                                    backgroundColor: colors.accent,
                                    borderColor: colors.accent,
                                },
                            ]}
                            onPress={() => setSetting('theme', theme).catch(() => {})}
                        >
                            <Text
                                style={[
                                    styles.optionText,
                                    {color: colors.text},
                                    settings.theme === theme && styles.optionTextActive,
                                ]}
                            >
                                {t(`settings.theme.${theme}`)}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.behaviorTitle')}</Text>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={[styles.toggleTitle, {color: colors.text}]}>{t('settings.animations')}</Text>
                        <Text style={[styles.toggleSubtitle, {color: colors.muted}]}>
                            {t('settings.animationsDescription')}
                        </Text>
                    </View>
                    <Switch
                        value={!settings.disableAnimations}
                        onValueChange={(value) => setSetting('disableAnimations', !value)}
                        trackColor={{false: colors.panel, true: colors.accent}}
                    />
                </View>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={[styles.toggleTitle, {color: colors.text}]}>{t('settings.sounds')}</Text>
                        <Text style={[styles.toggleSubtitle, {color: colors.muted}]}>
                            {t('settings.soundsDescription')}
                        </Text>
                    </View>
                    <Switch
                        value={!settings.disableSounds}
                        onValueChange={(value) => setSetting('disableSounds', !value)}
                        trackColor={{false: colors.panel, true: colors.accent}}
                    />
                </View>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={[styles.toggleTitle, {color: colors.text}]}>{t('settings.mysticMode')}</Text>
                        <Text style={[styles.toggleSubtitle, {color: colors.muted}]}>
                            {t('settings.mysticModeDescription')}
                        </Text>
                    </View>
                    <Switch
                        value={settings.showMysticMode}
                        onValueChange={() => toggle('showMysticMode')}
                        trackColor={{false: colors.panel, true: colors.accent}}
                    />
                </View>

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('settings.reversedChanceTitle')}</Text>
                <View style={styles.reversedRow}>
                    <TouchableOpacity
                        style={[styles.stepper, {backgroundColor: hexAlpha(colors.accent, 0.15)}]}
                        onPress={() => adjustReversedChance(-0.05)}
                    >
                        <Text style={[styles.stepperText, {color: colors.text}]}>−</Text>
                    </TouchableOpacity>
                    <Text style={[styles.reversedValue, {color: colors.gold}]}>
                        {Math.round(settings.reversedChance * 100)}%
                    </Text>
                    <TouchableOpacity
                        style={[styles.stepper, {backgroundColor: hexAlpha(colors.accent, 0.15)}]}
                        onPress={() => adjustReversedChance(0.05)}
                    >
                        <Text style={[styles.stepperText, {color: colors.text}]}>+</Text>
                    </TouchableOpacity>
                </View>
                <Text style={[styles.reversedHint, {color: colors.muted}]}>{t('settings.reversedHint')}</Text>
            </ScrollView>

            <PremiumModal visible={showPremiumModal} onClose={() => setShowPremiumModal(false)} />
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    container: {
        padding: 20,
        paddingBottom: 60,
        gap: 20,
    },
    sectionTitle: {
        color: '#f4d386',
        fontSize: 18,
        fontWeight: '600',
    },
    row: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 10,
    },
    option: {
        paddingHorizontal: 16,
        paddingVertical: 10,
        borderRadius: 18,
        backgroundColor: 'rgba(244,211,134,0.12)',
        borderColor: 'rgba(244,211,134,0.3)',
        borderWidth: 1,
    },
    optionActive: {
        backgroundColor: '#6c5ce7',
        borderColor: '#6c5ce7',
    },
    optionText: {
        color: '#f7f4ea',
        fontWeight: '500',
    },
    optionTextActive: {
        color: '#fff',
    },
    toggleRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingVertical: 12,
        borderBottomWidth: StyleSheet.hairlineWidth,
        borderColor: 'rgba(255,255,255,0.1)',
    },
    toggleTitle: {
        color: '#f7f4ea',
        fontWeight: '600',
        marginBottom: 4,
    },
    toggleSubtitle: {
        color: '#f7f4ea',
        opacity: 0.65,
        maxWidth: 240,
    },
    reversedRow: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 20,
        marginTop: 8,
    },
    reversedValue: {
        color: '#f4d386',
        fontSize: 24,
        fontWeight: '700',
    },
    reversedHint: {
        color: '#f7f4ea',
        opacity: 0.6,
        fontSize: 13,
    },
    stepper: {
        width: 44,
        height: 44,
        borderRadius: 22,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'rgba(108,92,231,0.15)',
    },
    stepperText: {
        color: '#f7f4ea',
        fontSize: 22,
        fontWeight: '700',
    },
    premiumCard: {
        backgroundColor: 'rgba(108,92,231,0.15)',
        borderRadius: 20,
        padding: 20,
        borderWidth: 2,
        borderColor: 'rgba(108,92,231,0.3)',
        gap: 16,
    },
    premiumHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    premiumTitle: {
        color: '#f4d386',
        fontSize: 20,
        fontWeight: '700',
    },
    premiumStatus: {
        color: '#f7f4ea',
        fontSize: 14,
        marginTop: 4,
        opacity: 0.8,
    },
    statusBadge: {
        width: 40,
        height: 40,
        borderRadius: 20,
        alignItems: 'center',
        justifyContent: 'center',
    },
    statusActive: {
        backgroundColor: '#51cf66',
    },
    statusInactive: {
        backgroundColor: 'rgba(255,255,255,0.2)',
    },
    statusBadgeText: {
        color: '#fff',
        fontSize: 20,
        fontWeight: '700',
    },
    premiumDescription: {
        color: '#f7f4ea',
        lineHeight: 20,
        opacity: 0.9,
    },
    premiumButton: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 14,
        borderRadius: 16,
        alignItems: 'center',
    },
    premiumButtonText: {
        color: '#fff',
        fontWeight: '700',
        fontSize: 16,
    },
    manageButton: {
        paddingVertical: 12,
        borderRadius: 14,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.4)',
        alignItems: 'center',
    },
    manageButtonText: {
        color: '#f4d386',
        fontWeight: '600',
    },
    accountCard: {
        backgroundColor: 'rgba(244,211,134,0.08)',
        borderRadius: 20,
        padding: 20,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        gap: 12,
    },
    accountEmail: {
        color: '#f7f4ea',
        fontSize: 16,
        fontWeight: '600',
    },
    linkRow: {
        paddingVertical: 8,
    },
    linkRowText: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
    deleteButton: {
        paddingVertical: 12,
        borderRadius: 14,
        borderWidth: 1,
        borderColor: 'rgba(255,107,107,0.45)',
        alignItems: 'center',
    },
    deleteButtonText: {
        color: '#ff6b6b',
        fontWeight: '600',
    },
});
