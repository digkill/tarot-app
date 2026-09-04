import AsyncStorage from '@react-native-async-storage/async-storage';
import type {AuthSession} from '../entities';

const STORAGE_KEY = 'tarot.session';

export const loadSession = async (): Promise<AuthSession | null> => {
    try {
        const raw = await AsyncStorage.getItem(STORAGE_KEY);
        if (!raw) {
            return null;
        }
        const parsed = JSON.parse(raw) as Partial<AuthSession>;
        if (!parsed.accessToken || !parsed.refreshToken || !parsed.user?.id || !parsed.user?.email) {
            return null;
        }
        return parsed as AuthSession;
    } catch (error) {
        console.warn('[session] failed to parse', error);
        return null;
    }
};

export const saveSession = async (session: AuthSession) => {
    try {
        await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } catch (error) {
        console.warn('[session] failed to save', error);
    }
};

export const clearSession = async () => {
    try {
        await AsyncStorage.removeItem(STORAGE_KEY);
    } catch (error) {
        console.warn('[session] failed to clear', error);
    }
};
