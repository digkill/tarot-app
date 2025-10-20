import React from 'react';
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

const LANGUAGES: LanguagePreference[] = ['en', 'ru', 'th'];
const THEMES: ThemePreference[] = ['system', 'light', 'dark'];

export const SettingsScreen = () => {
    const {settings, setSetting} = useSettings();
    const {t} = useTranslation();

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
                <Text style={styles.sectionTitle}>{t('settings.languageTitle')}</Text>
                <View style={styles.row}>
                    {LANGUAGES.map((language) => (
                        <TouchableOpacity
                            key={language}
                            style={[styles.option, settings.language === language && styles.optionActive]}
                            onPress={() => setSetting('language', language)}
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
                            onPress={() => setSetting('theme', theme)}
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
});
