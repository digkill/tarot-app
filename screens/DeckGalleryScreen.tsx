import React, {useMemo, useState} from 'react';
import {
    ActivityIndicator,
    Alert,
    FlatList,
    Image,
    ScrollView,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
    type ImageSourcePropType,
} from 'react-native';
import {Image as ExpoImage, type ImageProps} from 'expo-image';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useTranslation} from 'react-i18next';
import {loadDeck} from '../utils/decks';
import {useSettings} from '../providers/SettingsProvider';
import {useDeckShop} from '../providers/DeckShopProvider';
import type {Arcana, Card, Suit} from '../entities';
import type {ShopDeck} from '../features/shopApi';
import {localizeShopText} from '../features/shopApi';
import {formatRub, PurchaseCancelledError, RuStoreMissingError} from '../features/payments';
import {hexAlpha} from '../theme/appColors';
import {cardImages} from '../utils/cardImages';

const ARCANA_FILTERS: (Arcana | 'all')[] = ['all', 'major', 'minor'];
const SUIT_FILTERS: (Suit | 'all')[] = ['all', 'wands', 'cups', 'swords', 'pentacles'];
const CARD_ASPECT = 2 / 3;

const FaceArt = ({
    source,
    style,
}: {
    source: ImageSourcePropType | undefined;
    style: ImageProps['style'];
}) => {
    if (!source) {
        return <View style={{width: '100%', height: '100%'}} />;
    }
    return <ExpoImage source={source} style={style} contentFit="contain" />;
};

