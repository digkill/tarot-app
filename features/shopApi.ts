import {apiRequest, getApiBaseUrl} from './apiClient';
import type {AppColors} from '../theme/appColors';

export type ShopDeck = {
    slug: string;
    titles: Record<string, string>;
    descriptions: Record<string, string>;
    theme: AppColors;
    priceKop: number;
    originalPriceKop?: number;
    productId: string;
    isFree: boolean;
    owned: boolean;
    cardCount: number;
    hasBack: boolean;
    coverUrl: string;
    backUrl: string;
    cards: Record<string, string>;
    bundled?: boolean;
};

export const absMediaUrl = (path: string): string => {
    if (!path) {
        return '';
    }
    if (path.startsWith('http://') || path.startsWith('https://')) {
        return path;
    }
    return `${getApiBaseUrl()}${path.startsWith('/') ? path : `/${path}`}`;
};

type ShopListResponse = {decks: ShopDeck[]};

export const fetchShopDecks = () => apiRequest<ShopListResponse>('/api/v1/shop/decks');

export const fetchMyDeckSlugs = () =>
    apiRequest<{slugs: string[]}>('/api/v1/me/decks');

export const localizeShopText = (
    map: Record<string, string> | undefined,
    language: string,
    fallback: string,
): string => {
    if (!map) {
        return fallback;
    }
    return map[language] || map.ru || map.en || Object.values(map)[0] || fallback;
};
