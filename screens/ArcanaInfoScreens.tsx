import React, {useCallback, useEffect, useState} from 'react';
import {ActivityIndicator, Image, ScrollView, StyleSheet, Text, TouchableOpacity, View} from 'react-native';
import {useTranslation} from 'react-i18next';
import {Ionicons} from '@expo/vector-icons';
import {useArcana} from '../providers/ArcanaProvider';
import {useAppColors, useDeckShop} from '../providers/DeckShopProvider';
import {useSettings} from '../providers/SettingsProvider';
import {hexAlpha} from '../theme/appColors';
import {heroCardId, type ArcanaMatchSummary} from '../features/arcanaApi';
import {
    ARCANA_STATUS_ORDER,
    arcanaCardImageFile,
    arcanaCardNames,
    arcanaStatusIcon,
} from '../features/arcanaText';

export const ArcanaRulesScreen = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const {catalog} = useArcana();
    const rules = catalog?.rules;
    const paragraphs = t('arcana.rulesText', {
        returnObjects: true,
        turn: rules?.turn_seconds ?? 40,
        reconnect: rules?.reconnect_seconds ?? 45,
    }) as unknown as string[];

    return (
        <ScrollView contentContainerStyle={styles.content}>
            {(Array.isArray(paragraphs) ? paragraphs : []).map((paragraph, index) => (
                <Text key={index} style={[styles.paragraph, {color: colors.text}]}>
                    {paragraph}
                </Text>
            ))}
            <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('arcana.statusTitle')}</Text>
            {ARCANA_STATUS_ORDER.map((name) => (
                <View key={name} style={styles.statusRow}>
                    <Ionicons name={arcanaStatusIcon(name) as never} size={16} color={colors.gold} />
                    <View style={styles.statusText}>
                        <Text style={[styles.statusName, {color: colors.gold}]}>{t(`arcana.status.${name}`)}</Text>
                        <Text style={[styles.statusHint, {color: colors.text}]}>{t(`arcana.statusHint.${name}`)}</Text>
                    </View>
                </View>
            ))}
        </ScrollView>
    );
};

export const ArcanaHeroesScreen = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const {catalog} = useArcana();
    const {settings} = useSettings();
    const {selectedSlug, faceSource} = useDeckShop();
    const names = arcanaCardNames(settings.language);

    return (
        <ScrollView contentContainerStyle={styles.content}>
            {(catalog?.heroes ?? []).map((hero) => {
                const cardId = heroCardId(hero.id);
                const art = faceSource(arcanaCardImageFile(cardId), selectedSlug);
                return (
                    <View
                        key={hero.id}
                        style={[styles.heroCard, {backgroundColor: colors.panel, borderColor: hexAlpha(colors.gold, 0.2)}]}
                    >
                        {art ? <Image source={art} style={styles.heroArt} resizeMode="cover" /> : null}
                        <View style={styles.heroBody}>
                            <Text style={[styles.heroName, {color: colors.gold}]}>{names[cardId] ?? hero.id}</Text>
                            <View style={styles.heroHp}>
                                <Ionicons name="heart" size={12} color={colors.danger} />
                                <Text style={[styles.heroHpText, {color: colors.danger}]}>
                                    {t('arcana.hp', {count: hero.hp})}
                                </Text>
                            </View>
                            {(
                                [
                                    [t('arcana.passive'), `arcana.heroText.${hero.id}.passive`],
                                    [
                                        `${t('arcana.ability')} · ${t('arcana.cost', {cost: hero.ability.cost})}`,
                                        `arcana.heroText.${hero.id}.ability`,
                                    ],
                                    [t('arcana.ultimate'), `arcana.heroText.${hero.id}.ultimate`],
                                ] as const
                            ).map(([title, key]) => (
                                <View key={key} style={styles.power}>
                                    <Text style={[styles.powerTitle, {color: colors.accent}]}>{title}</Text>
                                    <Text style={[styles.powerText, {color: colors.text}]}>{t(key)}</Text>
                                </View>
                            ))}
                        </View>
                    </View>
                );
            })}
        </ScrollView>
    );
};