export const DeckGalleryScreen = () => {
    const {t, i18n} = useTranslation();
    const {settings} = useSettings();
    const {
        colors,
        decks,
        selectedSlug,
        selectDeck,
        purchaseDeck,
        isOwned,
        titleOf,
        loading,
        faceSource,
    } = useDeckShop();
    const [arcanaFilter, setArcanaFilter] = useState<(Arcana | 'all')>('all');
    const [suitFilter, setSuitFilter] = useState<(Suit | 'all')>('all');
    const [buying, setBuying] = useState<string | null>(null);
    const [listWidth, setListWidth] = useState(0);
    const gridImageHeight =
        listWidth > 0 ? Math.round((((listWidth - 16) / 2) - 20) * (1 / CARD_ASPECT)) : 210;

    const meanings = useMemo(() => loadDeck(settings.language), [settings.language]);
    const filtered = useMemo(() => {
        return meanings.filter((card) => {
            const arcanaMatch = arcanaFilter === 'all' || card.arcana === arcanaFilter;
            const suitMatch = suitFilter === 'all' || card.suit === suitFilter;
            return arcanaMatch && suitMatch;
        });
    }, [arcanaFilter, suitFilter, meanings]);

    const onBuy = async (deck: ShopDeck) => {
        if (buying) {
            return;
        }
        setBuying(deck.slug);
        try {
            await purchaseDeck(deck.slug);
        } catch (error) {
            if (error instanceof PurchaseCancelledError) {
                return;
            }
            if (error instanceof RuStoreMissingError) {
                Alert.alert(t('premium.rustoreMissingTitle'), t('premium.rustoreMissing'));
                return;
            }
            if (error instanceof Error && error.message === 'payments_unavailable') {
                Alert.alert(t('premium.paymentsUnavailable'));
                return;
            }
            Alert.alert(t('premium.purchaseError'));
        } finally {
            setBuying(null);
        }
    };

    const renderShopDeck = (item: ShopDeck) => {
        const owned = isOwned(item.slug) || item.isFree || item.bundled;
        const active = selectedSlug === item.slug;
        const price = item.isFree || item.bundled ? t('deck.free') : formatRub(item.priceKop / 100);
        return (
            <View
                key={item.slug}
                style={[
                    styles.shopCard,
                    {
                        backgroundColor: colors.panel,
                        borderColor: active ? colors.accent : hexAlpha(colors.gold, 0.45),
                    },
                ]}
            >
                <View style={[styles.coverFrame, {backgroundColor: colors.bg}]}>
                    {item.coverUrl ? (
                        <FaceArt source={{uri: item.coverUrl}} style={styles.coverArt} />
                    ) : item.bundled && cardImages['the_fool.jpeg'] ? (
                        <FaceArt source={cardImages['the_fool.jpeg']} style={styles.coverArt} />
                    ) : (
                        <View style={styles.coverArt} />
                    )}
                </View>
                <Text style={[styles.shopTitle, {color: colors.gold}]}>{titleOf(item)}</Text>
                <Text style={[styles.shopHint, {color: colors.text}]} numberOfLines={3}>
                    {localizeShopText(item.descriptions, i18n.language, t('deck.classicHint'))}
                </Text>
                <Text style={[styles.shopPrice, {color: colors.text}]}>
                    {price}
                    {owned ? ` · ${t('deck.owned')}` : ''}
                </Text>
                {owned ? (
                    <TouchableOpacity
                        style={[styles.shopBtn, {backgroundColor: active ? colors.gold : colors.accent}]}
                        onPress={() => {
                            selectDeck(item.slug).catch(() => {});
                        }}
                    >
                        <Text style={[styles.shopBtnText, {color: active ? colors.bg : '#fff'}]}>
                            {active ? t('deck.selected') : t('deck.select')}
                        </Text>
                    </TouchableOpacity>
                ) : (
                    <TouchableOpacity
                        style={[styles.shopBtn, {backgroundColor: colors.accent}]}
                        onPress={() => {
                            onBuy(item).catch(() => {});
                        }}
                        disabled={buying === item.slug}
                    >
                        {buying === item.slug ? (
                            <ActivityIndicator color="#fff" />
                        ) : (
                            <Text style={styles.shopBtnText}>{t('deck.buy')}</Text>
                        )}
                    </TouchableOpacity>
                )}
            </View>
        );
    };

    const renderCard = ({item}: {item: Card}) => {
        const source = faceSource(item.image, selectedSlug);
        return (
            <View style={[styles.card, {backgroundColor: colors.panel, borderColor: hexAlpha(colors.gold, 0.45)}]}>
                <View
                    style={[
                        styles.cardImageWrapper,
                        {
                            borderColor: hexAlpha(colors.gold, 0.5),
                            backgroundColor: colors.bg,
                            height: gridImageHeight,
                        },
                    ]}
                >
                    <FaceArt source={source} style={styles.cardImage} />
                </View>
                <Text style={[styles.cardName, {color: colors.gold}]}>{item.name}</Text>
                <Text style={[styles.cardDescription, {color: colors.text}]} numberOfLines={3}>
                    {item.upright.general}
                </Text>
            </View>
        );
    };

    return (
        <View style={[styles.root, {backgroundColor: colors.bg}]}>
            <Image
                source={require('../assets/pattern-print.png')}
                style={styles.backdrop}
                resizeMode="repeat"
            />
            <SafeAreaView style={styles.safe}>
                <FlatList
                    data={filtered}
                    keyExtractor={(item) => item.id}
                    renderItem={renderCard}
                    numColumns={2}
                    columnWrapperStyle={{gap: 16}}
                    showsVerticalScrollIndicator={false}
                    contentContainerStyle={styles.list}
                    extraData={gridImageHeight}
                    onLayout={(event) => {
                        const next = Math.round(event.nativeEvent.layout.width);
                        if (next > 0 && next !== listWidth) {
                            setListWidth(next);
                        }
                    }}
                    ListHeaderComponent={
                        <View>
                            <Text style={[styles.pageTitle, {color: colors.text}]}>{t('deck.shopTitle')}</Text>
                            <Text style={[styles.pageHint, {color: colors.muted}]}>{t('deck.shopHint')}</Text>
                            {loading ? <ActivityIndicator color={colors.accent} style={styles.loader} /> : null}
                            <ScrollView
                                horizontal
                                showsHorizontalScrollIndicator={false}
                                contentContainerStyle={styles.shopList}
                            >
                                {decks.map(renderShopDeck)}
                            </ScrollView>
                            <View style={styles.filters}>
                                <Text style={[styles.filterLabel, {color: colors.text}]}>{t('deck.arcana')}</Text>
                                <View style={styles.filterRow}>
                                    {ARCANA_FILTERS.map((filter) => (
                                        <TouchableOpacity
                                            key={filter}
                                            style={[
                                                styles.chip,
                                                {borderColor: hexAlpha(colors.gold, 0.45)},
                                                arcanaFilter === filter && {
                                                    backgroundColor: colors.accent,
                                                    borderColor: colors.accent,
                                                },
                                            ]}
                                            onPress={() => setArcanaFilter(filter)}
                                        >
                                            <Text style={[styles.chipText, {color: colors.text}]}>
                                                {t(`deck.arcanaFilter.${filter}`)}
                                            </Text>
                                        </TouchableOpacity>
                                    ))}
                                </View>
                                <Text style={[styles.filterLabel, {color: colors.text, marginTop: 16}]}>
                                    {t('deck.suits')}
                                </Text>
                                <View style={styles.filterRow}>
                                    {SUIT_FILTERS.map((filter) => (
                                        <TouchableOpacity
                                            key={filter}
                                            style={[
                                                styles.chip,
                                                {borderColor: hexAlpha(colors.gold, 0.45)},
                                                suitFilter === filter && {
                                                    backgroundColor: colors.accent,
                                                    borderColor: colors.accent,
                                                },
                                            ]}
                                            onPress={() => setSuitFilter(filter)}
                                        >
                                            <Text style={[styles.chipText, {color: colors.text}]}>
                                                {t(`deck.suitFilter.${filter}`)}
                                            </Text>
                                        </TouchableOpacity>
                                    ))}
                                </View>
                            </View>
                        </View>
                    }
                />
            </SafeAreaView>
        </View>
    );
};

