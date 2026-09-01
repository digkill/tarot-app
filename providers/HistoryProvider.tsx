import React, {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react';
import type {ReactNode} from 'react';
import type {Reading} from '../entities';
import {getReadingSummary, hasBrokenSummary} from '../features/interpretation';
import {
    addReading as addReadingStorage,
    clearReadings,
    loadReadings,
    removeReading,
    toggleFavorite,
    updateReading as updateReadingStorage,
} from '../storage/readings';
import {loadSettings} from '../storage/settings';

type HistoryContextValue = {
    readings: Reading[];
    loading: boolean;
    addReading: (reading: Omit<Reading, 'id' | 'drawnAt'>) => Promise<Reading>;
    updateReading: (id: string, updates: Partial<Reading>) => Promise<void>;
    deleteReading: (id: string) => Promise<void>;
    clearAll: () => Promise<void>;
    toggleFavorite: (id: string) => Promise<void>;
    refresh: () => Promise<void>;
};

const HistoryContext = createContext<HistoryContextValue | null>(null);

export const HistoryProvider = ({children}: {children: ReactNode}) => {
    const [readings, setReadings] = useState<Reading[]>([]);
    const [loading, setLoading] = useState(true);

    const refresh = useCallback(async () => {
        setLoading(true);
        try {
            const [data, settings] = await Promise.all([loadReadings(), loadSettings()]);
            const migrated = await Promise.all(
                data.map(async (reading) => {
                    if (!hasBrokenSummary(reading.summaryText)) {
                        return reading;
                    }
                    const summaryText = getReadingSummary(reading, settings.language);
                    await updateReadingStorage(reading.id, {summaryText});
                    return {...reading, summaryText};
                }),
            );
            setReadings(migrated);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        refresh();
    }, [refresh]);

    const addReading = useCallback(
        async (reading: Omit<Reading, 'id' | 'drawnAt'>) => {
            const created = await addReadingStorage(reading);
            setReadings((prev) => [created, ...prev]);
            return created;
        },
        [],
    );

    const updateReading = useCallback(async (id: string, updates: Partial<Reading>) => {
        await updateReadingStorage(id, updates);
        setReadings((prev) => prev.map((item) => (item.id === id ? {...item, ...updates} : item)));
    }, []);

    const deleteReading = useCallback(async (id: string) => {
        await removeReading(id);
        setReadings((prev) => prev.filter((item) => item.id !== id));
    }, []);

    const clearAll = useCallback(async () => {
        await clearReadings();
        setReadings([]);
    }, []);

    const toggleFavoriteReading = useCallback(async (id: string) => {
        const updated = await toggleFavorite(id);
        if (updated) {
            setReadings((prev) => prev.map((item) => (item.id === id ? updated : item)));
        }
    }, []);

    const value = useMemo(
        () => ({
            readings,
            loading,
            addReading,
            updateReading,
            deleteReading,
            clearAll,
            toggleFavorite: toggleFavoriteReading,
            refresh,
        }),
        [addReading, clearAll, deleteReading, loading, readings, refresh, toggleFavoriteReading, updateReading],
    );

    return <HistoryContext.Provider value={value}>{children}</HistoryContext.Provider>;
};

export const useHistory = () => {
    const context = useContext(HistoryContext);
    if (!context) {
        throw new Error('useHistory must be used within HistoryProvider');
    }
    return context;
};
