import AsyncStorage from '@react-native-async-storage/async-storage';

const STORAGE_KEY = 'tarot.dailyUsage.v1';

export const LOCAL_FREE_DAILY_CARDS = 1;
export const LOCAL_PREMIUM_DAILY_CARDS = 20;
export const LOCAL_FREE_AD_UNLOCKS = 5;

export type LocalDailyUsage = {
    date: string;
    cards: number;
    ads: number;
};

export const localDateKey = (now = new Date()): string => {
    const y = now.getFullYear();
    const m = String(now.getMonth() + 1).padStart(2, '0');
    const d = String(now.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
};

export const emptyLocalUsage = (now = new Date()): LocalDailyUsage => ({
    date: localDateKey(now),
    cards: 0,
    ads: 0,
});

export const loadLocalDailyUsage = async (): Promise<LocalDailyUsage> => {
    const today = localDateKey();
    try {
        const raw = await AsyncStorage.getItem(STORAGE_KEY);
        if (!raw) {
            return emptyLocalUsage();
        }
        const parsed = JSON.parse(raw) as Partial<LocalDailyUsage>;
        if (parsed.date !== today) {
            return emptyLocalUsage();
        }
        return {
            date: today,
            cards: Number(parsed.cards) || 0,
            ads: Number(parsed.ads) || 0,
        };
    } catch {
        return emptyLocalUsage();
    }
};

const persist = async (row: LocalDailyUsage): Promise<void> => {
    await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(row));
};

export const incrementLocalDailyCard = async (hasPremium: boolean): Promise<LocalDailyUsage> => {
    const row = await loadLocalDailyUsage();
    const cap = hasPremium ? LOCAL_PREMIUM_DAILY_CARDS : LOCAL_FREE_DAILY_CARDS;
    if (row.cards >= cap) {
        throw new Error('quota_exceeded');
    }
    const next = {...row, cards: row.cards + 1};
    await persist(next);
    return next;
};

export const incrementLocalAdCard = async (): Promise<LocalDailyUsage> => {
    const row = await loadLocalDailyUsage();
    if (row.ads >= LOCAL_FREE_AD_UNLOCKS || row.cards >= LOCAL_FREE_DAILY_CARDS + LOCAL_FREE_AD_UNLOCKS) {
        throw new Error('quota_exceeded');
    }
    const next = {...row, cards: row.cards + 1, ads: row.ads + 1};
    await persist(next);
    return next;
};
