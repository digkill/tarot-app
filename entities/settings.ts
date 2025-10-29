import type {LanguagePreference, Settings, ThemePreference} from './tarot';

export const DEFAULT_LANGUAGE: LanguagePreference = 'en';

export const DEFAULT_THEME: ThemePreference = 'system';

export const DEFAULT_REVERSED_CHANCE = 0.3;

export const DEFAULT_SETTINGS: Settings = {
    language: DEFAULT_LANGUAGE,
    theme: DEFAULT_THEME,
    disableAnimations: false,
    disableSounds: false,
    reversedChance: DEFAULT_REVERSED_CHANCE,
    showMysticMode: true,
    hasPremium: false,
    dailyReminder: false,
    hasCompletedOnboarding: false,
    acceptedDisclaimer: false,
};
