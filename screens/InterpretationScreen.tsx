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
import ViewShot, {ViewShotRef} from 'react-native-view-shot';
import * as Sharing from 'expo-sharing';
import {useTranslation} from 'react-i18next';
import {RootStackParamList} from '../navigation/types';
import {useHistory} from '../providers/HistoryProvider';
import {useSettings} from '../providers/SettingsProvider';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {SPREADS} from '../data';
import {loadDeck, findCardById} from '../utils/decks';
import {generateInterpretation} from '../features/interpretation';
import {fetchPremiumInterpretation} from '../features/aiInterpretation';
import {isApiError} from '../features/apiClient';
import {fetchUsage, type UsageSnapshot} from '../features/usageApi';
import TarotCard from '../components/TarotCard';
import {CardMeaningSheet} from '../components/CardMeaningSheet';
import {PremiumModal} from '../components/PremiumModal';
import type {Card, SpreadPosition} from '../entities';

type Route = RouteProp<RootStackParamList, 'Interpretation'>;
type Navigation = NativeStackNavigationProp<RootStackParamList, 'Interpretation'>;

export const InterpretationScreen = () => {
    const route = useRoute<Route>();
    const navigation = useNavigation<Navigation>();
    const {readingId} = route.params;
    const {readings, updateReading, toggleFavorite} = useHistory();
    const {settings} = useSettings();
    const colors = useAppColors();
    const {t} = useTranslation();
    const viewRef = useRef<ViewShotRef>(null);
    const [savingNotes, setSavingNotes] = useState(false);
    const [loadingAiInsights, setLoadingAiInsights] = useState(false);
    const [aiError, setAiError] = useState<string | null>(null);
    const [showPremiumPrompt, setShowPremiumPrompt] = useState(false);
    const [usage, setUsage] = useState<UsageSnapshot | null>(null);
    const [selectedEntry, setSelectedEntry] = useState<{
        card: Card;
        isReversed: boolean;
        position: SpreadPosition;
    } | null>(null);

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
        return generateInterpretation(spread, entries, (key, vars) => String(t(key, vars ?? {})));
    }, [entries, spread, t]);

    const [notes, setNotes] = useState(reading?.notes ?? '');

    useEffect(() => {
        if (reading?.notes !== undefined) {
            setNotes(reading.notes);
        }
    }, [reading?.notes]);

    useEffect(() => {
        if (!settings.hasPremium) {
            return;
        }
        fetchUsage()
            .then(setUsage)
            .catch(() => {});
    }, [settings.hasPremium, readingId]);

    if (!reading || !spread) {
        return (
            <SafeAreaView style={styles.safe}>
                <View style={styles.centered}>
                    <Text style={[styles.missingText, {color: colors.text}]}>{t('interpretation.missingReading')}</Text>
                    <TouchableOpacity onPress={() => navigation.goBack()}>
                        <Text style={[styles.link, {color: colors.accent}]}>{t('interpretation.goBack')}</Text>
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
            Alert.alert(t('interpretation.notesSaved'));
        } catch (error) {
            console.warn('[notes] failed to save', error);
            Alert.alert(t('interpretation.notesSaveError'));
        } finally {
            setSavingNotes(false);
        }
    };

    const handleShare = async () => {
        try {
            if (!(await Sharing.isAvailableAsync())) {
                Alert.alert(t('interpretation.shareError'));
                return;
            }
            const uri = await viewRef.current?.capture?.();
            if (!uri) {
                Alert.alert(t('interpretation.shareError'));
                return;
            }
            const shareUri = uri.startsWith('file://') || uri.startsWith('data:') ? uri : `file://${uri}`;
            await Sharing.shareAsync(shareUri, {dialogTitle: t('interpretation.shareDialog')});
        } catch (error) {
            console.warn('[share] failed', error);
            const message = error instanceof Error ? error.message : String(error);
            Alert.alert(t('interpretation.shareError'), message);
        }
    };

    const handleGenerateAiInsights = async () => {
        if (!settings.hasPremium) {
            setShowPremiumPrompt(true);
            return;
        }

        if (!spread || !entries.length) return;

        setLoadingAiInsights(true);
        setAiError(null);

        try {
            const aiRequest = {
                language: settings.language,
                spreadId: spread.id,
                spreadName: t(spread.nameKey),
                spreadDescription: t(spread.descriptionKey),
                entries: entries.map((entry, index) => ({
                    positionIndex: index,
                    positionTitle: t(entry.position.titleKey),
                    positionDescription: t(entry.position.descriptionKey),
                    cardName: entry.card.name,
                    uprightMeaning: entry.card.upright.general,
                    reversedMeaning: entry.card.reversed.general,
                    uprightKeywords: entry.card.upright.keywords,
                    reversedKeywords: entry.card.reversed.keywords,
                    isReversed: entry.isReversed,
                })),
            };

            const aiInsights = await fetchPremiumInterpretation(aiRequest);
            await updateReading(reading.id, {aiInsights});
            fetchUsage()
                .then(setUsage)
                .catch(() => {});
        } catch (error) {
            console.warn('[ai] interpretation failed', error);
            if (isApiError(error)) {
                if (error.code === 'unauthorized') {
                    setAiError(t('aiInterpretation.needLogin'));
                } else if (error.code === 'premium_required') {
                    setAiError(t('aiInterpretation.premiumRequired'));
                } else if (error.code === 'quota_exceeded') {
                    setAiError(
                        t('aiInterpretation.quotaReached', {
                            limit: usage?.interpretations.limit || 50,
                        }),
                    );
                } else if (error.code === 'rate_limited') {
                    setAiError(t('aiInterpretation.rateLimited'));
                } else {
                    setAiError(t('aiInterpretation.unavailable'));
                }
            } else {
                setAiError(t('aiInterpretation.unavailable'));
            }
        } finally {
            setLoadingAiInsights(false);
        }
    };

    return (
        <SafeAreaView style={styles.safe}>
            <ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
                <ViewShot
                    ref={viewRef}
                    options={{format: 'png', quality: 1}}
                    style={[
                        styles.capture,
                        {
                            backgroundColor: colors.panel,
                            borderColor: hexAlpha(colors.gold, 0.2),
                        },
                    ]}
                >
                    <Text style={[styles.title, {color: colors.gold}]}>{t(spread.nameKey)}</Text>
                    <Text style={[styles.date, {color: colors.muted}]}>
                        {new Date(reading.drawnAt).toLocaleString(settings.language)}
                    </Text>
                    {interpretation && (
                        <View style={styles.summaryBox}>
                            <Text style={[styles.summaryTitle, {color: colors.text}]}>
                                {t('interpretation.summaryTitle')}
                            </Text>
                            <Text style={[styles.summaryText, {color: colors.text}]}>{interpretation.summary}</Text>
                            {interpretation.keywords.length ? (
                                <View style={styles.keywordsRow}>
                                    {interpretation.keywords.map((keyword) => (
                                        <View
                                            key={keyword}
                                            style={[
                                                styles.keywordChip,
                                                {backgroundColor: hexAlpha(colors.accent, 0.2)},
                                            ]}
                                        >
                                            <Text style={[styles.keywordText, {color: colors.gold}]}>{keyword}</Text>
                                        </View>
                                    ))}
                                </View>
                            ) : null}
                        </View>
                    )}
                    {entries.map((entry) => (
                        <View key={`entry-${entry.position.index}`} style={styles.cardBlock}>
                            <View style={styles.cardHeader}>
                                <Text style={[styles.cardTitle, {color: colors.text}]}>
                                    {t(entry.position.titleKey)}
                                </Text>
                                <Text style={[styles.cardSubtitle, {color: colors.muted}]}>
                                    {t(entry.position.descriptionKey)}
                                </Text>
                            </View>
                            <View style={styles.cardRow}>
                                <TarotCard
                                    card={entry.card}
                                    isReversed={entry.isReversed}
                                    artDeckId={reading.deckId}
                                    onPressFace={() => setSelectedEntry(entry)}
                                />
                                <View style={styles.cardNarrative}>
                                    <Text style={[styles.cardName, {color: colors.gold}]}>
                                        {entry.card.name} {entry.isReversed ? t('reading.reversed') : ''}
                                    </Text>
                                    <Text style={[styles.cardNarrativeText, {color: colors.text}]}>
                                        {entry.isReversed ? entry.card.reversed.general : entry.card.upright.general}
                                    </Text>
                                </View>
                            </View>
                        </View>
                    ))}
                </ViewShot>

                <View style={styles.actionsRow}>
                    <TouchableOpacity
                        style={[styles.secondaryButton, {borderColor: hexAlpha(colors.gold, 0.4)}]}
                        onPress={handleFavorite}
                    >
                        <Text style={[styles.secondaryText, {color: colors.gold}]}>
                            {reading.favorite ? t('interpretation.unfavorite') : t('interpretation.favorite')}
                        </Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={[styles.primaryButton, {backgroundColor: colors.accent}]}
                        onPress={handleShare}
                    >
                        <Text style={styles.primaryText}>{t('interpretation.share')}</Text>
                    </TouchableOpacity>
                </View>

                {!settings.hasPremium && (
                    <View
                        style={[
                            styles.premiumBox,
                            {
                                backgroundColor: hexAlpha(colors.accent, 0.1),
                                borderColor: hexAlpha(colors.accent, 0.3),
                            },
                        ]}
                    >
                        <Text style={[styles.premiumTitle, {color: colors.gold}]}>
                            {t('aiInterpretation.premiumFeature')}
                        </Text>
                        <Text style={[styles.premiumText, {color: colors.text}]}>
                            {t('aiInterpretation.unlockMessage')}
                        </Text>
                        <TouchableOpacity
                            style={[styles.unlockButton, {backgroundColor: colors.accent}]}
                            onPress={() => setShowPremiumPrompt(true)}
                        >
                            <Text style={styles.unlockButtonText}>{t('aiInterpretation.unlock')}</Text>
                        </TouchableOpacity>
                    </View>
                )}

                {settings.hasPremium && (
                    <View
                        style={[
                            styles.aiInsightsBox,
                            {
                                backgroundColor: colors.panel,
                                borderColor: hexAlpha(colors.accent, 0.4),
                            },
                        ]}
                    >
                        <Text style={[styles.aiInsightsTitle, {color: colors.gold}]}>
                            {t('aiInterpretation.title')}
                        </Text>
                        {usage ? (
                            <Text style={[styles.quotaHint, {color: colors.muted}]}>
                                {t('aiInterpretation.quotaHint', {
                                    remaining: usage.interpretations.remaining,
                                    limit: usage.interpretations.limit,
                                })}
                            </Text>
                        ) : null}
                        {!reading.aiInsights && !loadingAiInsights && (
                            <TouchableOpacity
                                style={[styles.generateButton, {backgroundColor: colors.accent}]}
                                onPress={handleGenerateAiInsights}
                            >
                                <Text style={styles.generateButtonText}>{t('aiInterpretation.generate')}</Text>
                            </TouchableOpacity>
                        )}
                        {loadingAiInsights && (
                            <View style={styles.loadingBox}>
                                <ActivityIndicator color={colors.accent} />
                                <Text style={[styles.loadingText, {color: colors.text}]}>
                                    {t('aiInterpretation.loading')}
                                </Text>
                            </View>
                        )}
                        {aiError && (
                            <View style={styles.errorBox}>
                                <Text style={[styles.errorText, {color: colors.danger}]}>{aiError}</Text>
                                <TouchableOpacity onPress={handleGenerateAiInsights}>
                                    <Text style={[styles.retryText, {color: colors.accent}]}>
                                        {t('aiInterpretation.retry')}
                                    </Text>
                                </TouchableOpacity>
                            </View>
                        )}
                        {reading.aiInsights && !loadingAiInsights && (
                            <View style={styles.aiContent}>
                                <Text style={[styles.aiSummary, {color: colors.text}]}>
                                    {reading.aiInsights.summary}
                                </Text>
                                {reading.aiInsights.positions.map((pos) => (
                                    <View key={pos.positionIndex} style={styles.aiPositionBlock}>
                                        <Text style={[styles.aiPositionTitle, {color: colors.gold}]}>
                                            {pos.positionTitle} - {pos.cardName}
                                        </Text>
                                        <Text style={[styles.aiPositionText, {color: colors.text}]}>
                                            {pos.meaning}
                                        </Text>
                                    </View>
                                ))}
                        
                                <TouchableOpacity
                                    style={[styles.refreshButton, {borderColor: hexAlpha(colors.accent, 0.5)}]}
                                    onPress={handleGenerateAiInsights}
                                >
                                    <Text style={[styles.refreshButtonText, {color: colors.accent}]}>
                                        {t('aiInterpretation.refreshInterpretation')}
                                    </Text>
                                </TouchableOpacity>
                            </View>
                        )}
                    </View>
                )}

                <View
                    style={[
                        styles.notesBox,
                        {
                            backgroundColor: colors.panel,
                            borderColor: hexAlpha(colors.accent, 0.2),
                        },
                    ]}
                >
                    <Text style={[styles.notesTitle, {color: colors.text}]}>{t('interpretation.notesTitle')}</Text>
                    <TextInput
                        multiline
                        value={notes}
                        onChangeText={setNotes}
                        placeholder={t('interpretation.notesPlaceholder')}
                        placeholderTextColor={colors.muted}
                        style={[styles.notesInput, {color: colors.text}]}
                    />
                    <TouchableOpacity
                        style={[styles.saveButton, {backgroundColor: colors.accent}]}
                        onPress={handleSaveNotes}
                        disabled={savingNotes}
                    >
                        <Text style={styles.saveButtonText}>
                            {savingNotes ? t('interpretation.saving') : t('interpretation.saveNotes')}
                        </Text>
                    </TouchableOpacity>
                </View>
            </ScrollView>

            <PremiumModal visible={showPremiumPrompt} onClose={() => setShowPremiumPrompt(false)} />
            <CardMeaningSheet
                visible={!!selectedEntry}
                card={selectedEntry?.card ?? null}
                isReversed={selectedEntry?.isReversed}
                positionTitle={selectedEntry ? t(selectedEntry.position.titleKey) : undefined}
                artDeckId={reading.deckId}
                onClose={() => setSelectedEntry(null)}
            />
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: 'transparent',
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
    missingText: {
        color: '#f7f4ea',
        fontSize: 16,
    },
    link: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
    premiumBox: {
        backgroundColor: 'rgba(108,92,231,0.1)',
        borderRadius: 20,
        padding: 20,
        borderWidth: 2,
        borderColor: 'rgba(108,92,231,0.3)',
        gap: 12,
        alignItems: 'center',
    },
    premiumTitle: {
        color: '#f4d386',
        fontSize: 18,
        fontWeight: '700',
    },
    premiumText: {
        color: '#f7f4ea',
        textAlign: 'center',
        lineHeight: 20,
    },
    unlockButton: {
        paddingVertical: 12,
        paddingHorizontal: 24,
        borderRadius: 16,
        backgroundColor: '#6c5ce7',
    },
    unlockButtonText: {
        color: '#fff',
        fontWeight: '700',
        fontSize: 16,
    },
    aiInsightsBox: {
        backgroundColor: 'rgba(12,10,20,0.9)',
        borderRadius: 20,
        padding: 18,
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.4)',
        gap: 12,
    },
    aiInsightsTitle: {
        color: '#f4d386',
        fontSize: 18,
        fontWeight: '700',
    },
    quotaHint: {
        fontSize: 13,
        marginTop: -4,
    },
    generateButton: {
        paddingVertical: 14,
        borderRadius: 16,
        backgroundColor: '#6c5ce7',
        alignItems: 'center',
    },
    generateButtonText: {
        color: '#fff',
        fontWeight: '700',
        fontSize: 16,
    },
    loadingBox: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 12,
        padding: 20,
    },
    loadingText: {
        color: '#f7f4ea',
        fontSize: 14,
    },
    errorBox: {
        gap: 8,
    },
    errorText: {
        color: '#ff6b6b',
        fontSize: 14,
    },
    retryText: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
    aiContent: {
        gap: 16,
    },
    aiSummary: {
        color: '#f7f4ea',
        fontSize: 16,
        lineHeight: 24,
        fontWeight: '500',
    },
    aiPositionBlock: {
        gap: 8,
        paddingTop: 12,
        borderTopWidth: StyleSheet.hairlineWidth,
        borderColor: 'rgba(255,255,255,0.1)',
    },
    aiPositionTitle: {
        color: '#f4d386',
        fontWeight: '600',
        fontSize: 15,
    },
    aiPositionText: {
        color: '#f7f4ea',
        lineHeight: 20,
        opacity: 0.9,
    },
    aiModel: {
        color: '#f7f4ea',
        opacity: 0.5,
        fontSize: 12,
        fontStyle: 'italic',
    },
    refreshButton: {
        paddingVertical: 10,
        paddingHorizontal: 16,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.5)',
        alignSelf: 'flex-start',
    },
    refreshButtonText: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
});
