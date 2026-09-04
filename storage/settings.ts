import AsyncStorage from '@react-native-async-storage/async-storage';
import type {Settings} from '../entities';
import {DEFAULT_SETTINGS} from '../entities';
import {isLanguagePreference, resolveDeviceLanguage} from '../utils/locale';

const STORAGE_KEY = 'tarot.settings';

const withDeviceLanguage = (settings: Settings): Settings => ({
    ...settings,
    language: resolveDeviceLanguage(),
    languageExplicit: false,
});

const parseSettings = (raw: string | null): Settings => {
    if (!raw) {
        return withDeviceLanguage(DEFAULT_SETTINGS);
    }
    try {
        const parsed = JSON.parse(raw) as Partial<Settings>;
        const merged: Settings = {...DEFAULT_SETTINGS, ...parsed};
        const storedLanguage = isLanguagePreference(parsed.language) ? parsed.language : null;

        if (parsed.languageExplicit === true && storedLanguage) {
            merged.language = storedLanguage;
            merged.languageExplicit = true;
            return merged;
        }

        return withDeviceLanguage(merged);
    } catch (error) {
        console.warn('[settings] failed to parse', error);
        return withDeviceLanguage(DEFAULT_SETTINGS);
    }
};

export const loadSettings = async (): Promise<Settings> => {
    const raw = await AsyncStorage.getItem(STORAGE_KEY);
    return parseSettings(raw);
};

export const saveSettings = async (settings: Settings) => {
    try {
        await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
    } catch (error) {
        console.warn('[settings] failed to save', error);
    }
};
