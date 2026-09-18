import React, {useEffect, useMemo, useState} from 'react';
import {
    ActivityIndicator,
    Image,
    Modal,
    Pressable,
    ScrollView,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {Ionicons} from '@expo/vector-icons';
import {useArcana} from '../providers/ArcanaProvider';
import {useAppColors, useDeckShop} from '../providers/DeckShopProvider';
import {useSettings} from '../providers/SettingsProvider';
import {hexAlpha} from '../theme/appColors';
import {
    heroCardId,
    type ArcanaCardView,
    type ArcanaGameView,
    type ArcanaStatusView,
    type ArcanaTarget,
    type ArcanaTargetKind,
} from '../features/arcanaApi';
import {
    arcanaCardImageFile,
    arcanaCardNames,
    arcanaEventSummary,
    arcanaLogLine,
    arcanaStatusIcon,
    describeArcanaSide,
    isNegativeArcanaStatus,
} from '../features/arcanaText';
import type {RootStackParamList} from '../navigation/types';

type Navigation = NativeStackNavigationProp<RootStackParamList, 'ArcanaBattle'>;

const CLASSIC_DECK = 'rws';

/** The whole match: searching, the battlefield, mulligan and the result. */
export const ArcanaBattleScreen = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const navigation = useNavigation<Navigation>();
    const {settings} = useSettings();
    const {selectedSlug, faceSource, backSource, decks, titleOf} = useDeckShop();
    const arcana = useArcana();
    const {view, stage, opponentDeck, deadlineAt, lastError} = arcana;

    const [detail, setDetail] = useState<{card: ArcanaCardView; deck?: string; fromHand: boolean} | null>(null);
    const [power, setPower] = useState<'ability' | 'ultimate' | null>(null);
    const [mulliganPicks, setMulliganPicks] = useState<string[]>([]);
    const [menuOpen, setMenuOpen] = useState(false);
    const [toast, setToast] = useState<string | null>(null);
    const [now, setNow] = useState(Date.now());

    const names = arcanaCardNames(settings.language);
    const cardName = (id: string) => names[id] ?? id;
    const opponentDeckSlug = opponentDeck ?? CLASSIC_DECK;

    useEffect(() => {
        if (stage === 'lobby') {
            navigation.goBack();
        }
    }, [navigation, stage]);

    useEffect(() => {
        if (!deadlineAt) {
            return;
        }
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, [deadlineAt]);

    useEffect(() => {
        if (!lastError) {
            return;
        }
        const key = `arcana.errors.${lastError.code}`;
        const message = t(key);
        setToast(message === key ? t('arcana.errors.generic') : message);
        const timer = setTimeout(() => setToast(null), 2500);
        return () => clearTimeout(timer);
    }, [lastError, t]);

    const art = (cardId: string | undefined, deckSlug?: string) => {
        const slug = deckSlug ?? selectedSlug;
        if (!cardId) {
            return backSource(slug);
        }
        return faceSource(arcanaCardImageFile(cardId), slug) ?? backSource(slug);
    };

    const opponentDeckName = useMemo(() => {
        const deck = decks.find((item) => item.slug === opponentDeck && !item.bundled);
        return deck ? titleOf(deck) : null;
    }, [decks, opponentDeck, titleOf]);

    if (stage === 'searching') {
        return (
            <SafeAreaView style={styles.searching}>
                <ActivityIndicator size="large" color={colors.gold} />
                <Text style={[styles.searchingText, {color: colors.text}]}>
                    {arcana.online ? t('arcana.searching') : t('arcana.connecting')}
                </Text>
                {lastError?.code === 'connection_failed' ? (
                    <Text style={[styles.searchingError, {color: colors.danger}]}>{t('arcana.connectionFailed')}</Text>
                ) : null}
                <TouchableOpacity onPress={arcana.cancelSearch} style={styles.cancel}>
                    <Text style={[styles.cancelText, {color: colors.gold}]}>{t('arcana.cancelSearch')}</Text>
                </TouchableOpacity>
            </SafeAreaView>
        );
    }

    if (!view) {
        return (
            <SafeAreaView style={styles.searching}>
                <ActivityIndicator color={colors.gold} />
                <Text style={[styles.searchingText, {color: colors.text}]}>{t('arcana.connecting')}</Text>
            </SafeAreaView>
        );
    }

    const myTurn = view.phase === 'playing' && view.active_player === view.you.id;
    const secondsLeft = deadlineAt ? Math.max(0, Math.ceil((deadlineAt - now) / 1000)) : null;

    const statusChips = (list: ArcanaStatusView[]) => (
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.statusRow}>
            {list.map((status) => {
                const bad = isNegativeArcanaStatus(status.name);
                return (
                    <View
                        key={status.name}
                        style={[styles.statusChip, {backgroundColor: hexAlpha(bad ? colors.danger : colors.gold, 0.18)}]}
                    >
                        <Ionicons
                            name={arcanaStatusIcon(status.name) as never}
                            size={11}
                            color={bad ? colors.danger : colors.gold}
                        />
                        <Text style={[styles.statusText, {color: bad ? colors.danger : colors.gold}]}>
                            {status.amount}
                            {status.turns ? `·${status.turns}` : ''}
                        </Text>
                    </View>
                );
            })}
        </ScrollView>
    );

    const healthBar = (hp: number, maxHp: number, armor: number) => (
        <View style={styles.healthWrap}>
            <View style={[styles.healthTrack, {backgroundColor: hexAlpha(colors.text, 0.12)}]}>
                <View
                    style={[
                        styles.healthFill,
                        {backgroundColor: colors.danger, width: `${Math.max(0, (hp / Math.max(1, maxHp)) * 100)}%`},
                    ]}
                />
            </View>
            <View style={styles.healthLabels}>
                <Ionicons name="heart" size={12} color={colors.danger} />
                <Text style={[styles.healthText, {color: colors.text}]}>{`${Math.max(0, hp)}/${maxHp}`}</Text>
                {armor > 0 ? (
                    <>
                        <Ionicons name="shield" size={12} color={colors.gold} />
                        <Text style={[styles.healthText, {color: colors.gold}]}>{armor}</Text>
                    </>
                ) : null}
            </View>
        </View>
    );

    const lastPlayed = (mine: boolean): ArcanaCardView | null => {
        for (let i = view.log.length - 1; i >= 0; i -= 1) {
            const entry = view.log[i];
            if (entry.action !== 'card.play') {
                continue;
            }
            if ((entry.player === view.you.id) !== mine) {
                continue;
            }
            if (!entry.card_id) {
                return null;
            }
            return {
                uid: entry.card_uid ?? entry.card_id,
                card_id: entry.card_id,
                reversed: entry.reversed,
                cost: 0,
                base_cost: 0,
            };
        }
        return null;
    };

    const cardTile = (
        card: ArcanaCardView,
        options: {
            width: number;
            deck?: string;
            selected?: boolean;
            dimmed?: boolean;
            hideName?: boolean;
            onPress?: () => void;
        },
    ) => {
        const source = art(card.card_id, options.deck);
        const costColor =
            card.cost < card.base_cost ? '#2ecc71' : card.cost > card.base_cost ? colors.danger : colors.accent;
        return (
            <TouchableOpacity
                key={card.uid}
                disabled={!options.onPress}
                onPress={options.onPress}
                style={[
                    styles.tile,
                    {
                        width: options.width + 8,
                        borderColor: options.selected ? colors.gold : card.playable ? colors.accent : 'transparent',
                        opacity: options.dimmed ? 0.5 : 1,
                    },
                ]}
            >
                <View>
                    {source ? (
                        <Image
                            source={source}
                            style={[
                                {width: options.width, height: options.width * 1.5, borderRadius: 8},
                                card.reversed ? styles.reversed : null,
                            ]}
                            resizeMode="cover"
                        />
                    ) : null}
                    {card.base_cost > 0 || card.cost > 0 ? (
                        <View style={[styles.costBadge, {backgroundColor: costColor}]}>
                            <Text style={styles.costText}>{card.cost}</Text>
                        </View>
                    ) : null}
                    {card.reversed ? (
                        <View style={[styles.reversedBadge, {backgroundColor: colors.bg}]}>
                            <Ionicons name="swap-vertical" size={12} color={colors.gold} />
                        </View>
                    ) : null}
                </View>
                {options.hideName ? null : (
                    <Text numberOfLines={2} style={[styles.tileName, {color: colors.text, width: options.width}]}>
                        {card.card_id ? cardName(card.card_id) : t('arcana.hiddenCard')}
                    </Text>
                )}
            </TouchableOpacity>
        );
    };

    const targetsFor = (kinds: ArcanaTargetKind[] | undefined, selfUid?: string) => kinds ?? [];

    const chooseTarget = (
        kinds: ArcanaTargetKind[],
        selfUid: string | undefined,
        onChosen: (target?: ArcanaTarget) => void,
        actionLabel: string,
    ) => {
        if (kinds.length === 0) {
            return null;
        }
        return (
            <View style={styles.targets}>
                {kinds.length > 1 ? (
                    <Text style={[styles.sheetSection, {color: colors.text}]}>{t('arcana.chooseTarget')}</Text>
                ) : null}
                {kinds.map((kind) => {
                    if (kind === 'none') {
                        return (
                            <TouchableOpacity
                                key={kind}
                                style={[styles.primary, {backgroundColor: colors.accent}]}
                                onPress={() => onChosen(undefined)}
                            >
                                <Text style={styles.primaryText}>{actionLabel}</Text>
                            </TouchableOpacity>
                        );
                    }
                    if (kind === 'enemy' || kind === 'self') {
                        const label =
                            kinds.length > 1
                                ? t(kind === 'enemy' ? 'arcana.targetEnemy' : 'arcana.targetSelf')
                                : actionLabel;
                        return (
                            <TouchableOpacity
                                key={kind}
                                style={[styles.primary, {backgroundColor: colors.accent}]}
                                onPress={() => onChosen({kind})}
                            >
                                <Text style={styles.primaryText}>{label}</Text>
                            </TouchableOpacity>
                        );
                    }
                    const pool = (kind === 'hand_card' ? view.you.hand : view.you.discard).filter(
                        (item) => item.uid !== selfUid && item.card_id,
                    );
                    return (
                        <View key={kind} style={styles.targetPool}>
                            <Text style={[styles.sheetSection, {color: colors.text}]}>
                                {t(kind === 'hand_card' ? 'arcana.targetHandCard' : 'arcana.targetDiscardCard')}
                            </Text>
                            <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.handRow}>
                                {pool.map((item) =>
                                    cardTile(item, {
                                        width: 72,
                                        onPress: () => onChosen({kind, cardUid: item.uid}),
                                    }),
                                )}
                            </ScrollView>
                        </View>
                    );
                })}
            </View>
        );
    };

    const renderDetailSheet = () => {
        if (!detail) {
            return null;
        }
        const def = arcana.catalog?.cards.find((card) => card.id === detail.card.card_id);
        const side = def ? (detail.card.reversed ? def.reversed : def.upright) : null;
        const other = def ? (detail.card.reversed ? def.upright : def.reversed) : null;
        const canPlay = detail.fromHand && myTurn && Boolean(detail.card.playable);
        return (
            <Modal visible transparent animationType="slide" onRequestClose={() => setDetail(null)}>
                <Pressable style={styles.backdrop} onPress={() => setDetail(null)} />
                <View style={[styles.sheet, {backgroundColor: colors.panel}]}>
                    <ScrollView contentContainerStyle={styles.sheetContent}>
                        <Image
                            source={art(detail.card.card_id, detail.deck)}
                            style={[styles.sheetArt, detail.card.reversed ? styles.reversed : null]}
                            resizeMode="contain"
                        />
                        <Text style={[styles.sheetTitle, {color: colors.gold}]}>
                            {detail.card.card_id ? cardName(detail.card.card_id) : t('arcana.hiddenCard')}
                        </Text>
                        <Text style={[styles.sheetMeta, {color: colors.text}]}>
                            {(detail.fromHand ? `${t('arcana.cost', {cost: detail.card.cost})} · ` : '') +
                                t(detail.card.reversed ? 'arcana.reversed' : 'arcana.upright')}
                        </Text>
                        {side ? (
                            <View style={[styles.effects, {backgroundColor: hexAlpha(colors.bg, 0.6)}]}>
                                {describeArcanaSide(side, t).map((line) => (
                                    <Text key={line} style={[styles.effectLine, {color: colors.text}]}>
                                        • {line}
                                    </Text>
                                ))}
                            </View>
                        ) : null}
                        {other ? (
                            <View style={[styles.effects, {backgroundColor: hexAlpha(colors.bg, 0.3)}]}>
                                <Text style={[styles.effectTitle, {color: colors.muted}]}>
                                    {t(detail.card.reversed ? 'arcana.upright' : 'arcana.reversed')}
                                </Text>
                                {describeArcanaSide(other, t).map((line) => (
                                    <Text key={line} style={[styles.effectLine, {color: colors.muted}]}>
                                        • {line}
                                    </Text>
                                ))}
                            </View>
                        ) : null}
                        {canPlay
                            ? chooseTarget(
                                  targetsFor(detail.card.targets),
                                  detail.card.uid,
                                  (target) => {
                                      arcana.playCard(detail.card.uid, target);
                                      setDetail(null);
                                  },
                                  t('arcana.play'),
                              )
                            : null}
                        {detail.fromHand && myTurn && !detail.card.playable ? (
                            <Text style={[styles.sheetMeta, {color: colors.muted}]}>{t('arcana.notPlayable')}</Text>
                        ) : null}
                    </ScrollView>
                </View>
            </Modal>
        );
    };

    const renderPowerSheet = () => {
        if (!power) {
            return null;
        }
        const usable = power === 'ability' ? view.you.ability_usable : view.you.ultimate_usable;
        const kinds = (power === 'ability' ? view.you.ability_targets : view.you.ultimate_targets) ?? [];
        return (
            <Modal visible transparent animationType="slide" onRequestClose={() => setPower(null)}>
                <Pressable style={styles.backdrop} onPress={() => setPower(null)} />
                <View style={[styles.sheet, {backgroundColor: colors.panel}]}>
                    <ScrollView contentContainerStyle={styles.sheetContent}>
                        <Text style={[styles.sheetTitle, {color: colors.gold}]}>
                            {t(power === 'ability' ? 'arcana.ability' : 'arcana.ultimate')}
                        </Text>
                        <Text style={[styles.sheetMeta, {color: colors.text}]}>
                            {t(`arcana.heroText.${view.you.hero.id}.${power}`)}
                        </Text>
                        {usable
                            ? chooseTarget(
                                  kinds,
                                  undefined,
                                  (target) => {
                                      if (power === 'ability') {
                                          arcana.useAbility(target);
                                      } else {
                                          arcana.useUltimate(target);
                                      }
                                      setPower(null);
                                  },
                                  t(power === 'ability' ? 'arcana.ability' : 'arcana.ultimate'),
                              )
                            : <Text style={[styles.sheetMeta, {color: colors.muted}]}>{t('arcana.notPlayable')}</Text>}
                    </ScrollView>
                </View>
            </Modal>
        );
    };

    const renderMulligan = () => {
        const limit = arcana.catalog?.rules.mulligan_max ?? 3;
        return (
            <View style={[styles.overlay, {backgroundColor: hexAlpha(colors.bg, 0.95)}]}>
                <Text style={[styles.overlayTitle, {color: colors.gold}]}>{t('arcana.mulliganTitle')}</Text>
                {view.you.mulliganed ? (
                    <>
                        <ActivityIndicator color={colors.gold} />
                        <Text style={{color: colors.text}}>{t('arcana.waitingOpponent')}</Text>
                    </>
                ) : (
                    <>
                        <Text style={[styles.overlayHint, {color: colors.text}]}>
                            {t('arcana.mulliganHint', {count: limit})}
                        </Text>
                        <View style={styles.mulliganGrid}>
                            {view.you.hand.map((card) =>
                                cardTile(card, {
                                    width: 86,
                                    selected: mulliganPicks.includes(card.uid),
                                    onPress: () =>
                                        setMulliganPicks((current) =>
                                            current.includes(card.uid)
                                                ? current.filter((uid) => uid !== card.uid)
                                                : current.length < limit
                                                  ? [...current, card.uid]
                                                  : current,
                                        ),
                                }),
                            )}
                        </View>
                        <TouchableOpacity
                            style={[styles.primary, {backgroundColor: colors.accent, alignSelf: 'stretch'}]}
                            onPress={() => {
                                arcana.mulligan(mulliganPicks);
                                setMulliganPicks([]);
                            }}
                        >
                            <Text style={styles.primaryText}>
                                {mulliganPicks.length === 0
                                    ? t('arcana.keepHand')
                                    : t('arcana.replace', {count: mulliganPicks.length})}
                            </Text>
                        </TouchableOpacity>
                    </>
                )}
            </View>
        );
    };

    const renderResult = () => {
        const result = arcana.finished;
        if (!result) {
            return null;
        }
        const won = result.result === 'win';
        const deck = decks.find((item) => item.slug === opponentDeck && !item.bundled);
        return (
            <View style={[styles.overlay, {backgroundColor: hexAlpha(colors.bg, 0.94)}]}>
                <Ionicons
                    name={won ? 'trophy' : result.result === 'loss' ? 'moon' : 'help-circle'}
                    size={64}
                    color={won ? colors.gold : colors.muted}
                />
                <Text style={[styles.resultTitle, {color: won ? colors.gold : colors.text}]}>
                    {t(won ? 'arcana.victory' : result.result === 'loss' ? 'arcana.defeat' : 'arcana.noResult')}
                </Text>
                <Text style={[styles.overlayHint, {color: colors.text}]}>{t(`arcana.reason.${result.reason}`)}</Text>
                {deck ? (
                    <TouchableOpacity
                        style={[styles.deckPromo, {backgroundColor: colors.panel}]}
                        onPress={() => {
                            arcana.closeResult();
                            navigation.navigate('Main', {
                                screen: 'Decks',
                                params: {slug: deck.slug},
                            } as never);
                        }}
                    >
                        <Image
                            source={deck.coverUrl ? {uri: deck.coverUrl} : art('the_fool', deck.slug)}
                            style={styles.deckCover}
                            resizeMode="cover"
                        />
                        <View style={styles.deckPromoText}>
                            <Text style={[styles.deckPromoTitle, {color: colors.text}]}>
                                {t('arcana.opponentDeck', {deck: titleOf(deck)})}
                            </Text>
                            <Text style={[styles.deckPromoLink, {color: colors.accent}]}>{t('arcana.viewDeck')}</Text>
                        </View>
                        <Ionicons name="chevron-forward" size={18} color={colors.muted} />
                    </TouchableOpacity>
                ) : null}
                <TouchableOpacity
                    style={[styles.primary, {backgroundColor: colors.accent, alignSelf: 'stretch'}]}
                    onPress={() => arcana.battle(selectedSlug)}
                >
                    <Text style={styles.primaryText}>{t('arcana.playAgain')}</Text>
                </TouchableOpacity>
                <TouchableOpacity onPress={arcana.closeResult}>
                    <Text style={[styles.cancelText, {color: colors.gold}]}>{t('arcana.close')}</Text>
                </TouchableOpacity>
            </View>
        );
    };

    const theirLast = lastPlayed(false);
    const myLast = lastPlayed(true);

    return (
        <SafeAreaView style={styles.safe} edges={['top', 'bottom']}>
            <View style={[styles.panel, {backgroundColor: hexAlpha(colors.panel, 0.9), borderColor: !myTurn && view.phase === 'playing' ? hexAlpha(colors.danger, 0.7) : 'transparent'}]}>
                <Image source={art(heroCardId(view.opponent.hero.id), opponentDeckSlug)} style={styles.portrait} resizeMode="cover" />
                <View style={styles.panelBody}>
                    <View style={styles.panelHeader}>
                        <Text style={[styles.heroName, {color: colors.gold}]}>
                            {names[heroCardId(view.opponent.hero.id)] ?? view.opponent.hero.id}
                        </Text>
                        <View style={styles.counters}>
                            <Ionicons name="albums-outline" size={12} color={colors.text} />
                            <Text style={[styles.counterText, {color: colors.text}]}>{view.opponent.hand_count}</Text>
                            <Ionicons name="layers-outline" size={12} color={colors.text} />
                            <Text style={[styles.counterText, {color: colors.text}]}>{view.opponent.deck_count}</Text>
                            <Ionicons name="water-outline" size={12} color={colors.text} />
                            <Text style={[styles.counterText, {color: colors.text}]}>
                                {`${view.opponent.mana}/${view.opponent.max_mana}`}
                            </Text>
                            {arcana.opponentConnected ? null : <Ionicons name="wifi-outline" size={12} color={colors.danger} />}
                        </View>
                    </View>
                    {healthBar(view.opponent.hero.hp, view.opponent.hero.max_hp, view.opponent.hero.armor)}
                    {statusChips(view.opponent.statuses)}
                    {opponentDeckName ? (
                        <Text style={[styles.deckLabel, {color: colors.muted}]}>
                            {t('arcana.opponentDeck', {deck: opponentDeckName})}
                        </Text>
                    ) : null}
                </View>
            </View>

            <View style={styles.center}>
                <View style={styles.turnRow}>
                    <Text style={[styles.turnText, {color: myTurn ? colors.gold : colors.text}]}>
                        {view.phase === 'mulligan'
                            ? t('arcana.mulliganTitle')
                            : myTurn
                              ? t('arcana.yourTurn')
                              : t('arcana.opponentTurn')}
                    </Text>
                    {secondsLeft === null ? null : (
                        <View style={styles.timer}>
                            <Ionicons name="timer-outline" size={16} color={secondsLeft <= 10 ? colors.danger : colors.text} />
                            <Text style={[styles.timerText, {color: secondsLeft <= 10 ? colors.danger : colors.text}]}>
                                {secondsLeft}
                            </Text>
                        </View>
                    )}
                </View>
                <View style={[styles.logBox, {backgroundColor: hexAlpha(colors.bg, 0.55)}]}>
                    <View style={styles.lastPlayed}>
                        {theirLast
                            ? cardTile(theirLast, {
                                  width: 44,
                                  hideName: true,
                                  deck: opponentDeckSlug,
                                  onPress: () => setDetail({card: theirLast, deck: opponentDeckSlug, fromHand: false}),
                              })
                            : null}
                        {myLast
                            ? cardTile(myLast, {
                                  width: 44,
                                  hideName: true,
                                  onPress: () => setDetail({card: myLast, fromHand: false}),
                              })
                            : null}
                    </View>
                    <View style={styles.logLines}>
                        {[...view.log].slice(-4).reverse().map((entry) => {
                            const line = arcanaLogLine(entry, view.you.id, cardName, t);
                            if (!line) {
                                return null;
                            }
                            return (
                                <View key={entry.seq} style={styles.logRow}>
                                    <Text
                                        numberOfLines={1}
                                        style={[styles.logText, {color: entry.player === view.you.id ? colors.gold : colors.text}]}
                                    >
                                        {line}
                                    </Text>
                                    <Text style={[styles.logAmount, {color: colors.muted}]}>
                                        {arcanaEventSummary(entry, view.you.id)}
                                    </Text>
                                </View>
                            );
                        })}
                    </View>
                </View>
                {arcana.emotes.map((bubble) => (
                    <View
                        key={bubble.id}
                        style={[
                            styles.emote,
                            {backgroundColor: bubble.mine ? colors.gold : colors.text, alignSelf: bubble.mine ? 'flex-end' : 'flex-start'},
                        ]}
                    >
                        <Text style={[styles.emoteText, {color: colors.bg}]}>{t(`arcana.emotes.${bubble.emote}`)}</Text>
                    </View>
                ))}
                {!arcana.online && stage === 'match' ? (
                    <Text style={[styles.banner, {color: colors.danger}]}>{t('arcana.reconnecting')}</Text>
                ) : null}
                {!arcana.opponentConnected && stage === 'match' ? (
                    <Text style={[styles.banner, {color: colors.muted}]}>{t('arcana.opponentDisconnected')}</Text>
                ) : null}
                {toast ? <Text style={[styles.banner, {color: colors.danger}]}>{toast}</Text> : null}
            </View>

            <View style={[styles.panel, {backgroundColor: hexAlpha(colors.panel, 0.9), borderColor: myTurn ? hexAlpha(colors.gold, 0.8) : 'transparent'}]}>
                <Image source={art(heroCardId(view.you.hero.id))} style={styles.portrait} resizeMode="cover" />
                <View style={styles.panelBody}>
                    <View style={styles.panelHeader}>
                        <Text style={[styles.heroName, {color: colors.gold}]}>
                            {names[heroCardId(view.you.hero.id)] ?? view.you.hero.id}
                        </Text>
                        <View style={styles.manaRow}>
                            {Array.from({length: Math.max(view.you.max_mana, view.you.mana)}).map((_, index) => (
                                <View
                                    key={index}
                                    style={[
                                        styles.manaPip,
                                        {backgroundColor: index < view.you.mana ? '#3ec6e0' : hexAlpha(colors.text, 0.15)},
                                    ]}
                                />
                            ))}
                            <Text style={[styles.manaText, {color: '#3ec6e0'}]}>{view.you.mana}</Text>
                        </View>
                    </View>
                    {healthBar(view.you.hero.hp, view.you.hero.max_hp, view.you.hero.armor)}
                    {statusChips(view.you.statuses)}
                    <View style={styles.powers}>
                        <TouchableOpacity
                            disabled={!view.you.ability_usable}
                            onPress={() => setPower('ability')}
                            style={[styles.powerBtn, {backgroundColor: colors.accent, opacity: view.you.ability_usable ? 1 : 0.4}]}
                        >
                            <Ionicons name="color-wand-outline" size={13} color="#fff" />
                            <Text style={styles.powerBtnText}>{`${t('arcana.ability')} · ${view.you.ability_cost}`}</Text>
                        </TouchableOpacity>
                        <TouchableOpacity
                            disabled={!view.you.ultimate_usable}
                            onPress={() => setPower('ultimate')}
                            style={[
                                styles.powerBtn,
                                {
                                    backgroundColor: view.you.hero.ultimate_ready ? colors.gold : colors.accent,
                                    opacity: view.you.ultimate_usable ? 1 : 0.4,
                                },
                            ]}
                        >
                            <Ionicons name="star-outline" size={13} color={view.you.hero.ultimate_ready ? colors.bg : '#fff'} />
                            <Text
                                style={[styles.powerBtnText, {color: view.you.hero.ultimate_ready ? colors.bg : '#fff'}]}
                            >
                                {`${t('arcana.ultimate')} · ${Math.min(
                                    view.you.hero.id === 'death' ? view.you.hero.souls : view.you.hero.charge,
                                    view.you.hero.ultimate_cost,
                                )}/${view.you.hero.ultimate_cost}`}
                            </Text>
                        </TouchableOpacity>
                    </View>
                </View>
            </View>

            <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                style={styles.hand}
                contentContainerStyle={styles.handRow}
            >
                {view.you.hand.map((card) =>
                    cardTile(card, {
                        width: 88,
                        dimmed: myTurn && !card.playable,
                        onPress: () => setDetail({card, fromHand: true}),
                    }),
                )}
            </ScrollView>

            <View style={styles.bottomBar}>
                <TouchableOpacity style={[styles.menuBtn, {backgroundColor: colors.panel}]} onPress={() => setMenuOpen(true)}>
                    <Ionicons name="chatbubble-ellipses-outline" size={20} color={colors.text} />
                </TouchableOpacity>
                <TouchableOpacity
                    disabled={!myTurn}
                    onPress={arcana.endTurn}
                    style={[styles.endTurn, {backgroundColor: myTurn ? colors.gold : colors.panel}]}
                >
                    <Text style={[styles.endTurnText, {color: myTurn ? colors.bg : colors.muted}]}>
                        {t('arcana.endTurn')}
                    </Text>
                </TouchableOpacity>
            </View>

            {view.phase === 'mulligan' ? renderMulligan() : null}
            {stage === 'finished' ? renderResult() : null}
            {renderDetailSheet()}
            {renderPowerSheet()}

            <Modal visible={menuOpen} transparent animationType="fade" onRequestClose={() => setMenuOpen(false)}>
                <Pressable style={styles.backdrop} onPress={() => setMenuOpen(false)} />
                <View style={[styles.sheet, {backgroundColor: colors.panel}]}>
                    <ScrollView contentContainerStyle={styles.sheetContent}>
                        <Text style={[styles.sheetTitle, {color: colors.gold}]}>{t('arcana.emote')}</Text>
                        {(arcana.catalog?.emotes ?? []).map((emote) => (
                            <TouchableOpacity
                                key={emote}
                                style={[styles.menuRow, {borderColor: hexAlpha(colors.gold, 0.2)}]}
                                onPress={() => {
                                    arcana.sendEmote(emote);
                                    setMenuOpen(false);
                                }}
                            >
                                <Text style={{color: colors.text, fontSize: 16}}>{t(`arcana.emotes.${emote}`)}</Text>
                            </TouchableOpacity>
                        ))}
                        <TouchableOpacity
                            style={[styles.menuRow, {borderColor: hexAlpha(colors.danger, 0.4)}]}
                            onPress={() => {
                                setMenuOpen(false);
                                arcana.surrender();
                            }}
                        >
                            <Text style={{color: colors.danger, fontSize: 16, fontWeight: '700'}}>
                                {t('arcana.surrender')}
                            </Text>
                        </TouchableOpacity>
                    </ScrollView>
                </View>
            </Modal>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {flex: 1, paddingHorizontal: 12, gap: 8},
    searching: {flex: 1, alignItems: 'center', justifyContent: 'center', gap: 16, padding: 24},
    searchingText: {fontSize: 18, fontWeight: '600'},
    searchingError: {fontSize: 14, textAlign: 'center'},
    cancel: {marginTop: 12},
    cancelText: {fontSize: 16, fontWeight: '700'},
    panel: {flexDirection: 'row', gap: 10, borderRadius: 16, borderWidth: 2, padding: 10},
    portrait: {width: 52, height: 78, borderRadius: 8},
    panelBody: {flex: 1, gap: 5},
    panelHeader: {flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center'},
    heroName: {fontSize: 14, fontWeight: '700'},
    counters: {flexDirection: 'row', alignItems: 'center', gap: 4},
    counterText: {fontSize: 11, fontWeight: '600'},
    healthWrap: {gap: 3},
    healthTrack: {height: 10, borderRadius: 5, overflow: 'hidden'},
    healthFill: {height: 10, borderRadius: 5},
    healthLabels: {flexDirection: 'row', alignItems: 'center', gap: 6},
    healthText: {fontSize: 12, fontWeight: '700'},
    statusRow: {gap: 5},
    statusChip: {flexDirection: 'row', alignItems: 'center', gap: 3, paddingHorizontal: 7, paddingVertical: 3, borderRadius: 999},
    statusText: {fontSize: 11, fontWeight: '700'},
    deckLabel: {fontSize: 11, fontWeight: '600'},
    center: {flex: 1, gap: 8},
    turnRow: {flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center'},
    turnText: {fontSize: 17, fontWeight: '700'},
    timer: {flexDirection: 'row', alignItems: 'center', gap: 4},
    timerText: {fontSize: 17, fontWeight: '700'},
    logBox: {flex: 1, flexDirection: 'row', gap: 10, borderRadius: 14, padding: 10},
    lastPlayed: {gap: 6},
    logLines: {flex: 1, gap: 4},
    logRow: {flexDirection: 'row', justifyContent: 'space-between', gap: 8},
    logText: {flex: 1, fontSize: 12},
    logAmount: {fontSize: 12, fontWeight: '700'},
    emote: {paddingHorizontal: 16, paddingVertical: 10, borderRadius: 999},
    emoteText: {fontSize: 15, fontWeight: '700'},
    banner: {fontSize: 13, fontWeight: '600', textAlign: 'center'},
    manaRow: {flexDirection: 'row', alignItems: 'center', gap: 3},
    manaPip: {width: 9, height: 9, borderRadius: 5},
    manaText: {fontSize: 12, fontWeight: '700', marginLeft: 2},
    powers: {flexDirection: 'row', gap: 8},
    powerBtn: {flex: 1, flexDirection: 'row', gap: 5, alignItems: 'center', justifyContent: 'center', borderRadius: 10, paddingVertical: 8},
    powerBtnText: {color: '#fff', fontSize: 12, fontWeight: '700'},
    hand: {flexGrow: 0, height: 178},
    handRow: {gap: 8, paddingVertical: 6, alignItems: 'flex-start'},
    tile: {alignItems: 'center', gap: 4, borderRadius: 12, borderWidth: 2, padding: 4},
    reversed: {transform: [{rotate: '180deg'}]},
    costBadge: {position: 'absolute', top: -6, left: -6, width: 26, height: 26, borderRadius: 13, alignItems: 'center', justifyContent: 'center'},
    costText: {color: '#fff', fontSize: 13, fontWeight: '800'},
    reversedBadge: {position: 'absolute', top: -4, right: -4, borderRadius: 999, padding: 2},
    tileName: {fontSize: 11, fontWeight: '600', textAlign: 'center'},
    bottomBar: {flexDirection: 'row', gap: 10, paddingBottom: 6},
    menuBtn: {width: 52, height: 48, borderRadius: 14, alignItems: 'center', justifyContent: 'center'},
    endTurn: {flex: 1, height: 48, borderRadius: 14, alignItems: 'center', justifyContent: 'center'},
    endTurnText: {fontSize: 17, fontWeight: '700'},
    overlay: {position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 14},
    overlayTitle: {fontSize: 24, fontWeight: '800'},
    overlayHint: {fontSize: 15, lineHeight: 21, textAlign: 'center'},
    mulliganGrid: {flexDirection: 'row', flexWrap: 'wrap', justifyContent: 'center', gap: 8},
    resultTitle: {fontSize: 30, fontWeight: '800'},
    deckPromo: {flexDirection: 'row', alignItems: 'center', gap: 12, borderRadius: 14, padding: 12, alignSelf: 'stretch'},
    deckCover: {width: 44, height: 66, borderRadius: 6},
    deckPromoText: {flex: 1, gap: 2},
    deckPromoTitle: {fontSize: 14, fontWeight: '700'},
    deckPromoLink: {fontSize: 12, fontWeight: '700'},
    backdrop: {position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.6)'},
    sheet: {position: 'absolute', left: 0, right: 0, bottom: 0, maxHeight: '85%', borderTopLeftRadius: 20, borderTopRightRadius: 20},
    sheetContent: {padding: 20, gap: 12, alignItems: 'center'},
    sheetArt: {width: 170, height: 255, borderRadius: 12},
    sheetTitle: {fontSize: 22, fontWeight: '800', textAlign: 'center'},
    sheetMeta: {fontSize: 14, fontWeight: '600', textAlign: 'center'},
    sheetSection: {fontSize: 15, fontWeight: '700'},
    effects: {borderRadius: 12, padding: 12, gap: 6, alignSelf: 'stretch'},
    effectTitle: {fontSize: 11, fontWeight: '800', textTransform: 'uppercase'},
    effectLine: {fontSize: 14, lineHeight: 20},
    targets: {gap: 10, alignSelf: 'stretch'},
    targetPool: {gap: 6},
    primary: {borderRadius: 14, paddingVertical: 14, alignItems: 'center'},
    primaryText: {color: '#fff', fontSize: 16, fontWeight: '700'},
    menuRow: {borderWidth: 1, borderRadius: 12, padding: 14, alignSelf: 'stretch', alignItems: 'center'},
});
