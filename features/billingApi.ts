import {apiRequest} from './apiClient';
import type {WebCheckoutProvider} from './premiumSource';

export type ReportPurchaseInput = {
    productId: string;
    invoiceId?: string;
    purchaseId?: string;
    orderId?: string;
    source?: 'rustore' | 'restore' | 'dev';
    sandbox?: boolean;
    expiresAt?: string;
};

export type CheckoutSession = {
    transactionId: string;
    paymentId?: string;
    checkoutUrl: string;
    confirmationUrl?: string;
    productId?: string;
};

export type CheckoutStatus = {
    transactionId: string;
    status: string;
    productId?: string;
    hasPremium: boolean;
    premiumSource?: string;
};

export const createCheckoutRequest = (productId: string, provider: WebCheckoutProvider) =>
    apiRequest<CheckoutSession>('/api/v1/billing/checkout', {
        method: 'POST',
        body: {productId, provider},
    });

export const fetchCheckoutRequest = (transactionId: string) =>
    apiRequest<CheckoutStatus>(`/api/v1/billing/checkout/${transactionId}`);

export type ReportPurchaseResult = {
    ok: boolean;
    transactionId?: string;
    hasPremium?: boolean;
    deckSlug?: string;
};

export const reportPurchaseRequest = (input: ReportPurchaseInput) =>
    apiRequest<ReportPurchaseResult>('/api/v1/billing/purchases', {
        method: 'POST',
        body: {
            productId: input.productId,
            invoiceId: input.invoiceId,
            purchaseId: input.purchaseId,
            orderId: input.orderId,
            source: input.source ?? 'rustore',
            sandbox: input.sandbox ?? false,
            expiresAt: input.expiresAt,
        },
    });

export const reportSubscriptionStatus = (input: {
    active: boolean;
    productId?: string;
    expiresAt?: string;
}) =>
    apiRequest<{ok: boolean; hasPremium?: boolean}>('/api/v1/billing/subscription-status', {
        method: 'POST',
        body: {
            active: input.active,
            productId: input.productId ?? '',
            expiresAt: input.expiresAt ?? '',
        },
    });
