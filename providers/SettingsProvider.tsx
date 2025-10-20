import React, {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react';
import type {ReactNode} from 'react';
import type {Settings} from '../entities';
import {DEFAULT_SETTINGS} from '../entities';
import {loadSettings, saveSettings} from '../storage/settings';

type SettingsContextValue = {
    settings: Settings;
    loading: boolean;
    updateSettings: (values: Partial<Settings>) => Promise<void>;
    setSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => Promise<void>;
};

const SettingsContext = createContext<SettingsContextValue | null>(null);

export const SettingsProvider = ({children}: {children: ReactNode}) => {
    const [settings, setSettings] = useState<Settings>(DEFAULT_SETTINGS);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        loadSettings()
            .then((loaded) => setSettings(loaded))
            .finally(() => setLoading(false));
    }, []);

    const persist = useCallback(async (next: Settings) => {
        setSettings(next);
        await saveSettings(next);
    }, []);

    const updateSettings = useCallback(
        async (values: Partial<Settings>) => {
            const next = {...settings, ...values};
            await persist(next);
        },
        [persist, settings],
    );

    const setSetting = useCallback(
        async (key: keyof Settings, value: Settings[keyof Settings]) => {
            await updateSettings({[key]: value} as Partial<Settings>);
        },
        [updateSettings],
    );

    const value = useMemo(
        () => ({
            settings,
            loading,
            updateSettings,
            setSetting,
        }),
        [loading, setSetting, settings, updateSettings],
    );

    return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
};

export const useSettings = () => {
    const context = useContext(SettingsContext);
    if (!context) {
        throw new Error('useSettings must be used within SettingsProvider');
    }
    return context;
};
