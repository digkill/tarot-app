import {apiRequest, deviceTimezone} from './apiClient';

export type QuotaBucket = {
    used: number;
    limit: number;
    remaining: number;
};

export type UsageSnapshot = {
    date: string;
    timezone: string;
    hasPremium: boolean;
    dailyCards: QuotaBucket;
    adUnlocks: QuotaBucket;
    interpretations: QuotaBucket;
};

export type AdSession = {
    adToken: string;
    minWatchSeconds: number;
    date: string;
};

const withTz = (path: string): string => {
    const tz = encodeURIComponent(deviceTimezone());
    return path.includes('?') ? `${path}&tz=${tz}` : `${path}?tz=${tz}`;
};

export const fetchUsage = () => apiRequest<UsageSnapshot>(withTz('/api/v1/usage'));

export const startAdSession = () =>
    apiRequest<AdSession>(withTz('/api/v1/usage/ad-session'), {method: 'POST', body: {}});

export const consumeDailyCard = (adToken?: string) =>
    apiRequest<UsageSnapshot>(withTz('/api/v1/usage/daily-card'), {
        method: 'POST',
        body: adToken ? {adToken} : {},
    });