export const ArcanaMatchHistoryScreen = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const {matchHistory} = useArcana();
    const {settings} = useSettings();
    const {decks, titleOf} = useDeckShop();
    const [matches, setMatches] = useState<ArcanaMatchSummary[] | null>(null);
    const [failed, setFailed] = useState(false);
    const names = arcanaCardNames(settings.language);

    const load = useCallback(async () => {
        try {
            setMatches(await matchHistory());
            setFailed(false);
        } catch {
            setFailed(true);
        }
    }, [matchHistory]);

    useEffect(() => {
        load().catch(() => {});
    }, [load]);

    const deckTitle = (slug?: string) => {
        const deck = decks.find((item) => item.slug === slug && !item.bundled);
        return deck ? titleOf(deck) : null;
    };

    if (failed) {
        return (
            <View style={styles.center}>
                <Text style={[styles.paragraph, {color: colors.danger}]}>{t('arcana.historyError')}</Text>
                <TouchableOpacity onPress={load}>
                    <Text style={[styles.retry, {color: colors.gold}]}>{t('arcana.retry')}</Text>
                </TouchableOpacity>
            </View>
        );
    }
    if (!matches) {
        return (
            <View style={styles.center}>
                <ActivityIndicator color={colors.accent} />
            </View>
        );
    }
    if (matches.length === 0) {
        return (
            <View style={styles.center}>
                <Text style={[styles.paragraph, {color: colors.muted, textAlign: 'center'}]}>
                    {t('arcana.historyEmpty')}
                </Text>
            </View>
        );
    }

    return (
        <ScrollView contentContainerStyle={styles.content}>
            {matches.map((match) => {
                const tone = match.result === 'win' ? colors.gold : match.result === 'loss' ? colors.danger : colors.muted;
                const opponentDeck = deckTitle(match.opponent_deck);
                return (
                    <View
                        key={match.id}
                        style={[styles.matchCard, {backgroundColor: colors.panel, borderColor: hexAlpha(colors.gold, 0.2)}]}
                    >
                        <View style={styles.matchHeader}>
                            <Text style={[styles.matchResult, {color: tone}]}>{t(`arcana.result.${match.result}`)}</Text>
                            <Text style={[styles.matchDate, {color: colors.muted}]}>
                                {new Date(match.finished_at).toLocaleString(settings.language)}
                            </Text>
                        </View>
                        <Text style={{color: colors.text}}>
                            {`${names[heroCardId(match.hero)] ?? match.hero} — ${
                                names[heroCardId(match.opponent_hero)] ?? match.opponent_hero
                            }`}
                        </Text>
                        {opponentDeck ? (
                            <Text style={[styles.matchMeta, {color: colors.accent}]}>
                                {t('arcana.opponentDeck', {deck: opponentDeck})}
                            </Text>
                        ) : null}
                        <Text style={[styles.matchMeta, {color: colors.muted}]}>
                            {[
                                t(`arcana.reason.${match.reason}`),
                                t('arcana.turns', {count: match.turns}),
                                t('arcana.minutes', {count: Math.max(1, Math.round(match.duration_ms / 60000))}),
                            ].join(' · ')}
                        </Text>
                    </View>
                );
            })}
        </ScrollView>
    );
};

const styles = StyleSheet.create({
    content: {padding: 20, gap: 12},
    center: {flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 12},
    paragraph: {fontSize: 15, lineHeight: 22},
    sectionTitle: {fontSize: 17, fontWeight: '700', marginTop: 8},
    statusRow: {flexDirection: 'row', gap: 10, alignItems: 'flex-start'},
    statusText: {flex: 1, gap: 2},
    statusName: {fontSize: 14, fontWeight: '700'},
    statusHint: {fontSize: 14, lineHeight: 20},
    retry: {fontSize: 15, fontWeight: '700'},
    heroCard: {flexDirection: 'row', gap: 14, borderRadius: 16, borderWidth: 1, padding: 14},
    heroArt: {width: 90, height: 135, borderRadius: 8},
    heroBody: {flex: 1, gap: 8},
    heroName: {fontSize: 17, fontWeight: '700'},
    heroHp: {flexDirection: 'row', alignItems: 'center', gap: 4},
    heroHpText: {fontSize: 12, fontWeight: '700'},
    power: {gap: 2},
    powerTitle: {fontSize: 11, fontWeight: '800', textTransform: 'uppercase'},
    powerText: {fontSize: 14, lineHeight: 20},
    matchCard: {borderRadius: 14, borderWidth: 1, padding: 14, gap: 6},
    matchHeader: {flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center'},
    matchResult: {fontSize: 16, fontWeight: '700'},
    matchDate: {fontSize: 12},
    matchMeta: {fontSize: 12, lineHeight: 18},
});
