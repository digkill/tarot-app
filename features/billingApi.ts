import {apiRequest} from './apiClient';

export type ReportPurchaseInput = {
    productId: string;
    invoiceId?: string;
    purchaseId?: string;
    orderId?: string;
    source?: 'rustore' | 'restore' | 'dev';
    sandbox?: boolean;
    expiresAt?: string;
};

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
