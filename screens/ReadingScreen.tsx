import React, {useEffect, useMemo, useRef, useState} from 'react';
import {
    Alert,
    Animated,
    Easing,
    LayoutChangeEvent,
    ScrollView,
    StyleSheet,
    Text,
    TextInput,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation, useRoute} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {RouteProp} from '@react-navigation/native';
import {useTranslation} from 'react-i18next';
import {RootStackParamList} from '../navigation/types';
import {SPREADS} from '../data';
import type {Card, Spread, SpreadPosition} from '../entities';
import {useSettings} from '../providers/SettingsProvider';
import {useHistory} from '../providers/HistoryProvider';
import {useAppColors} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';
import {loadDeck} from '../utils/decks';
import {generateInterpretation} from '../features/interpretation';
import {consumeOneCardSlot, isOneCardSpread, isQuotaExceeded} from '../features/dailyCard';
import TarotCard from '../components/TarotCard';
import {CardMeaningSheet} from '../components/CardMeaningSheet';
import {ZoomableView} from '../components/ZoomableView';
import {cardMatchesQuery} from '../features/premiumSource';

type Route = RouteProp<RootStackParamList, 'Reading'>;
type Navigation = NativeStackNavigationProp<RootStackParamList, 'Reading'>;

type DrawnEntry = {
    card: Card;
    position: SpreadPosition;
    isReversed: boolean;
};

const BASE_CARD_W = 110;
const BASE_CARD_H = 165;
const ASPECT = BASE_CARD_H / BASE_CARD_W;
const LABEL_H = 24;
const CANVAS_PAD = 8;

function computeCardWidth(positions: SpreadPosition[], canvasW: number, canvasH: number): number {
    if (positions.length <= 1) {
        return Math.min(BASE_CARD_W, canvasW - CANVAS_PAD * 2);
    }

    let minDeltaX = Infinity;
    let minDeltaY = Infinity;

    for (let i = 0; i < positions.length; i++) {
        for (let j = i + 1; j < positions.length; j++) {
            const pi = positions[i]!;
            const pj = positions[j]!;
            const dx = Math.abs(pi.x - pj.x) * canvasW;
            const dy = Math.abs(pi.y - pj.y) * canvasH;
            if (dy < 20 && dx > 1) {
                minDeltaX = Math.min(minDeltaX, dx);
            }
            if (dx < 20 && dy > 1) {
                minDeltaY = Math.min(minDeltaY, dy);
            }
        }
    }

    const candidates: number[] = [];

    if (isFinite(minDeltaX)) {
        candidates.push(minDeltaX * 0.85);
    }

    if (isFinite(minDeltaY)) {
        const wFromY = (minDeltaY * 0.85 - LABEL_H) / ASPECT;
        candidates.push(wFromY);
    }

    const minX = Math.min(...positions.map((p) => p.x));
    const maxX = Math.max(...positions.map((p) => p.x));
    const minY = Math.min(...positions.map((p) => p.y));
    const maxY = Math.max(...positions.map((p) => p.y));

    const usableW = canvasW - CANVAS_PAD * 2;
    const usableH = canvasH - CANVAS_PAD * 2;
    const xSpan = Math.max(maxX - minX, 0.01);
    const ySpan = Math.max(maxY - minY, 0.01);
    const cols = xSpan / (1 / (positions.length + 1)) + 1;
    const rows = ySpan / (1 / (positions.length + 1)) + 1;
    candidates.push(usableW / Math.max(cols, 1) * 0.85);
    candidates.push((usableH / Math.max(rows, 1) - LABEL_H) / ASPECT * 0.85);

    const computed = candidates.reduce((min, v) => (v > 0 && v < min ? v : min), BASE_CARD_W);
    return Math.max(36, Math.round(computed));
}

