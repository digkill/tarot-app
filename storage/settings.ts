import AsyncStorage from '@react-native-async-storage/async-storage';
import type {Settings} from '../entities';
import {DEFAULT_SETTINGS} from '../entities';

const STORAGE_KEY = 'tarot.settings';

const parseSettings = (raw: string | null): Settings => {
    if (!raw) return DEFAULT_SETTINGS;
    try {
        const parsed = JSON.parse(raw);
        return {...DEFAULT_SETTINGS, ...parsed};
    } catch (error) {
        console.warn('[settings] failed to parse', error);
        return DEFAULT_SETTINGS;
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
