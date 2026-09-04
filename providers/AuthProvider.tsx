import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useRef,
    useState,
} from 'react';
import type {ReactNode} from 'react';
import type {AuthSession, AuthUser, LanguagePreference} from '../entities';
import {configureApiClient} from '../features/apiClient';
import {
    deleteAccountRequest,
    fetchMe,
    forgotPasswordRequest,
    loginRequest,
    logoutRequest,
    registerRequest,
    resendVerificationRequest,
    resetPasswordRequest,
    verifyEmailRequest,
    type RegisterInput,
} from '../features/authApi';
import {clearSession, loadSession, saveSession} from '../storage/session';

type AuthContextValue = {
    user: AuthUser | null;
    loading: boolean;
    login: (email: string, password: string) => Promise<void>;
    register: (input: RegisterInput) => Promise<void>;
    verifyEmail: (email: string, code: string) => Promise<void>;
    resendVerification: (email: string, language?: LanguagePreference) => Promise<void>;
    forgotPassword: (email: string, language?: LanguagePreference) => Promise<void>;
    resetPassword: (email: string, code: string, password: string) => Promise<void>;
    logout: () => Promise<void>;
    deleteAccount: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export const AuthProvider = ({children}: {children: ReactNode}) => {
    const [user, setUser] = useState<AuthUser | null>(null);
    const [loading, setLoading] = useState(true);
    const accessRef = useRef<string | null>(null);
    const refreshRef = useRef<string | null>(null);
    const userRef = useRef<AuthUser | null>(null);

    const persist = useCallback(async (session: AuthSession | null) => {
        if (!session) {
            accessRef.current = null;
            refreshRef.current = null;
            userRef.current = null;
            setUser(null);
            await clearSession();
            return;
        }
        accessRef.current = session.accessToken;
        refreshRef.current = session.refreshToken;
        userRef.current = session.user;
        setUser(session.user);
        await saveSession(session);
    }, []);

    useEffect(() => {
        configureApiClient({
            getAccessToken: () => accessRef.current,
            getRefreshToken: () => refreshRef.current,
            onRotated: async (rotated) => {
                const nextUser = rotated.user ?? userRef.current;
                if (!nextUser) {
                    await persist(null);
                    return;
                }
                await persist({
                    user: nextUser,
                    accessToken: rotated.accessToken,
                    refreshToken: rotated.refreshToken,
                });
            },
            onInvalid: async () => {
                await persist(null);
            },
        });
    }, [persist]);

    useEffect(() => {
        let cancelled = false;
        const restore = async () => {
            const stored = await loadSession();
            if (!stored) {
                if (!cancelled) {
                    setLoading(false);
                }
                return;
            }
            accessRef.current = stored.accessToken;
            refreshRef.current = stored.refreshToken;
            userRef.current = stored.user;
            setUser(stored.user);
            try {
                const me = await fetchMe();
                if (cancelled) {
                    return;
                }
                await persist({
                    user: me,
                    accessToken: accessRef.current ?? stored.accessToken,
                    refreshToken: refreshRef.current ?? stored.refreshToken,
                });
            } catch {
                if (!cancelled) {
                    await persist(null);
                }
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        };
        restore().catch(() => {
            if (!cancelled) {
                setLoading(false);
            }
        });
        return () => {
            cancelled = true;
        };
    }, [persist]);

    const login = useCallback(
        async (email: string, password: string) => {
            const session = await loginRequest(email, password);
            await persist(session);
        },
        [persist],
    );

    const register = useCallback(async (input: RegisterInput) => {
        await registerRequest(input);
    }, []);

    const verifyEmail = useCallback(
        async (email: string, code: string) => {
            const session = await verifyEmailRequest(email, code);
            await persist(session);
        },
        [persist],
    );

    const resendVerification = useCallback(async (email: string, language?: LanguagePreference) => {
        await resendVerificationRequest(email, language);
    }, []);

    const forgotPassword = useCallback(async (email: string, language?: LanguagePreference) => {
        await forgotPasswordRequest(email, language);
    }, []);

    const resetPassword = useCallback(
        async (email: string, code: string, password: string) => {
            const session = await resetPasswordRequest(email, code, password);
            await persist(session);
        },
        [persist],
    );

    const logout = useCallback(async () => {
        const refreshToken = refreshRef.current;
        try {
            if (refreshToken) {
                await logoutRequest(refreshToken);
            }
        } catch {
            // local sign-out still proceeds
        }
        await persist(null);
    }, [persist]);

    const deleteAccount = useCallback(async () => {
        await deleteAccountRequest();
        await persist(null);
    }, [persist]);

    const value = useMemo(
        () => ({
            user,
            loading,
            login,
            register,
            verifyEmail,
            resendVerification,
            forgotPassword,
            resetPassword,
            logout,
            deleteAccount,
        }),
        [deleteAccount, forgotPassword, loading, login, logout, register, resendVerification, resetPassword, user, verifyEmail],
    );

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider');
    }
    return context;
};