const styles = StyleSheet.create({
    root: {flex: 1},
    backdrop: {
        ...StyleSheet.absoluteFill,
        width: '100%',
        height: '100%',
        opacity: 0.25,
    },
    safe: {flex: 1, paddingHorizontal: 16, backgroundColor: 'transparent'},
    pageTitle: {fontSize: 22, fontWeight: '700', marginTop: 8},
    pageHint: {fontSize: 14, marginTop: 6, marginBottom: 12, lineHeight: 20},
    loader: {marginBottom: 8},
    shopList: {gap: 12, paddingBottom: 8},
    shopCard: {
        width: 168,
        borderRadius: 16,
        padding: 12,
        borderWidth: 1.5,
        marginRight: 12,
        overflow: 'hidden',
    },
    coverFrame: {
        width: '100%',
        height: 216,
        borderRadius: 10,
        marginBottom: 10,
        overflow: 'hidden',
    },
    coverArt: {
        width: '100%',
        height: '100%',
    },
    shopTitle: {fontWeight: '700', fontSize: 16, marginBottom: 4},
    shopHint: {opacity: 0.8, fontSize: 13, lineHeight: 18, marginBottom: 8, minHeight: 54},
    shopPrice: {fontSize: 13, marginBottom: 10},
    shopBtn: {borderRadius: 12, paddingVertical: 10, alignItems: 'center', minHeight: 40, justifyContent: 'center'},
    shopBtnText: {color: '#fff', fontWeight: '700'},
    filters: {paddingVertical: 12, gap: 6},
    filterLabel: {fontWeight: '600', fontSize: 16},
    filterRow: {flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginTop: 8},
    chip: {
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: 20,
        backgroundColor: 'rgba(244,211,134,0.12)',
        borderWidth: 1,
    },
    chipText: {fontSize: 13, fontWeight: '500'},
    list: {paddingBottom: 40, gap: 16},
    card: {
        flex: 1,
        maxWidth: '48%',
        borderRadius: 16,
        padding: 10,
        borderWidth: 1.5,
        overflow: 'hidden',
    },
    cardImageWrapper: {
        width: '100%',
        borderRadius: 10,
        borderWidth: 1,
        overflow: 'hidden',
        marginBottom: 10,
    },
    cardImage: {
        width: '100%',
        height: '100%',
    },
    cardName: {fontWeight: '600', fontSize: 16, marginBottom: 6},
    cardDescription: {opacity: 0.8, fontSize: 13, lineHeight: 18},
});
