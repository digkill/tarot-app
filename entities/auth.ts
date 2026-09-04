export const CURRENT_CONSENT_VERSION = '1.0';

export type AuthUser = {
    id: string;
    email: string;
    hasPremium: boolean;
    premiumExpiresAt?: string | null;
    premiumProductId?: string;
    premiumSource?: string;
    emailVerified?: boolean;
    createdAt: string;
};

export type AuthSession = {
    user: AuthUser;
    accessToken: string;
    refreshToken: string;
};

export type LegalDocumentId = 'privacy' | 'terms';
