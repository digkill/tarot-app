import React, {useMemo, useState} from 'react';
import {
    ActivityIndicator,
    KeyboardAvoidingView,
    Platform,
    ScrollView,
    StyleSheet,
    Text,
    TextInput,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import type {NativeStackScreenProps} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {ConsentCheckbox} from '../components/ConsentCheckbox';
import {isApiError} from '../features/apiClient';
import {RootStackParamList} from '../navigation/types';
import {useAuth} from '../providers/AuthProvider';
import {useSettings} from '../providers/SettingsProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'Auth'>;
type Mode = 'login' | 'register';

export const AuthScreen = ({navigation}: Props) => {
    const {t} = useTranslation();
    const {login, register} = useAuth();
    const {settings} = useSettings();
    const [mode, setMode] = useState<Mode>('login');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [acceptedTerms, setAcceptedTerms] = useState(false);
    const [acceptedPrivacy, setAcceptedPrivacy] = useState(false);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const canSubmit = useMemo(() => {
        if (!email.trim() || password.length < 8 || submitting) {
            return false;
        }
        if (mode === 'register') {
            return password === confirmPassword && acceptedTerms && acceptedPrivacy;
        }
        return true;
    }, [acceptedPrivacy, acceptedTerms, confirmPassword, email, mode, password, submitting]);

    const mapError = (err: unknown): string => {
        if (isApiError(err)) {
            if (err.code === 'network') {
                return t('auth.errors.network');
            }
            const key = `auth.errors.${err.code}`;
            const translated = t(key);
            if (translated !== key) {
                return translated;
            }
        }
        return t('auth.errors.generic');
    };

    const submit = async () => {
        if (!canSubmit) {
            return;
        }
        setError(null);
        setSubmitting(true);
        try {
            if (mode === 'login') {
                try {
                    await login(email.trim(), password);
                } catch (err) {
                    if (isApiError(err) && err.code === 'email_unverified') {
                        navigation.navigate('VerifyEmail', {email: email.trim()});
                        return;
                    }
                    throw err;
                }
                return;
            }
            try {
                await register({
                    email: email.trim(),
                    password,
                    language: settings.language,
                    acceptedTerms,
                    acceptedPrivacy,
                    acceptedPersonalData: acceptedPrivacy,
                });
            } catch (err) {
                if (isApiError(err) && (err.code === 'mail_failed' || err.code === 'code_cooldown')) {
                    navigation.navigate('VerifyEmail', {email: email.trim()});
                    return;
                }
                throw err;
            }
            navigation.navigate('VerifyEmail', {email: email.trim()});
        } catch (err) {
            setError(mapError(err));
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <SafeAreaView style={styles.safe}>
            <KeyboardAvoidingView
                style={styles.flex}
                behavior={Platform.OS === 'ios' ? 'padding' : undefined}
            >
                <ScrollView
                    contentContainerStyle={styles.content}
                    keyboardShouldPersistTaps="handled"
                >
                    <Text style={styles.title}>
                        {mode === 'login' ? t('auth.loginTitle') : t('auth.registerTitle')}
                    </Text>
                    <Text style={styles.subtitle}>
                        {mode === 'login' ? t('auth.loginSubtitle') : t('auth.registerSubtitle')}
                    </Text>

                    <Text style={styles.label}>{t('auth.email')}</Text>
                    <TextInput
                        value={email}
                        onChangeText={setEmail}
                        autoCapitalize="none"
                        autoCorrect={false}
                        keyboardType="email-address"
                        textContentType="emailAddress"
                        autoComplete="email"
                        placeholder={t('auth.emailPlaceholder')}
                        placeholderTextColor="rgba(247,244,234,0.4)"
                        style={styles.input}
                    />

                    <Text style={styles.label}>{t('auth.password')}</Text>
                    <TextInput
                        value={password}
                        onChangeText={setPassword}
                        secureTextEntry
                        textContentType={mode === 'register' ? 'newPassword' : 'password'}
                        autoComplete={mode === 'register' ? 'new-password' : 'password'}
                        placeholder={t('auth.passwordPlaceholder')}
                        placeholderTextColor="rgba(247,244,234,0.4)"
                        style={styles.input}
                    />

                    {mode === 'register' ? (
                        <>
                            <Text style={styles.label}>{t('auth.confirmPassword')}</Text>
                            <TextInput
                                value={confirmPassword}
                                onChangeText={setConfirmPassword}
                                secureTextEntry
                                textContentType="newPassword"
                                placeholder={t('auth.confirmPasswordPlaceholder')}
                                placeholderTextColor="rgba(247,244,234,0.4)"
                                style={styles.input}
                            />
                            {password.length > 0 && password.length < 8 ? (
                                <Text style={styles.hint}>{t('auth.passwordHint')}</Text>
                            ) : null}
                            {confirmPassword.length > 0 && confirmPassword !== password ? (
                                <Text style={styles.hint}>{t('auth.passwordMismatch')}</Text>
                            ) : null}

                            <ConsentCheckbox
                                checked={acceptedTerms}
                                onToggle={() => setAcceptedTerms((value) => !value)}
                            >
                                <Text style={styles.consentText}>
                                    {t('auth.acceptTermsPrefix')}
                                    <Text
                                        style={styles.link}
                                        onPress={() => navigation.navigate('LegalDocument', {doc: 'terms'})}
                                    >
                                        {t('auth.termsLink')}
                                    </Text>
                                </Text>
                            </ConsentCheckbox>

                            <ConsentCheckbox
                                checked={acceptedPrivacy}
                                onToggle={() => setAcceptedPrivacy((value) => !value)}
                            >
                                <Text style={styles.consentText}>
                                    {t('auth.acceptPrivacyPrefix')}
                                    <Text
                                        style={styles.link}
                                        onPress={() =>
                                            navigation.navigate('LegalDocument', {doc: 'privacy'})
                                        }
                                    >
                                        {t('auth.privacyLink')}
                                    </Text>
                                </Text>
                            </ConsentCheckbox>

                            <Text style={styles.ageNote}>{t('auth.ageNote')}</Text>
                        </>
                    ) : null}

                    {error ? <Text style={styles.error}>{error}</Text> : null}

                    <TouchableOpacity
                        style={[styles.button, !canSubmit && styles.buttonDisabled]}
                        onPress={() => {
                            submit().catch(() => {});
                        }}
                        disabled={!canSubmit}
                    >
                        {submitting ? (
                            <ActivityIndicator color="#fff" />
                        ) : (
                            <Text style={styles.buttonText}>
                                {mode === 'login' ? t('auth.loginCta') : t('auth.registerCta')}
                            </Text>
                        )}
                    </TouchableOpacity>

                    {mode === 'login' ? (
                        <TouchableOpacity
                            onPress={() => navigation.navigate('ForgotPassword', {email: email.trim()})}
                        >
                            <Text style={styles.switchText}>{t('auth.forgotLink')}</Text>
                        </TouchableOpacity>
                    ) : null}

                    <TouchableOpacity
                        onPress={() => {
                            setMode(mode === 'login' ? 'register' : 'login');
                            setError(null);
                        }}
                    >
                        <Text style={styles.switchText}>
                            {mode === 'login' ? t('auth.needAccount') : t('auth.haveAccount')}
                        </Text>
                    </TouchableOpacity>
                </ScrollView>
            </KeyboardAvoidingView>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    flex: {
        flex: 1,
    },
    content: {
        padding: 24,
        paddingBottom: 48,
        gap: 8,
    },
    title: {
        fontSize: 28,
        fontWeight: '700',
        color: '#f4d386',
        marginBottom: 8,
    },
    subtitle: {
        color: '#f7f4ea',
        opacity: 0.75,
        lineHeight: 22,
        marginBottom: 16,
    },
    label: {
        color: '#f7f4ea',
        fontWeight: '600',
        marginTop: 8,
    },
    input: {
        backgroundColor: 'rgba(247,244,234,0.08)',
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        borderRadius: 14,
        color: '#f7f4ea',
        paddingHorizontal: 14,
        paddingVertical: 12,
        fontSize: 16,
    },
    hint: {
        color: '#ff6b6b',
        fontSize: 13,
    },
    consentText: {
        color: '#f7f4ea',
        lineHeight: 20,
        fontSize: 14,
    },
    link: {
        color: '#6c5ce7',
        fontWeight: '700',
    },
    ageNote: {
        color: '#f7f4ea',
        opacity: 0.6,
        fontSize: 13,
        lineHeight: 18,
        marginTop: 4,
    },
    error: {
        color: '#ff6b6b',
        marginTop: 8,
        lineHeight: 20,
    },
    button: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 16,
        borderRadius: 16,
        alignItems: 'center',
        marginTop: 16,
    },
    buttonDisabled: {
        opacity: 0.45,
    },
    buttonText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: '600',
    },
    switchText: {
        color: '#f4d386',
        textAlign: 'center',
        marginTop: 18,
        fontWeight: '600',
    },
});
