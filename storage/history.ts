import AsyncStorage from '@react-native-async-storage/async-storage';
import type { TarotCard } from '../App';

const HISTORY_KEY = 'TAROT_HISTORY';

export async function saveSpread(cards: TarotCard[]) {
    const history = await getHistory();
    const entry = {
        id: Date.now().toString(),
        timestamp: new Date().toISOString(),
        cards,
    };
    const updated = [entry, ...history];
    await AsyncStorage.setItem(HISTORY_KEY, JSON.stringify(updated));
}

export async function getHistory() {
    const raw = await AsyncStorage.getItem(HISTORY_KEY);
    if (!raw) return [];
    try {
        return JSON.parse(raw);
    } catch {
        return [];
    }
}

export async function clearHistory() {
    await AsyncStorage.removeItem(HISTORY_KEY);
}

