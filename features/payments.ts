import {NativeModules, Platform} from 'react-native';
import Constants from 'expo-constants';
import {RuStoreReactPay} from '../libs/RuStoreReactPay';
import type {Product, RuStorePayError} from '../libs/RuStoreReactPay';

export const PREMIUM_PRODUCT_ID: string =
    Constants.expoConfig?.extra?.rustorePremiumProductId ?? 'premium_monthly';

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

export const isPurchaseAvailable = async (): Promise<boolean> => {
    if (!hasNativeModule()) return false;
    try {
        const {availability} = await RuStoreReactPay.getPurchaseAvailability();
        return availability;
    } catch {
        return false;
    }
};

export const getPremiumProduct = async (): Promise<Product | null> => {
    assertAvailable();
    const products = await RuStoreReactPay.getProducts([PREMIUM_PRODUCT_ID]);
    return products.find((item) => item.productId === PREMIUM_PRODUCT_ID) ?? null;
};

const isPayError = (error: unknown): error is RuStorePayError =>
    RuStoreReactPay.isRuStorePayError(error);

export const purchasePremium = async (theme: 'LIGHT' | 'DARK' = 'DARK'): Promise<void> => {
    assertAvailable();
    try {
        // Подписки в RuStore поддерживают только одностадийную оплату (ONE_STEP)
        await RuStoreReactPay.purchase({
            productId: PREMIUM_PRODUCT_ID,
            preferredPurchaseType: 'ONE_STEP',
            sdkTheme: theme,
        });
    } catch (error) {
        if (isPayError(error) && error.code === 'ProductPurchaseCancelled') {
            throw new PurchaseCancelledError();
        }
        throw error;
    }
};

export const hasActivePremiumSubscription = async (): Promise<boolean> => {
    assertAvailable();
    const purchases = await RuStoreReactPay.getPurchases({productType: 'SUBSCRIPTION'});
    return purchases.some(
        (purchase) =>
            purchase.subscriptionPurchase?.productId === PREMIUM_PRODUCT_ID &&
            purchase.subscriptionPurchase.status === 'ACTIVE',
    );
};

export const describePayError = (error: unknown): string | null =>
    isPayError(error) ? `${error.code}: ${error.message}` : null;
