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
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import type {NativeStackScreenProps} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {isApiError} from '../features/apiClient';
import {RootStackParamList} from '../navigation/types';
import {useAuth} from '../providers/AuthProvider';
import {useSettings} from '../providers/SettingsProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'VerifyEmail'>;

export const VerifyEmailScreen = ({navigation, route}: Props) => {
    const {t} = useTranslation();
    const {settings} = useSettings();
    const {verifyEmail, resendVerification} = useAuth();
    const email = route.params.email;
    const [code, setCode] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [info, setInfo] = useState<string | null>(null);

    const canSubmit = useMemo(() => code.trim().length === 6 && !submitting, [code, submitting]);

    const mapError = (err: unknown): string => {
        if (isApiError(err)) {
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
            await verifyEmail(email, code.trim());
        } catch (err) {
            setError(mapError(err));
        } finally {
            setSubmitting(false);
        }
    };

    const resend = async () => {
        setError(null);
        setInfo(null);
        setSubmitting(true);
        try {
            await resendVerification(email, settings.language);
            setInfo(t('auth.codeSent'));
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
                <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
                    <Text style={styles.title}>{t('auth.verifyTitle')}</Text>
                    <Text style={styles.subtitle}>{t('auth.verifySubtitle', {email})}</Text>

                    <Text style={styles.label}>{t('auth.code')}</Text>
                    <TextInput
                        value={code}
                        onChangeText={(value) => setCode(value.replace(/[^\d]/g, '').slice(0, 6))}
                        keyboardType="number-pad"
                        textContentType="oneTimeCode"
                        placeholder="000000"
                        placeholderTextColor="rgba(247,244,234,0.4)"
                        style={styles.input}
                    />

                    {error ? <Text style={styles.error}>{error}</Text> : null}
                    {info ? <Text style={styles.info}>{info}</Text> : null}

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
                            <Text style={styles.buttonText}>{t('auth.verifyCta')}</Text>
                        )}
                    </TouchableOpacity>

                    <TouchableOpacity onPress={() => { resend().catch(() => {}); }} disabled={submitting}>
                        <Text style={styles.switchText}>{t('auth.resendCode')}</Text>
                    </TouchableOpacity>
                    <TouchableOpacity onPress={() => navigation.navigate('Auth')}>
                        <Text style={styles.switchText}>{t('auth.backToLogin')}</Text>
                    </TouchableOpacity>
                </ScrollView>
            </KeyboardAvoidingView>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {flex: 1, backgroundColor: '#040307'},
    flex: {flex: 1},
    content: {padding: 24, paddingBottom: 48, gap: 8},
    title: {fontSize: 28, fontWeight: '700', color: '#f4d386', marginBottom: 8},
    subtitle: {color: '#f7f4ea', opacity: 0.75, lineHeight: 22, marginBottom: 16},
    label: {color: '#f7f4ea', fontWeight: '600', marginTop: 8},
    input: {
        backgroundColor: 'rgba(247,244,234,0.08)',
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        borderRadius: 14,
        color: '#f7f4ea',
        paddingHorizontal: 14,
        paddingVertical: 12,
        fontSize: 22,
        letterSpacing: 8,
        textAlign: 'center',
    },
    error: {color: '#ff6b6b', marginTop: 8, lineHeight: 20},
    info: {color: '#51cf66', marginTop: 8, lineHeight: 20},
    button: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 16,
        borderRadius: 16,
        alignItems: 'center',
        marginTop: 16,
    },
    buttonDisabled: {opacity: 0.45},
    buttonText: {color: '#fff', fontSize: 18, fontWeight: '600'},
    switchText: {color: '#f4d386', textAlign: 'center', marginTop: 18, fontWeight: '600'},
});
