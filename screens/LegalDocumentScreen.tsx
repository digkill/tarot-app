import React from 'react';
import {ScrollView, StyleSheet, Text} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import type {NativeStackScreenProps} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {CURRENT_CONSENT_VERSION} from '../entities';
import {RootStackParamList} from '../navigation/types';

type Props = NativeStackScreenProps<RootStackParamList, 'LegalDocument'>;

export const LegalDocumentScreen = ({route}: Props) => {
    const {t} = useTranslation();
    const key = route.params.doc === 'privacy' ? 'legal.privacyParagraphs' : 'legal.termsParagraphs';
    const paragraphs = t(key, {returnObjects: true});
    const items = Array.isArray(paragraphs) ? paragraphs.filter((item) => typeof item === 'string') : [];

    return (
        <SafeAreaView style={styles.safe} edges={['bottom']}>
            <ScrollView contentContainerStyle={styles.content}>
                <Text style={styles.version}>{t('legal.version', {version: CURRENT_CONSENT_VERSION})}</Text>
                {items.map((paragraph, index) => (
                    <Text key={`${route.params.doc}-${index}`} style={styles.paragraph}>
                        {paragraph}
                    </Text>
                ))}
            </ScrollView>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    content: {
        padding: 24,
        paddingBottom: 48,
        gap: 14,
    },
    version: {
        color: '#f4d386',
        fontSize: 13,
        marginBottom: 4,
    },
    paragraph: {
        color: '#f7f4ea',
        fontSize: 15,
        lineHeight: 22,
    },
});
