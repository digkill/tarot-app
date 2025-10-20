import AsyncStorage from '@react-native-async-storage/async-storage';
import type {Reading, ReadingCard} from '../entities';

const STORAGE_KEY = 'tarot.readings';

type PersistedReading = Reading;

const withDefault = (value: PersistedReading[] | null): PersistedReading[] =>
    Array.isArray(value) ? value : [];

export const loadReadings = async (): Promise<Reading[]> => {
    try {
        const raw = await AsyncStorage.getItem(STORAGE_KEY);
        if (!raw) return [];
        const parsed = JSON.parse(raw) as PersistedReading[];
        return withDefault(parsed);
    } catch (error) {
        console.warn('[readings] failed to load', error);
        return [];
    }
};

const persist = async (readings: Reading[]) => {
    try {
        await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(readings));
    } catch (error) {
        console.warn('[readings] failed to save', error);
    }
};

const identity = () => `${Date.now().toString(36)}-${Math.random().toString(16).slice(2, 8)}`;

export const addReading = async (reading: Omit<Reading, 'id' | 'drawnAt'>): Promise<Reading> => {
    const existing = await loadReadings();
    const entry: Reading = {
        ...reading,
        id: identity(),
        drawnAt: Date.now(),
    };
    await persist([entry, ...existing]);
    return entry;
};

export const updateReading = async (id: string, updates: Partial<Reading>) => {
    const existing = await loadReadings();
    const updated = existing.map((item) => (item.id === id ? {...item, ...updates} : item));
    await persist(updated);
};

export const removeReading = async (id: string) => {
    const existing = await loadReadings();
    await persist(existing.filter((item) => item.id !== id));
};

export const clearReadings = async () => {
    await AsyncStorage.removeItem(STORAGE_KEY);
};

export const appendCardToReading = async (id: string, card: ReadingCard) => {
    const existing = await loadReadings();
    const updated = existing.map((item) =>
        item.id === id ? {...item, items: [...item.items, card]} : item,
    );
    await persist(updated);
};

export const toggleFavorite = async (id: string): Promise<Reading | undefined> => {
    const existing = await loadReadings();
    let toggled: Reading | undefined;
    const updated = existing.map((item) => {
        if (item.id !== id) return item;
        toggled = {...item, favorite: !item.favorite};
        return toggled;
    });
    await persist(updated);
    return toggled;
};
