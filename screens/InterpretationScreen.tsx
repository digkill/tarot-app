import React, {useEffect, useMemo, useRef, useState} from 'react';
import {
    Alert,
    ActivityIndicator,
    ScrollView,
    StyleSheet,
    Text,
    TextInput,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation, useRoute} from '@react-navigation/native';
import type {RouteProp} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import ViewShot from 'react-native-view-shot';
import * as Sharing from 'expo-sharing';
import {useTranslation} from 'react-i18next';
import {RootStackParamList} from '../navigation/types';
import {useHistory} from '../providers/HistoryProvider';
import {useSettings} from '../providers/SettingsProvider';
import {SPREADS} from '../data';
import {loadDeck, findCardById} from '../utils/decks';
import {generateInterpretation} from '../features/interpretation';
import {fetchPremiumInterpretation, MissingOpenAiKeyError} from '../features/aiInterpretation';
import TarotCard from '../components/TarotCard';
import type {Card, ReadingAiInsight, SpreadPosition} from '../entities';

type Route = RouteProp<RootStackParamList, 'Interpretation'>;
type Navigation = NativeStackNavigationProp<RootStackParamList, 'Interpretation'>;

export const InterpretationScreen = () => {
    const route = useRoute<Route>();
    const navigation = useNavigation<Navigation>();
    const {readingId} = route.params;
    const {readings, updateReading, toggleFavorite} = useHistory();
    const {settings} = useSettings();
    const {t} = useTranslation();
    const viewRef = useRef<ViewShot>(null);
    const [savingNotes, setSavingNotes] = useState(false);

    const reading = readings.find((item) => item.id === readingId);
    const spread = useMemo(
        () => SPREADS.find((item) => item.id === reading?.spreadId),
        [reading?.spreadId],
    );
    const deck = useMemo(() => loadDeck(settings.language), [settings.language]);

    const entries = useMemo(() => {
        if (!reading || !spread) return [];
        return reading.items
            .map((item) => {
                const position = spread.positions.find((pos) => pos.index === item.positionIndex);
                const card = findCardById(deck, item.cardId);
                if (!position || !card) return null;
                return {position, card, isReversed: item.isReversed};
            })
            .filter(Boolean) as Array<{position: SpreadPosition; card: Card; isReversed: boolean}>;
    }, [deck, reading, spread]);

    const interpretation = useMemo(() => {
        if (!spread || !entries.length) return null;
        return generateInterpretation(spread, entries, (key, vars) => t(key, vars));
    }, [entries, spread, t]);

    const [notes, setNotes] = useState(reading?.notes ?? '');

    useEffect(() => {
        if (reading?.notes !== undefined) {
            setNotes(reading.notes);
        }
    }, [reading?.notes]);

    if (!reading || !spread) {
        return (
            <SafeAreaView style={styles.safe}>
                <View style={styles.centered}>
                    <Text style={styles.errorText}>{t('interpretation.missingReading')}</Text>
                    <TouchableOpacity onPress={() => navigation.goBack()}>
                        <Text style={styles.link}>{t('interpretation.goBack')}</Text>
                    </TouchableOpacity>
                </View>
            </SafeAreaView>
        );
    }

    const handleFavorite = () => {
        toggleFavorite(reading.id).catch(() => {});
    };

    const handleSaveNotes = async () => {
        setSavingNotes(true);
        try {
            await updateReading(reading.id, {notes});
        } finally {
            setSavingNotes(false);
        }
    };

    const handleShare = async () => {
        try {
            const uri = await viewRef.current?.capture?.();
            if (!uri) return;
            await Sharing.shareAsync(uri, {dialogTitle: t('interpretation.shareDialog')});
        } catch (error) {
            Alert.alert(t('interpretation.shareError')); 
        }
    };

    return (
        <SafeAreaView style={styles.safe}>
            <ScrollView contentContainerStyle={styles.container}>
                <ViewShot ref={viewRef} options={{format: 'png', quality: 1}} style={styles.capture}>
                    <Text style={styles.title}>{t(spread.nameKey)}</Text>
                    <Text style={styles.date}>{new Date(reading.drawnAt).toLocaleString(settings.language)}</Text>
                    {interpretation && (
                        <View style={styles.summaryBox}>
                            <Text style={styles.summaryTitle}>{t('interpretation.summaryTitle')}</Text>
                            <Text style={styles.summaryText}>{interpretation.summary}</Text>
                            {interpretation.keywords.length ? (
                                <View style={styles.keywordsRow}>
                                    {interpretation.keywords.map((keyword) => (
                                        <View key={keyword} style={styles.keywordChip}>
                                            <Text style={styles.keywordText}>{keyword}</Text>
                                        </View>
                                    ))}
                                </View>
                            ) : null}
                        </View>
                    )}
                    {entries.map((entry) => (
                        <View key={`entry-${entry.position.index}`} style={styles.cardBlock}>
                            <View style={styles.cardHeader}>
                                <Text style={styles.cardTitle}>{t(entry.position.titleKey)}</Text>
                                <Text style={styles.cardSubtitle}>{t(entry.position.descriptionKey)}</Text>
                            </View>
                            <View style={styles.cardRow}>
                                <TarotCard card={entry.card} isReversed={entry.isReversed} theme="dark" />
                                <View style={styles.cardNarrative}>
                                    <Text style={styles.cardName}>
                                        {entry.card.name} {entry.isReversed ? t('reading.reversed') : ''}
                                    </Text>
                                    <Text style={styles.cardNarrativeText}>
                                        {entry.isReversed ? entry.card.reversed.general : entry.card.upright.general}
                                    </Text>
                                </View>
                            </View>
                        </View>
                    ))}
                </ViewShot>

                <View style={styles.actionsRow}>
                    <TouchableOpacity style={styles.secondaryButton} onPress={handleFavorite}>
                        <Text style={styles.secondaryText}>
                            {reading.favorite ? t('interpretation.unfavorite') : t('interpretation.favorite')}
                        </Text>
                    </TouchableOpacity>
                    <TouchableOpacity style={styles.primaryButton} onPress={handleShare}>
                        <Text style={styles.primaryText}>{t('interpretation.share')}</Text>
                    </TouchableOpacity>
                </View>

                <View style={styles.notesBox}>
                    <Text style={styles.notesTitle}>{t('interpretation.notesTitle')}</Text>
                    <TextInput
                        multiline
                        value={notes}
                        onChangeText={setNotes}
                        placeholder={t('interpretation.notesPlaceholder')}
                        placeholderTextColor="rgba(247,244,234,0.5)"
                        style={styles.notesInput}
                    />
                    <TouchableOpacity style={styles.saveButton} onPress={handleSaveNotes} disabled={savingNotes}>
                        <Text style={styles.saveButtonText}>
                            {savingNotes ? t('interpretation.saving') : t('interpretation.saveNotes')}
                        </Text>
                    </TouchableOpacity>
                </View>
            </ScrollView>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    container: {
        padding: 20,
        paddingBottom: 40,
        gap: 20,
    },
    capture: {
        backgroundColor: 'rgba(12,10,20,0.9)',
        borderRadius: 24,
        padding: 20,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.2)',
        gap: 16,
    },
    title: {
        color: '#f4d386',
        fontSize: 22,
        fontWeight: '700',
    },
    date: {
        color: '#f7f4ea',
        opacity: 0.7,
    },
    summaryBox: {
        gap: 10,
    },
    summaryTitle: {
        color: '#f7f4ea',
        fontWeight: '600',
    },
    summaryText: {
        color: '#f7f4ea',
        lineHeight: 20,
    },
    keywordsRow: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 8,
    },
    keywordChip: {
        paddingHorizontal: 12,
        paddingVertical: 6,
        borderRadius: 14,
        backgroundColor: 'rgba(108,92,231,0.2)',
    },
    keywordText: {
        color: '#f4d386',
        fontWeight: '600',
        fontSize: 12,
    },
    cardBlock: {
        borderTopWidth: StyleSheet.hairlineWidth,
        borderColor: 'rgba(255,255,255,0.1)',
        paddingTop: 16,
        gap: 12,
    },
    cardHeader: {
        gap: 4,
    },
    cardTitle: {
        color: '#f7f4ea',
        fontWeight: '600',
        fontSize: 16,
    },
    cardSubtitle: {
        color: '#f7f4ea',
        opacity: 0.6,
        fontSize: 13,
    },
    cardRow: {
        flexDirection: 'row',
        gap: 16,
        alignItems: 'center',
    },
    cardNarrative: {
        flex: 1,
        gap: 6,
    },
    cardName: {
        color: '#f4d386',
        fontWeight: '600',
    },
    cardNarrativeText: {
        color: '#f7f4ea',
        opacity: 0.8,
        lineHeight: 20,
    },
    actionsRow: {
        flexDirection: 'row',
        gap: 12,
    },
    secondaryButton: {
        flex: 1,
        paddingVertical: 14,
        borderRadius: 16,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.4)',
    },
    secondaryText: {
        color: '#f4d386',
        textAlign: 'center',
        fontWeight: '600',
    },
    primaryButton: {
        flex: 1,
        paddingVertical: 14,
        borderRadius: 16,
        backgroundColor: '#6c5ce7',
    },
    primaryText: {
        color: '#fff',
        textAlign: 'center',
        fontWeight: '700',
    },
    notesBox: {
        backgroundColor: 'rgba(12,10,20,0.9)',
        borderRadius: 20,
        padding: 18,
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.2)',
        gap: 12,
    },
    notesTitle: {
        color: '#f7f4ea',
        fontWeight: '600',
    },
    notesInput: {
        minHeight: 100,
        color: '#f7f4ea',
        textAlignVertical: 'top',
    },
    saveButton: {
        alignSelf: 'flex-start',
        paddingVertical: 10,
        paddingHorizontal: 18,
        borderRadius: 16,
        backgroundColor: '#6c5ce7',
    },
    saveButtonText: {
        color: '#fff',
        fontWeight: '600',
    },
    centered: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        gap: 12,
    },
    errorText: {
        color: '#f7f4ea',
        fontSize: 16,
    },
    link: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
});
