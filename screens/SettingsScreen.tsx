import React, {useState} from 'react';
import {
    ScrollView,
    StyleSheet,
    Switch,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useTranslation} from 'react-i18next';
import {useSettings} from '../providers/SettingsProvider';
import type {LanguagePreference, ThemePreference} from '../entities';
import {PremiumModal} from '../components/PremiumModal';

const LANGUAGES: LanguagePreference[] = ['en', 'ru', 'th', 'zh'];
const THEMES: ThemePreference[] = ['system', 'light', 'dark'];

export const SettingsScreen = () => {
    const {settings, setSetting} = useSettings();
    const {t} = useTranslation();
    const [showPremiumModal, setShowPremiumModal] = useState(false);

    const toggle = (key: 'disableAnimations' | 'disableSounds' | 'showMysticMode') => {
        setSetting(key, !settings[key]).catch(() => {});
    };

    const adjustReversedChance = (delta: number) => {
        const next = Math.min(1, Math.max(0, settings.reversedChance + delta));
        setSetting('reversedChance', parseFloat(next.toFixed(2))).catch(() => {});
    };

    return (
        <SafeAreaView style={styles.safe}>
            <ScrollView contentContainerStyle={styles.container}>
                <Text style={styles.sectionTitle}>{t('settings.premiumSection')}</Text>
                <View style={styles.premiumCard}>
                    <View style={styles.premiumHeader}>
                        <View>
                            <Text style={styles.premiumTitle}>{t('premium.title')}</Text>
                            <Text style={styles.premiumStatus}>
                                {settings.hasPremium
                                    ? t('premium.status.active')
                                    : t('premium.status.inactive')}
                            </Text>
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
                            <Text style={styles.premiumDescription}>{t('premium.benefits')}</Text>
                            <TouchableOpacity
                                style={styles.premiumButton}
                                onPress={() => setShowPremiumModal(true)}
                            >
                                <Text style={styles.premiumButtonText}>{t('premium.subscribe')}</Text>
                            </TouchableOpacity>
                        </>
                    ) : (
                        <TouchableOpacity
                            style={styles.manageButton}
<<<<<<< Current (Your changes)
                            onPress={() => setSetting('hasPremium', false).catch(() => {})}
=======
                            onPress={() => setSetting('hasPremium', false)}
>>>>>>> Incoming (Background Agent changes)
                        >
                            <Text style={styles.manageButtonText}>{t('premium.manage')}</Text>
                        </TouchableOpacity>
                    )}
                </View>

                <Text style={styles.sectionTitle}>{t('settings.languageTitle')}</Text>
                <View style={styles.row}>
                    {LANGUAGES.map((language) => (
                        <TouchableOpacity
                            key={language}
                            style={[styles.option, settings.language === language && styles.optionActive]}
                            onPress={() => setSetting('language', language).catch(() => {})}
                        >
                            <Text
                                style={[styles.optionText, settings.language === language && styles.optionTextActive]}
                            >
                                {t(`settings.language.${language}`)}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>

                <Text style={styles.sectionTitle}>{t('settings.themeTitle')}</Text>
                <View style={styles.row}>
                    {THEMES.map((theme) => (
                        <TouchableOpacity
                            key={theme}
                            style={[styles.option, settings.theme === theme && styles.optionActive]}
                            onPress={() => setSetting('theme', theme).catch(() => {})}
                        >
                            <Text
                                style={[styles.optionText, settings.theme === theme && styles.optionTextActive]}
                            >
                                {t(`settings.theme.${theme}`)}
                            </Text>
                        </TouchableOpacity>
                    ))}
                </View>

                <Text style={styles.sectionTitle}>{t('settings.behaviorTitle')}</Text>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={styles.toggleTitle}>{t('settings.animations')}</Text>
                        <Text style={styles.toggleSubtitle}>{t('settings.animationsDescription')}</Text>
                    </View>
                    <Switch
                        value={!settings.disableAnimations}
                        onValueChange={(value) => setSetting('disableAnimations', !value)}
                    />
                </View>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={styles.toggleTitle}>{t('settings.sounds')}</Text>
                        <Text style={styles.toggleSubtitle}>{t('settings.soundsDescription')}</Text>
                    </View>
                    <Switch
                        value={!settings.disableSounds}
                        onValueChange={(value) => setSetting('disableSounds', !value)}
                    />
                </View>
                <View style={styles.toggleRow}>
                    <View>
                        <Text style={styles.toggleTitle}>{t('settings.mysticMode')}</Text>
                        <Text style={styles.toggleSubtitle}>{t('settings.mysticModeDescription')}</Text>
                    </View>
                    <Switch
                        value={settings.showMysticMode}
                        onValueChange={() => toggle('showMysticMode')}
                    />
                </View>

                <Text style={styles.sectionTitle}>{t('settings.reversedChanceTitle')}</Text>
                <View style={styles.reversedRow}>
                    <TouchableOpacity style={styles.stepper} onPress={() => adjustReversedChance(-0.05)}>
                        <Text style={styles.stepperText}>−</Text>
                    </TouchableOpacity>
                    <Text style={styles.reversedValue}>{Math.round(settings.reversedChance * 100)}%</Text>
                    <TouchableOpacity style={styles.stepper} onPress={() => adjustReversedChance(0.05)}>
                        <Text style={styles.stepperText}>+</Text>
                    </TouchableOpacity>
                </View>
                <Text style={styles.reversedHint}>{t('settings.reversedHint')}</Text>
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
});
