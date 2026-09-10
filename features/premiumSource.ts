import {Platform} from 'react-native';

export type CheckoutProvider = 'rustore' | 'yookassa';
export type ManagedProvider = CheckoutProvider | 'support';

export type PremiumGate =
    | {kind: 'buy'; provider: CheckoutProvider}
    | {kind: 'managed'; provider: ManagedProvider; local: boolean};

const normalize = (source?: string | null): string => (source ?? '').trim().toLowerCase();

export const localCheckoutProvider = (): CheckoutProvider =>
    Platform.OS === 'android' ? 'rustore' : 'yookassa';

export const isYooKassaSource = (source?: string | null): boolean => normalize(source) === 'yookassa';

export const isSupportSource = (source?: string | null): boolean => {
    const value = normalize(source);
    return value === 'admin' || value === 'promo' || value === 'dev';
};

export const isRuStoreSource = (source?: string | null): boolean => {
    const value = normalize(source);
    return value === 'rustore' || value === '';
};

export const premiumGate = (hasPremium: boolean, source?: string | null): PremiumGate => {
    if (!hasPremium) {
        return {kind: 'buy', provider: localCheckoutProvider()};
    }
    const local = localCheckoutProvider();
    if (isSupportSource(source)) {
        return {kind: 'managed', provider: 'support', local: false};
    }
    if (isYooKassaSource(source)) {
        return {kind: 'managed', provider: 'yookassa', local: local === 'yookassa'};
    }
    return {kind: 'managed', provider: 'rustore', local: local === 'rustore'};
};

export const cardSearchHaystack = (card: {
    id: string;
    name: string;
    upright: {keywords: string[]; general: string};
    reversed: {keywords: string[]; general: string};
}): string =>
    [
        card.name,
        card.id,
        ...card.upright.keywords,
        ...card.reversed.keywords,
        card.upright.general,
        card.reversed.general,
    ]
        .join(' ')
        .toLowerCase();

export const cardMatchesQuery = (
    card: {
        id: string;
        name: string;
        upright: {keywords: string[]; general: string};
        reversed: {keywords: string[]; general: string};
    },
    query: string,
): boolean => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
        return true;
    }
    return cardSearchHaystack(card).includes(needle);
};
