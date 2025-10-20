import React from 'react';
import {ScrollView, StyleSheet, Text, TouchableOpacity, View} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import type {NativeStackScreenProps} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {RootStackParamList} from '../navigation/types';
import {useSettings} from '../providers/SettingsProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'Disclaimer'>;

export const DisclaimerScreen = ({navigation}: Props) => {
    const {t} = useTranslation();
    const {setSetting} = useSettings();

    const accept = () => {
        setSetting('acceptedDisclaimer', true).catch(() => {});
        navigation.reset({
            index: 0,
            routes: [{name: 'Main'}],
        });
    };

    return (
        <SafeAreaView style={styles.container}>
            <ScrollView contentContainerStyle={styles.content}>
                <Text style={styles.title}>{t('disclaimer.title')}</Text>
                <Text style={styles.body}>{t('disclaimer.description1')}</Text>
                <Text style={styles.body}>{t('disclaimer.description2')}</Text>
                <View style={styles.adviceBox}>
                    <Text style={styles.adviceTitle}>{t('disclaimer.remember')}</Text>
                    <Text style={styles.adviceText}>{t('disclaimer.rememberDescription')}</Text>
                </View>
            </ScrollView>
            <View style={styles.footer}>
                <TouchableOpacity style={styles.button} onPress={accept}>
                    <Text style={styles.buttonText}>{t('disclaimer.accept')}</Text>
                </TouchableOpacity>
            </View>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#08070f',
    },
    content: {
        padding: 24,
    },
    title: {
        fontSize: 24,
        fontWeight: '700',
        color: '#f4d386',
        marginBottom: 16,
    },
    body: {
        fontSize: 16,
        color: '#f7f4ea',
        lineHeight: 22,
        marginBottom: 16,
    },
    adviceBox: {
        padding: 18,
        borderRadius: 16,
        backgroundColor: 'rgba(108, 92, 231, 0.1)',
        borderWidth: 1,
        borderColor: 'rgba(108, 92, 231, 0.4)',
        marginTop: 12,
    },
    adviceTitle: {
        fontSize: 18,
        fontWeight: '600',
        color: '#6c5ce7',
        marginBottom: 8,
    },
    adviceText: {
        fontSize: 15,
        color: '#f7f4ea',
        lineHeight: 22,
    },
    footer: {
        padding: 24,
    },
    button: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 16,
        borderRadius: 16,
    },
    buttonText: {
        color: '#fff',
        fontSize: 18,
        fontWeight: '600',
        textAlign: 'center',
    },
});
