import React, {useMemo, useState} from 'react';
import {
    FlatList,
    Image,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useTranslation} from 'react-i18next';
import {loadDeck} from '../utils/decks';
import {useSettings} from '../providers/SettingsProvider';
import type {Arcana, Card, Suit} from '../entities';
import {cardImages} from '../utils/cardImages';

const ARCANA_FILTERS: (Arcana | 'all')[] = ['all', 'major', 'minor'];
const SUIT_FILTERS: (Suit | 'all')[] = ['all', 'wands', 'cups', 'swords', 'pentacles'];

export const DeckGalleryScreen = () => {
    const {t} = useTranslation();
    const {settings} = useSettings();
    const [arcanaFilter, setArcanaFilter] = useState<(Arcana | 'all')>('all');
    const [suitFilter, setSuitFilter] = useState<(Suit | 'all')>('all');

    const deck = useMemo(() => loadDeck(settings.language), [settings.language]);

    const filtered = useMemo(() => {
        return deck.filter((card) => {
            const arcanaMatch = arcanaFilter === 'all' || card.arcana === arcanaFilter;
            const suitMatch = suitFilter === 'all' || card.suit === suitFilter;
            return arcanaMatch && suitMatch;
        });
    }, [arcanaFilter, suitFilter, deck]);

    const renderCard = ({item}: {item: Card}) => {
        const source = cardImages[item.image];
        return (
            <View style={styles.card}>
                {source ? (
                    <View style={styles.cardImageWrapper}>
                        <Image source={source} style={styles.cardImage} />
                    </View>
                ) : null}
                <Text style={styles.cardName}>{item.name}</Text>
                <Text style={styles.cardDescription} numberOfLines={3}>
                    {item.upright.general}
                </Text>
            </View>
        );
    };

    return (
        <View style={styles.root}>
            <Image
                source={require('../assets/pattern-print.png')}
                style={styles.backdrop}
                resizeMode="repeat"
            />
            <SafeAreaView style={styles.safe}>
                <View style={styles.filters}>
                    <Text style={styles.filterLabel}>{t('deck.arcana')}</Text>
                    <View style={styles.filterRow}>
                        {ARCANA_FILTERS.map((filter) => (
                            <TouchableOpacity
                                key={filter}
                                style={[styles.chip, arcanaFilter === filter && styles.chipActive]}
                                onPress={() => setArcanaFilter(filter)}
                            >
                                <Text
                                    style={[
                                        styles.chipText,
                                        arcanaFilter === filter && styles.chipTextActive,
                                    ]}
                                >
                                    {t(`deck.arcanaFilter.${filter}`)}
                                </Text>
                            </TouchableOpacity>
                        ))}
                    </View>
                    <Text style={[styles.filterLabel, {marginTop: 16}]}>{t('deck.suits')}</Text>
                    <View style={styles.filterRow}>
                        {SUIT_FILTERS.map((filter) => (
                            <TouchableOpacity
                                key={filter}
                                style={[styles.chip, suitFilter === filter && styles.chipActive]}
                                onPress={() => setSuitFilter(filter)}
                            >
                                <Text
                                    style={[
                                        styles.chipText,
                                        suitFilter === filter && styles.chipTextActive,
                                    ]}
                                >
                                    {t(`deck.suitFilter.${filter}`)}
                                </Text>
                            </TouchableOpacity>
                        ))}
                    </View>
                </View>
                <FlatList
                    data={filtered}
                    keyExtractor={(item) => item.id}
                    renderItem={renderCard}
                    contentContainerStyle={styles.list}
                    numColumns={2}
                    columnWrapperStyle={{gap: 16}}
                    showsVerticalScrollIndicator={false}
                />
            </SafeAreaView>
        </View>
    );
};

const styles = StyleSheet.create({
    root: {
        flex: 1,
        backgroundColor: '#040307',
    },
    backdrop: {
        ...StyleSheet.absoluteFill,
        width: '100%',
        height: '100%',
        opacity: 0.25,
    },
    safe: {
        flex: 1,
        backgroundColor: 'transparent',
        paddingHorizontal: 16,
    },
    filters: {
        paddingVertical: 16,
        gap: 6,
    },
    filterLabel: {
        color: '#f7f4ea',
        fontWeight: '600',
        fontSize: 16,
    },
    filterRow: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 8,
        marginTop: 8,
    },
    chip: {
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: 20,
        backgroundColor: 'rgba(244,211,134,0.12)',
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.3)',
    },
    chipActive: {
        backgroundColor: '#6c5ce7',
        borderColor: '#6c5ce7',
    },
    chipText: {
        color: '#f7f4ea',
        fontSize: 13,
        fontWeight: '500',
    },
    chipTextActive: {
        color: '#fff',
    },
    list: {
        paddingBottom: 40,
        gap: 16,
    },
    card: {
        flex: 1,
        backgroundColor: '#1a1030',
        borderRadius: 16,
        padding: 10,
        borderWidth: 1.5,
        borderColor: '#d4af37',
        shadowColor: '#d4af37',
        shadowOpacity: 0.18,
        shadowOffset: {width: 0, height: 4},
        shadowRadius: 8,
        elevation: 6,
    },
    cardImageWrapper: {
        borderRadius: 10,
        borderWidth: 1,
        borderColor: 'rgba(212,175,55,0.5)',
        overflow: 'hidden',
        marginBottom: 10,
    },
    cardImage: {
        width: '100%',
        aspectRatio: 2 / 3,
        resizeMode: 'cover',
    },
    cardName: {
        color: '#f4d386',
        fontWeight: '600',
        fontSize: 16,
        marginBottom: 6,
    },
    cardDescription: {
        color: '#f7f4ea',
        opacity: 0.8,
        fontSize: 13,
        lineHeight: 18,
    },
});
