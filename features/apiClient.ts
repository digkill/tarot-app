import Constants from 'expo-constants';
import * as Localization from 'expo-localization';
import type {AuthSession} from '../entities';

export class ApiError extends Error {
    code: string;
    status: number;

    constructor(code: string, message: string, status: number) {
        super(message);
        this.name = 'ApiError';
        this.code = code;
        this.status = status;
    }
}

type TokenBridge = {
    getAccessToken: () => string | null;
    getRefreshToken: () => string | null;
    onRotated: (session: Pick<AuthSession, 'accessToken' | 'refreshToken' | 'user'>) => Promise<void>;
    onInvalid: () => Promise<void>;
};

let tokenBridge: TokenBridge | null = null;

export const configureApiClient = (bridge: TokenBridge) => {
    tokenBridge = bridge;
};

export const getApiBaseUrl = (): string => {
    const extra = Constants.expoConfig?.extra as {apiBaseUrl?: string} | undefined;
    const fromExtra = extra?.apiBaseUrl?.trim();
    const url = fromExtra || 'https://tarot.sorapure.fun';
    return url.replace(/\/+$/, '');
};

export const deviceTimezone = (): string => {
    try {
        return Localization.getCalendars()[0]?.timeZone || 'Europe/Moscow';
    } catch {
        return 'Europe/Moscow';
    }
};

type ErrorPayload = {
    error?: {
        code?: string;
        message?: string;
    };
};

type RequestOptions = {
    method?: string;
    body?: unknown;
    auth?: boolean;
    skipRefresh?: boolean;
};

let refreshInFlight: Promise<boolean> | null = null;

const parseError = async (response: Response): Promise<ApiError> => {
    let code = 'http_error';
    let message = `Request failed (${response.status})`;
    try {
        const payload = (await response.json()) as ErrorPayload;
        if (payload.error?.code) {
            code = payload.error.code;
        }
        if (payload.error?.message) {
            message = payload.error.message;
        }
    } catch {
        // keep fallback
    }
    return new ApiError(code, message, response.status);
};

const rotateTokens = async (): Promise<boolean> => {
    if (refreshInFlight) {
        return refreshInFlight;
    }
    refreshInFlight = (async () => {
        const refreshToken = tokenBridge?.getRefreshToken() ?? null;
        if (!refreshToken) {
            await tokenBridge?.onInvalid();
            return false;
        }
        try {
            const response = await fetch(`${getApiBaseUrl()}/api/v1/auth/refresh`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json', Accept: 'application/json'},
                body: JSON.stringify({refreshToken}),
            });
            if (!response.ok) {
                await tokenBridge?.onInvalid();
                return false;
            }
            const session = (await response.json()) as AuthSession;
            if (!session.accessToken || !session.refreshToken || !session.user) {
                await tokenBridge?.onInvalid();
                return false;
            }
            await tokenBridge?.onRotated(session);
            return true;
        } catch {
            await tokenBridge?.onInvalid();
            return false;
        }
    })();
    try {
        return await refreshInFlight;
    } finally {
        refreshInFlight = null;
    }
};

export const apiRequest = async <T>(path: string, options: RequestOptions = {}): Promise<T> => {
    const method = options.method ?? 'GET';
    const headers: Record<string, string> = {
        Accept: 'application/json',
    };
    if (options.body !== undefined) {
        headers['Content-Type'] = 'application/json';
    }
    if (options.auth !== false) {
        const access = tokenBridge?.getAccessToken();
        if (access) {
            headers.Authorization = `Bearer ${access}`;
        }
    }

    let response: Response;
    try {
        response = await fetch(`${getApiBaseUrl()}${path}`, {
            method,
            headers,
            body: options.body === undefined ? undefined : JSON.stringify(options.body),
        });
    } catch {
        throw new ApiError('network', 'network error', 0);
    }

    if (response.status === 401 && options.auth !== false && !options.skipRefresh) {
        const rotated = await rotateTokens();
        if (rotated) {
            return apiRequest<T>(path, {...options, skipRefresh: true});
        }
    }

    if (!response.ok) {
        throw await parseError(response);
    }

    if (response.status === 204) {
        return undefined as T;
    }

    const text = await response.text();
    if (!text) {
        return undefined as T;
    }
    return JSON.parse(text) as T;
};

export const isApiError = (error: unknown): error is ApiError => error instanceof ApiError;

/**
 * The access token for connections that authenticate once instead of per
 * request, such as the Arcana Clash WebSocket. `force` rotates first, for a
 * reconnect after the server rejected the token.
 */
export const socketAccessToken = async (force = false): Promise<string | null> => {
    if (force) {
        const rotated = await rotateTokens();
        return rotated ? tokenBridge?.getAccessToken() ?? null : null;
    }
    return tokenBridge?.getAccessToken() ?? null;
};
