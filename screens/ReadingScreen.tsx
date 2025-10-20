import React, {useEffect, useMemo, useRef, useState} from 'react';
import {
    Animated,
    Easing,
    LayoutChangeEvent,
    ScrollView,
    StyleSheet,
    Text,
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
import {loadDeck} from '../utils/decks';
import {generateInterpretation} from '../features/interpretation';
import TarotCard from '../components/TarotCard';

type Route = RouteProp<RootStackParamList, 'Reading'>;
type Navigation = NativeStackNavigationProp<RootStackParamList, 'Reading'>;

type DrawnEntry = {
    card: Card;
    position: SpreadPosition;
    isReversed: boolean;
};

const CARD_WIDTH = 110;
const CARD_HEIGHT = 190;

export const ReadingScreen = () => {
    const route = useRoute<Route>();
    const navigation = useNavigation<Navigation>();
    const {t} = useTranslation();
    const {spreadId, deckId} = route.params;
    const {settings} = useSettings();
    const {addReading} = useHistory();
    const [entries, setEntries] = useState<DrawnEntry[]>([]);
    const [phase, setPhase] = useState<'shuffle' | 'dealing' | 'review'>('shuffle');
    const [animatedValues, setAnimatedValues] = useState<Animated.Value[]>([]);
    const [layout, setLayout] = useState({width: 0, height: 0});
    const [saving, setSaving] = useState(false);
    const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const spread = useMemo<Spread | undefined>(() => SPREADS.find((s) => s.id === spreadId), [spreadId]);
    const deck = useMemo(() => loadDeck(settings.language), [settings.language]);

    useEffect(() => {
        startReading(settings.disableAnimations);
        return () => {
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
            const values = drawn.map((entry, index) => {
                const value = new Animated.Value(skipAnimation ? 1 : 0);
                return value;
            });
            setEntries(drawn);
            setAnimatedValues(values);
            animateEntries(values, skipAnimation);
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
        const left = layout.width * entry.position.x - CARD_WIDTH / 2;
        const top = layout.height * entry.position.y - CARD_HEIGHT / 2;
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
                style={[styles.cardWrapper, {left, top}, animatedStyle]}
            >
                <TarotCard card={entry.card} isReversed={entry.isReversed} theme="dark" />
                <Text style={styles.cardLabel}>{t(entry.position.titleKey)}</Text>
            </Animated.View>
        );
    };

    const skipAnimations = () => {
        if (phase === 'review') return;
        startReading(true);
    };

    const redraw = () => {
        startReading(false);
    };

    const handleContinue = async () => {
        if (!spread || !entries.length) return;
        setSaving(true);
        try {
            const interpretation = generateInterpretation(
                spread,
                entries,
                (key, vars) => t(key, {...vars, defaultValue: key}),
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
                    <Text style={styles.errorText}>{t('reading.missingSpread')}</Text>
                </View>
            </SafeAreaView>
        );
    }

    return (
        <SafeAreaView style={styles.safe}>
            <View style={styles.header}>
                <View>
                    <Text style={styles.spreadTitle}>{t(spread.nameKey)}</Text>
                    <Text style={styles.spreadMeta}>{t('reading.cardCount', {count: spread.maxCards})}</Text>
                </View>
                <TouchableOpacity onPress={redraw}>
                    <Text style={styles.action}>{t('reading.redraw')}</Text>
                </TouchableOpacity>
            </View>

            <View style={styles.canvasContainer}>
                <View style={styles.canvas} onLayout={handleLayout}>
                    {phase === 'shuffle' && (
                        <View style={styles.centered}>
                            <Text style={styles.shuffleText}>{t('reading.shuffling')}</Text>
                        </View>
                    )}
                    {entries.map(renderCard)}
                </View>
            </View>

            <ScrollView style={styles.details} contentContainerStyle={{paddingBottom: 20}}>
                {entries.map((entry) => (
                    <View key={`detail-${entry.position.index}`} style={styles.detailItem}>
                        <Text style={styles.detailTitle}>{t(entry.position.titleKey)}</Text>
                        <Text style={styles.detailSubtitle}>{t(entry.position.descriptionKey)}</Text>
                        <Text style={styles.detailCardName}>
                            {entry.card.name} {entry.isReversed ? t('reading.reversed') : ''}
                        </Text>
                    </View>
                ))}
            </ScrollView>

            <View style={styles.footer}>
                <TouchableOpacity style={styles.secondaryButton} onPress={skipAnimations}>
                    <Text style={styles.secondaryText}>{t('reading.skipAnimations')}</Text>
                </TouchableOpacity>
                <TouchableOpacity
                    style={[styles.primaryButton, (phase !== 'review' || saving) && styles.primaryButtonDisabled]}
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
        backgroundColor: '#040307',
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
        height: 360,
        borderRadius: 24,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        backgroundColor: 'rgba(12,10,20,0.85)',
    },
    cardWrapper: {
        position: 'absolute',
        width: CARD_WIDTH,
        alignItems: 'center',
    },
    cardLabel: {
        color: '#f7f4ea',
        fontSize: 12,
        marginTop: 6,
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
