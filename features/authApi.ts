import type {AuthSession, AuthUser, LanguagePreference} from '../entities';
import {CURRENT_CONSENT_VERSION} from '../entities';
import {apiRequest} from './apiClient';

export type RegisterInput = {
    email: string;
    password: string;
    language?: LanguagePreference;
    acceptedTerms: boolean;
    acceptedPrivacy: boolean;
    acceptedPersonalData: boolean;
};

export type VerificationStart = {
    email: string;
    verificationRequired: boolean;
};

export const loginRequest = (email: string, password: string) =>
    apiRequest<AuthSession>('/api/v1/auth/login', {
        method: 'POST',
        auth: false,
        body: {email, password},
    });

export const registerRequest = (input: RegisterInput) =>
    apiRequest<VerificationStart>('/api/v1/auth/register', {
        method: 'POST',
        auth: false,
        body: {
            email: input.email,
            password: input.password,
            language: input.language,
            acceptedTerms: input.acceptedTerms,
            acceptedPrivacy: input.acceptedPrivacy,
            acceptedPersonalData: input.acceptedPersonalData,
            consentVersion: CURRENT_CONSENT_VERSION,
        },
    });

export const verifyEmailRequest = (email: string, code: string) =>
    apiRequest<AuthSession>('/api/v1/auth/verify-email', {
        method: 'POST',
        auth: false,
        body: {email, code},
    });

export const resendVerificationRequest = (email: string, language?: LanguagePreference) =>
    apiRequest<{ok: boolean}>('/api/v1/auth/resend-verification', {
        method: 'POST',
        auth: false,
        body: {email, language},
    });

export const forgotPasswordRequest = (email: string, language?: LanguagePreference) =>
    apiRequest<{ok: boolean}>('/api/v1/auth/forgot-password', {
        method: 'POST',
        auth: false,
        body: {email, language},
    });

export const resetPasswordRequest = (email: string, code: string, password: string) =>
    apiRequest<AuthSession>('/api/v1/auth/reset-password', {
        method: 'POST',
        auth: false,
        body: {email, code, password},
    });

export const fetchMe = () => apiRequest<AuthUser>('/api/v1/me');

export const logoutRequest = (refreshToken: string) =>
    apiRequest<{ok: boolean}>('/api/v1/auth/logout', {
        method: 'POST',
        auth: false,
        body: {refreshToken},
    });

export const deleteAccountRequest = () =>
    apiRequest<{ok: boolean}>('/api/v1/me', {
        method: 'DELETE',
    });
