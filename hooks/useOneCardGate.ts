import {useCallback, useState} from 'react';
import {Alert} from 'react-native';
import {useTranslation} from 'react-i18next';
import type {RootStackParamList} from '../navigation/types';
import {useHistory} from '../providers/HistoryProvider';
import {useSettings} from '../providers/SettingsProvider';
import {isApiError} from '../features/apiClient';
import {
    consumeOneCardSlot,
    findTodaysDailyCard,
    ONE_CARD_SPREAD_ID,
} from '../features/dailyCard';

export const useOneCardGate = (
    goReading: (params: RootStackParamList['Reading']) => void,
    goInterpretation: (readingId: string) => void,
    deckId: string,
) => {
    const {t} = useTranslation();
    const {readings} = useHistory();
    const {settings} = useSettings();
    const [limitOpen, setLimitOpen] = useState(false);
    const [busy, setBusy] = useState(false);

    const today = findTodaysDailyCard(readings);

    const openToday = useCallback(() => {
        if (!today) {
            return false;
        }
        goInterpretation(today.id);
        return true;
    }, [goInterpretation, today]);

    const goRead = useCallback(
        (kind: 'daily' | 'bonus') => {
            goReading({
                spreadId: ONE_CARD_SPREAD_ID,
                deckId,
                quotaGranted: true,
                readingKind: kind,
            });
        },
        [deckId, goReading],
    );

    const start = useCallback(
        async (preferExisting: boolean) => {
            if (preferExisting && openToday()) {
                return;
            }
            setBusy(true);
            try {
                const kind = await consumeOneCardSlot(Boolean(today), settings.hasPremium);
                goRead(kind);
            } catch (error) {
                if (isApiError(error) && error.code === 'quota_exceeded') {
                    if (preferExisting && openToday()) {
                        return;
                    }
                    setLimitOpen(true);
                    return;
                }
                Alert.alert(t('dailyCard.unlockError'));
            } finally {
                setBusy(false);
            }
        },
        [goRead, openToday, t, today, settings.hasPremium],
    );

    const onUnlocked = useCallback(
        async (result: {consumed: boolean; kind: 'daily' | 'bonus'}) => {
            setLimitOpen(false);
            if (result.consumed) {
                goRead(result.kind);
                return;
            }
            setBusy(true);
            try {
                const kind = await consumeOneCardSlot(Boolean(today), settings.hasPremium);
                goRead(kind);
            } catch {
                Alert.alert(t('dailyCard.quotaReached'));
            } finally {
                setBusy(false);
            }
        },
        [goRead, t, today, settings.hasPremium],
    );

    return {
        today,
        busy,
        limitOpen,
        setLimitOpen,
        start,
        openToday,
        onUnlocked,
        hasPremium: settings.hasPremium,
    };
};
