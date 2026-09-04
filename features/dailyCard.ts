import type {Reading} from '../entities';
import {ApiError, isApiError} from './apiClient';
import {consumeDailyCard, startAdSession, type AdSession} from './usageApi';
import {incrementLocalAdCard, incrementLocalDailyCard} from '../storage/dailyUsage';

export const ONE_CARD_SPREAD_ID = 'one-card';
export const LOCAL_AD_TOKEN = 'local-ad';

export const startOfLocalDay = (now = Date.now()): number => {
    const date = new Date(now);
    date.setHours(0, 0, 0, 0);
    return date.getTime();
};

export const findTodaysDailyCard = (readings: Reading[], now = Date.now()): Reading | undefined =>
    readings.find(
        (item) =>
            item.spreadId === ONE_CARD_SPREAD_ID &&
            item.kind !== 'bonus' &&
            item.drawnAt >= startOfLocalDay(now),
    );

export const isOneCardSpread = (spreadId: string): boolean => spreadId === ONE_CARD_SPREAD_ID;

export const isUsageUnavailable = (error: unknown): boolean => {
    if (!isApiError(error)) {
        return false;
    }
    if (error.code === 'network' || error.status === 404) {
        return true;
    }
    return error.code === 'http_error' && (error.status === 404 || error.status >= 500);
};

export const isQuotaExceeded = (error: unknown): boolean =>
    isApiError(error) && error.code === 'quota_exceeded';

const quotaError = (): ApiError => new ApiError('quota_exceeded', 'daily limit reached', 429);

export const consumeOneCardSlot = async (
    hasExistingDaily: boolean,
    hasPremium: boolean,
): Promise<'daily' | 'bonus'> => {
    try {
        await consumeDailyCard();
        return hasExistingDaily ? 'bonus' : 'daily';
    } catch (error) {
        if (isQuotaExceeded(error)) {
            throw error;
        }
        if (!isUsageUnavailable(error)) {
            throw error;
        }
        if (hasExistingDaily && !hasPremium) {
            throw quotaError();
        }
        try {
            await incrementLocalDailyCard(hasPremium);
        } catch {
            throw quotaError();
        }
        return hasExistingDaily ? 'bonus' : 'daily';
    }
};

export const consumeAdOneCard = async (adToken: string): Promise<'bonus'> => {
    if (adToken === LOCAL_AD_TOKEN) {
        try {
            await incrementLocalAdCard();
        } catch {
            throw quotaError();
        }
        return 'bonus';
    }
    try {
        await consumeDailyCard(adToken);
        return 'bonus';
    } catch (error) {
        if (isQuotaExceeded(error)) {
            throw error;
        }
        if (!isUsageUnavailable(error)) {
            throw error;
        }
        try {
            await incrementLocalAdCard();
        } catch {
            throw quotaError();
        }
        return 'bonus';
    }
};

export const beginAdSession = async (): Promise<AdSession> => {
    try {
        return await startAdSession();
    } catch (error) {
        if (isQuotaExceeded(error)) {
            throw error;
        }
        if (!isUsageUnavailable(error)) {
            throw error;
        }
        return {
            adToken: LOCAL_AD_TOKEN,
            minWatchSeconds: 15,
            date: new Date().toISOString().slice(0, 10),
        };
    }
};
