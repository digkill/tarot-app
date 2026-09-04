import {Linking, NativeModules, Platform} from 'react-native';
import {RuStoreReactPay} from '../libs/RuStoreReactPay';
import type {Product, ProductPurchaseResult, RuStorePayError} from '../libs/RuStoreReactPay';

export const PREMIUM_PRODUCT_IDS = ['premium_monthly', 'premium_yearly', 'premium_lifetime'] as const;

export type PremiumProductId = (typeof PREMIUM_PRODUCT_IDS)[number];

export const FALLBACK_PREMIUM_PRICES: Record<PremiumProductId, number> = {
    premium_monthly: 599,
    premium_yearly: 4990,
    premium_lifetime: 6990,
};

const RUSTORE_SUBSCRIPTIONS_URL = 'rustore://profile/subscriptions';

export class PaymentsUnavailableError extends Error {
    constructor(message = 'RuStore payments are not available in this build') {
        super(message);
        this.name = 'PaymentsUnavailableError';
    }
}

export class PurchaseCancelledError extends Error {
    constructor() {
        super('Purchase cancelled by user');
        this.name = 'PurchaseCancelledError';
    }
}

export class RuStoreMissingError extends Error {
    constructor() {
        super('RuStore is not installed');
        this.name = 'RuStoreMissingError';
    }
}

// Нативный модуль есть только в Android-сборке с подключённым плагином withRuStorePay
// (dev build / EAS). В Expo Go и на iOS его нет — все методы кидают PaymentsUnavailableError.
const hasNativeModule = (): boolean =>
    Platform.OS === 'android' && !!NativeModules.RuStoreReactPaySDKModule;

const assertAvailable = () => {
    if (!hasNativeModule()) {
        throw new PaymentsUnavailableError();
    }
};

export const isPaymentsSupported = (): boolean => hasNativeModule();

export const isPremiumProductId = (value: string): value is PremiumProductId =>
    (PREMIUM_PRODUCT_IDS as readonly string[]).includes(value);

export const formatRub = (amount: number): string => `${amount.toLocaleString('ru-RU')} ₽`;

export const isPurchaseAvailable = async (): Promise<boolean> => {
    if (!hasNativeModule()) {
        return false;
    }
    try {
        const {availability} = await RuStoreReactPay.getPurchaseAvailability();
        return availability;
    } catch {
        return false;
    }
};

export const getPremiumProducts = async (): Promise<Product[]> => {
    assertAvailable();
    return RuStoreReactPay.getProducts([...PREMIUM_PRODUCT_IDS]);
};

const isPayError = (error: unknown): error is RuStorePayError =>
    RuStoreReactPay.isRuStorePayError(error);

export const purchaseStoreProduct = async (
    productId: string,
    theme: 'LIGHT' | 'DARK' = 'DARK',
): Promise<ProductPurchaseResult> => {
    assertAvailable();
    if (hasNativeModule()) {
        const installed = await RuStoreReactPay.isRuStoreInstalled();
        if (!installed) {
            throw new RuStoreMissingError();
        }
    }
    try {
        return await RuStoreReactPay.purchase({
            productId,
            preferredPurchaseType: 'ONE_STEP',
            sdkTheme: theme,
        });
    } catch (error) {
        if (isPayError(error) && error.code === 'ProductPurchaseCancelled') {
            throw new PurchaseCancelledError();
        }
        if (isPayError(error) && error.code === 'RuStoreNotInstalledException') {
            throw new RuStoreMissingError();
        }
        throw error;
    }
};

export const purchasePremium = async (
    productId: PremiumProductId,
    theme: 'LIGHT' | 'DARK' = 'DARK',
): Promise<ProductPurchaseResult> => purchaseStoreProduct(productId, theme);

export const hasActivePremiumSubscription = async (): Promise<boolean> => {
    const status = await readStorePremiumStatus();
    return status.active;
};

export type StorePremiumStatus = {
    active: boolean;
    productId?: PremiumProductId;
    expiresAt?: string;
};

const LIVE_SUB_STATUSES = new Set<string>(['ACTIVE', 'PAUSED']);
const PAID_PRODUCT_STATUSES = new Set<string>(['CONFIRMED', 'PAID']);

export const readStorePremiumStatus = async (): Promise<StorePremiumStatus> => {
    assertAvailable();
    const subscriptions = await RuStoreReactPay.getPurchases({productType: 'SUBSCRIPTION'});
    for (const row of subscriptions) {
        const sub = row.subscriptionPurchase;
        if (!sub || !isPremiumProductId(sub.productId) || !LIVE_SUB_STATUSES.has(sub.status)) {
            continue;
        }
        return {
            active: true,
            productId: sub.productId,
            expiresAt: sub.expirationDate || undefined,
        };
    }
    const owned = await RuStoreReactPay.getPurchases({productType: 'NON_CONSUMABLE_PRODUCT'});
    for (const row of owned) {
        const item = row.productPurchase;
        if (!item || item.productId !== 'premium_lifetime' || !PAID_PRODUCT_STATUSES.has(item.status)) {
            continue;
        }
        return {active: true, productId: 'premium_lifetime'};
    }
    return {active: false};
};

export const openRuStoreSubscriptions = async (): Promise<void> => {
    const canOpen = await Linking.canOpenURL(RUSTORE_SUBSCRIPTIONS_URL);
    if (canOpen) {
        await Linking.openURL(RUSTORE_SUBSCRIPTIONS_URL);
        return;
    }
    if (hasNativeModule()) {
        await RuStoreReactPay.openRuStore();
        return;
    }
    throw new RuStoreMissingError();
};

export const promptInstallRuStore = async (): Promise<void> => {
    assertAvailable();
    await RuStoreReactPay.openRuStoreDownloadInstruction();
};

export const describePayError = (error: unknown): string | null =>
    isPayError(error) ? `${error.code}: ${error.message}` : null;