export const ReadingScreen = () => {
    const route = useRoute<Route>();
    const navigation = useNavigation<Navigation>();
    const {t} = useTranslation();
    const {spreadId, deckId, quotaGranted, readingKind} = route.params;
    const {settings} = useSettings();
    const colors = useAppColors();
    const {addReading} = useHistory();
    const [entries, setEntries] = useState<DrawnEntry[]>([]);
    const [phase, setPhase] = useState<'shuffle' | 'dealing' | 'review'>('shuffle');
    const [animatedValues, setAnimatedValues] = useState<Animated.Value[]>([]);
    const [layout, setLayout] = useState({width: 0, height: 0});
    const [saving, setSaving] = useState(false);
    const [search, setSearch] = useState('');
    const [selectedEntry, setSelectedEntry] = useState<DrawnEntry | null>(null);
    const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const savedIdRef = useRef<string | null>(null);
    const persistInFlight = useRef<Promise<string | null> | null>(null);
    const startedRef = useRef(false);
    const isDailyDraw = isOneCardSpread(spreadId);

    const spread = useMemo<Spread | undefined>(() => SPREADS.find((s) => s.id === spreadId), [spreadId]);
    const deck = useMemo(() => loadDeck(settings.language), [settings.language]);

    const cardWidth = useMemo(() => {
        if (!spread || layout.width === 0) return BASE_CARD_W;
        return computeCardWidth(spread.positions, layout.width, layout.height);
    }, [spread, layout]);

    const cardHeight = Math.round(cardWidth * ASPECT);

    useEffect(() => {
        let cancelled = false;
        const boot = async () => {
            if (startedRef.current) {
                return;
            }
            startedRef.current = true;
            if (isDailyDraw && !quotaGranted) {
                try {
                    await consumeOneCardSlot(Boolean(readingKind === 'bonus'), settings.hasPremium);
                } catch (error) {
                    if (cancelled) {
                        return;
                    }
                    Alert.alert(
                        t('dailyCard.limitTitle'),
                        isQuotaExceeded(error) ? t('dailyCard.quotaReached') : t('dailyCard.unlockError'),
                    );
                    navigation.goBack();
                    return;
                }
            }
            if (!cancelled) {
                startReading(settings.disableAnimations);
            }
        };
        boot().catch(() => {});
        return () => {
            cancelled = true;
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [spread?.id, settings.language]);

    const drawCards = (): DrawnEntry[] => {
        if (!spread) return [];
        const positions = spread.positions;
        const source = [...deck];
        const drawn: DrawnEntry[] = [];

        positions.forEach((position) => {
            if (!source.length) return;
            const index = Math.floor(Math.random() * source.length);
            const [card] = source.splice(index, 1);
            if (!card) return;
            const isReversed = Math.random() < settings.reversedChance;
            drawn.push({card, position, isReversed});
        });

        return drawn;
    };

    const animateEntries = (values: Animated.Value[], skipAnimation: boolean) => {
        if (skipAnimation || settings.disableAnimations) {
            values.forEach((value) => value.setValue(1));
            setPhase('review');
            return;
        }

        setPhase('dealing');

        const animations = values.map((value) =>
            Animated.timing(value, {
                toValue: 1,
                duration: 600,
                easing: Easing.out(Easing.ease),
                useNativeDriver: true,
            }),
        );

        Animated.stagger(140, animations).start(() => {
            setPhase('review');
        });
    };

    const startReading = (skipAnimation = false) => {
        if (!spread) return;
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }

        const initiate = () => {
            const drawn = drawCards();
            const values = drawn.map(() => new Animated.Value(skipAnimation ? 1 : 0));
            setEntries(drawn);
            setAnimatedValues(values);
            animateEntries(values, skipAnimation);
            if (isDailyDraw) {
                persistDaily(drawn).catch(() => {});
            }
        };

        if (skipAnimation || settings.disableAnimations) {
            initiate();
            return;
        }

        setPhase('shuffle');
        timeoutRef.current = setTimeout(() => {
            initiate();
        }, 800);
    };

    const handleLayout = (event: LayoutChangeEvent) => {
        const {width, height} = event.nativeEvent.layout;
        setLayout({width, height});
    };

    const renderCard = (entry: DrawnEntry, index: number) => {
        const value = animatedValues[index] ?? new Animated.Value(1);

        const rawLeft = layout.width * entry.position.x - cardWidth / 2;
        const rawTop = layout.height * entry.position.y - cardHeight / 2;
        const left = Math.max(CANVAS_PAD, Math.min(rawLeft, layout.width - cardWidth - CANVAS_PAD));
        const top = Math.max(CANVAS_PAD, Math.min(rawTop, layout.height - cardHeight - LABEL_H - CANVAS_PAD));

        const animatedStyle = {
            opacity: value,
            transform: [
                {
                    translateY: value.interpolate({
                        inputRange: [0, 1],
                        outputRange: [40, 0],
                    }),
                },
                {
                    scale: value.interpolate({
                        inputRange: [0, 1],
                        outputRange: [0.9, 1],
                    }),
                },
            ],
        };

        return (
            <Animated.View
                key={entry.position.index}
                style={[styles.cardWrapper, {left, top, width: cardWidth}, animatedStyle]}
            >
                <TarotCard
                    card={entry.card}
                    isReversed={entry.isReversed}
                    startFaceDown
                    width={cardWidth}
                    artDeckId={deckId}
                    interactive={phase === 'review'}
                    highlighted={Boolean(search.trim()) && cardMatchesQuery(entry.card, search)}
                    dimmed={Boolean(search.trim()) && !cardMatchesQuery(entry.card, search)}
                    onPressFace={() => setSelectedEntry(entry)}
                />
                <Text style={[styles.cardLabel, {color: colors.text}]} numberOfLines={2}>
                    {t(entry.position.titleKey)}
                </Text>
            </Animated.View>
        );
    };

    const persistDaily = (drawn: DrawnEntry[]): Promise<string | null> => {
        if (savedIdRef.current) {
            return Promise.resolve(savedIdRef.current);
        }
        if (persistInFlight.current) {
            return persistInFlight.current;
        }
        persistInFlight.current = (async () => {
            if (!spread || !drawn.length) {
                return null;
            }
            const interpretation = generateInterpretation(
                spread,
                drawn,
                (key, vars) => String(t(key, vars ?? {})),
            );
            const reading = await addReading({
                spreadId: spread.id,
                deckId,
                items: drawn.map((entry) => ({
                    positionIndex: entry.position.index,
                    cardId: entry.card.id,
                    isReversed: entry.isReversed,
                })),
                summaryText: interpretation.summary,
                notes: '',
                kind: readingKind ?? 'daily',
            });
            savedIdRef.current = reading.id;
            return reading.id;
        })();
        return persistInFlight.current;
    };

    const skipAnimations = () => {
        if (phase === 'review') return;
        if (isDailyDraw) {
            animatedValues.forEach((value) => value.setValue(1));
            setPhase('review');
            return;
        }
        startReading(true);
    };

    const redraw = () => {
        if (isDailyDraw) {
            return;
        }
        startReading(false);
    };

    const handleContinue = async () => {
        if (!spread || !entries.length) return;
        setSaving(true);
        try {
            if (isDailyDraw) {
                const existingId = savedIdRef.current ?? (await persistDaily(entries));
                if (existingId) {
                    navigation.replace('Interpretation', {readingId: existingId});
                }
                return;
            }
            const interpretation = generateInterpretation(
                spread,
                entries,
                (key, vars) => String(t(key, vars ?? {})),
            );
            const reading = await addReading({
                spreadId: spread.id,
                deckId,
                items: entries.map((entry) => ({
                    positionIndex: entry.position.index,
                    cardId: entry.card.id,
                    isReversed: entry.isReversed,
                })),
                summaryText: interpretation.summary,
                notes: '',
                kind: isDailyDraw ? readingKind ?? 'daily' : undefined,
            });
            navigation.replace('Interpretation', {readingId: reading.id});
        } finally {
            setSaving(false);
        }
    };

    if (!spread) {
        return (
            <SafeAreaView style={styles.safe}>
                <View style={styles.centered}>
                    <Text style={[styles.errorText, {color: colors.text}]}>{t('reading.missingSpread')}</Text>
                </View>
            </SafeAreaView>
        );
    }

    const canvasHeight = spread.maxCards >= 10 ? 440 : spread.maxCards >= 7 ? 400 : 320;
    const zoomResetKey = `${spread.id}-${phase === 'shuffle' ? 'shuffle' : entries.map((entry) => `${entry.position.index}:${entry.card.id}`).join(',')}`;

    return (
        <SafeAreaView style={styles.safe}>
            <View style={styles.header}>
                <View>
                    <Text style={[styles.spreadTitle, {color: colors.gold}]}>{t(spread.nameKey)}</Text>
                    <Text style={[styles.spreadMeta, {color: colors.muted}]}>
                        {t('reading.cardCount', {count: spread.maxCards})}
                    </Text>
                </View>
                {!isDailyDraw ? (
                    <TouchableOpacity onPress={redraw}>
                        <Text style={[styles.action, {color: colors.accent}]}>{t('reading.redraw')}</Text>
                    </TouchableOpacity>
                ) : (
                    <View />
                )}
            </View>

            <View style={styles.canvasContainer}>
                <View
                    style={[
                        styles.canvas,
                        {
                            height: canvasHeight,
                            borderColor: hexAlpha(colors.gold, 0.25),
                            backgroundColor: colors.panel,
                        },
                    ]}
                    onLayout={handleLayout}
                >
                    <ZoomableView
                        style={styles.zoomLayer}
                        enabled={phase === 'review'}
                        resetKey={zoomResetKey}
                        hint={phase === 'review' ? t('reading.zoomHint') : undefined}
                        resetLabel={t('reading.resetZoom')}
                    >
                        {layout.width > 0 && entries.map(renderCard)}
                    </ZoomableView>
                    {phase === 'shuffle' && (
                        <View style={[styles.centered, styles.shuffleOverlay]}>
                            <Text style={[styles.shuffleText, {color: colors.text}]}>{t('reading.shuffling')}</Text>
                        </View>
                    )}
                </View>
            </View>

            {phase === 'review' ? (
                <TextInput
                    value={search}
                    onChangeText={setSearch}
                    placeholder={t('reading.searchPlaceholder')}
                    placeholderTextColor={hexAlpha(colors.muted, 0.8)}
                    style={[
                        styles.search,
                        {
                            color: colors.text,
                            borderColor: hexAlpha(colors.gold, 0.35),
                            backgroundColor: hexAlpha(colors.panel, 0.72),
                        },
                    ]}
                    autoCorrect={false}
                    autoCapitalize="none"
                />
            ) : null}

            <ScrollView style={styles.details} contentContainerStyle={{paddingBottom: 20}}>
                {entries
                    .filter((entry) => cardMatchesQuery(entry.card, search))
                    .map((entry) => (
                    <TouchableOpacity
                        key={`detail-${entry.position.index}`}
                        style={styles.detailItem}
                        onPress={() => setSelectedEntry(entry)}
                    >
                        <Text style={[styles.detailTitle, {color: colors.text}]}>{t(entry.position.titleKey)}</Text>
                        <Text style={[styles.detailSubtitle, {color: colors.muted}]}>
                            {t(entry.position.descriptionKey)}
                        </Text>
                        <Text style={[styles.detailCardName, {color: colors.gold}]}>
                            {entry.card.name} {entry.isReversed ? t('reading.reversed') : ''}
                        </Text>
                    </TouchableOpacity>
                ))}
                {phase === 'review' && search.trim() && !entries.some((entry) => cardMatchesQuery(entry.card, search)) ? (
                    <Text style={[styles.searchEmpty, {color: colors.muted}]}>{t('reading.searchEmpty')}</Text>
                ) : null}
            </ScrollView>

            <CardMeaningSheet
                visible={!!selectedEntry}
                card={selectedEntry?.card ?? null}
                isReversed={selectedEntry?.isReversed}
                positionTitle={selectedEntry ? t(selectedEntry.position.titleKey) : undefined}
                artDeckId={deckId}
                onClose={() => setSelectedEntry(null)}
            />

            <View style={styles.footer}>
                <TouchableOpacity
                    style={[styles.secondaryButton, {borderColor: hexAlpha(colors.gold, 0.4)}]}
                    onPress={skipAnimations}
                >
                    <Text style={[styles.secondaryText, {color: colors.gold}]}>{t('reading.skipAnimations')}</Text>
                </TouchableOpacity>
                <TouchableOpacity
                    style={[
                        styles.primaryButton,
                        {backgroundColor: colors.accent},
                        (phase !== 'review' || saving) && {opacity: 0.5},
                    ]}
                    onPress={handleContinue}
                    disabled={phase !== 'review' || saving}
                >
                    <Text style={styles.primaryText}>
                        {saving ? t('reading.saving') : t('reading.continue')}
                    </Text>
                </TouchableOpacity>
            </View>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: 'transparent',
    },
    search: {
        marginHorizontal: 20,
        marginBottom: 8,
        borderWidth: 1,
        borderRadius: 14,
        paddingHorizontal: 14,
        paddingVertical: 10,
        fontSize: 15,
    },
    searchEmpty: {
        fontSize: 14,
        textAlign: 'center',
        marginTop: 12,
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingHorizontal: 20,
        paddingTop: 12,
    },
    spreadTitle: {
        color: '#f4d386',
        fontSize: 22,
        fontWeight: '700',
    },
    spreadMeta: {
        color: '#f7f4ea',
        opacity: 0.7,
    },
    action: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
    canvasContainer: {
        paddingHorizontal: 10,
        paddingVertical: 10,
    },
    canvas: {
        borderRadius: 24,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        backgroundColor: 'rgba(12,10,20,0.85)',
        overflow: 'hidden',
    },
    zoomLayer: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
    },
    shuffleOverlay: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(4,3,7,0.55)',
    },
    cardWrapper: {
        position: 'absolute',
        alignItems: 'center',
    },
    cardLabel: {
        color: '#f7f4ea',
        fontSize: 10,
        marginTop: 2,
        textAlign: 'center',
        opacity: 0.8,
    },
    centered: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    shuffleText: {
        color: '#f7f4ea',
        fontSize: 16,
        fontWeight: '600',
    },
    errorText: {
        color: '#f7f4ea',
        fontSize: 16,
    },
    details: {
        flex: 1,
        paddingHorizontal: 20,
    },
    detailItem: {
        marginBottom: 16,
        borderBottomWidth: StyleSheet.hairlineWidth,
        borderColor: 'rgba(255,255,255,0.1)',
        paddingBottom: 12,
    },
    detailTitle: {
        color: '#f7f4ea',
        fontWeight: '600',
        marginBottom: 4,
    },
    detailSubtitle: {
        color: '#f7f4ea',
        opacity: 0.6,
        marginBottom: 6,
        fontSize: 13,
    },
    detailCardName: {
        color: '#f4d386',
        fontWeight: '600',
    },
    footer: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        gap: 12,
        padding: 20,
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
    primaryButtonDisabled: {
        backgroundColor: 'rgba(108,92,231,0.5)',
    },
    primaryText: {
        color: '#fff',
        textAlign: 'center',
        fontWeight: '700',
    },
});
