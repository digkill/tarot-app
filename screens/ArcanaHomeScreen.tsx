import React, {useCallback, useEffect} from 'react';
import {
    Image,
    ScrollView,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useFocusEffect, useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {Ionicons} from '@expo/vector-icons';
import {useArcana} from '../providers/ArcanaProvider';
import {useAppColors, useDeckShop} from '../providers/DeckShopProvider';
import {useSettings} from '../providers/SettingsProvider';
import {hexAlpha} from '../theme/appColors';
import {heroCardId} from '../features/arcanaApi';
import {arcanaCardImageFile, arcanaCardNames} from '../features/arcanaText';
import type {ArcanaStackParamList, RootStackParamList} from '../navigation/types';

type Navigation = NativeStackNavigationProp<ArcanaStackParamList, 'ArcanaHome'>;
type RootNavigation = NativeStackNavigationProp<RootStackParamList>;

/** The Arcana Clash lobby: hero choice, Battle, and the reference screens. */
export const ArcanaHomeScreen = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    const navigation = useNavigation<Navigation>();
    // The battlefield lives on the root stack, so it covers the tab bar.
    const rootNavigation = useNavigation<RootNavigation>();
    const {settings} = useSettings();
    const {selectedSlug, faceSource, backSource} = useDeckShop();
    const {
        catalog,
        catalogFailed,
        loadCatalog,
        hero,
        setHero,
        battle,
        stage,
        resumableMatch,
        resume,
        enter,
        leave,
    } = useArcana();

    useEffect(() => {
        if (!catalog) {
            loadCatalog().catch(() => {});
        }
    }, [catalog, loadCatalog]);

    useFocusEffect(
        useCallback(() => {
            enter();
            return () => leave();
        }, [enter, leave]),
    );

    // The match screen owns the match; the lobby only starts it.
    useEffect(() => {
        if (stage === 'searching' || stage === 'match' || stage === 'finished') {
            rootNavigation.navigate('ArcanaBattle');
        }
    }, [rootNavigation, stage]);

    const names = arcanaCardNames(settings.language);
    const heroName = (id: string) => names[heroCardId(id)] ?? id;

    const heroChip = (id: string | null, title: string, cardId: string | null, hp: number | null) => {
        const active = hero === id;
        const source = cardId ? faceSource(arcanaCardImageFile(cardId), selectedSlug) : backSource(selectedSlug);
        return (
            <TouchableOpacity
                key={id ?? 'random'}
                style={[
                    styles.heroChip,
                    {
                        backgroundColor: active ? hexAlpha(colors.accent, 0.25) : colors.panel,
                        borderColor: active ? colors.accent : hexAlpha(colors.gold, 0.2),
                    },
                ]}
                onPress={() => setHero(id)}
            >
                {source ? <Image source={source} style={styles.heroArt} resizeMode="cover" /> : null}
                <Text numberOfLines={1} style={[styles.heroName, {color: active ? colors.gold : colors.text}]}>
                    {title}
                </Text>
                {hp === null ? null : (
                    <View style={styles.heroHp}>
                        <Ionicons name="heart" size={11} color={colors.danger} />
                        <Text style={[styles.heroHpText, {color: colors.muted}]}>{hp}</Text>
                    </View>
                )}
            </TouchableOpacity>
        );
    };

    const link = (label: string, icon: keyof typeof Ionicons.glyphMap, screen: keyof ArcanaStackParamList) => (
        <TouchableOpacity
            style={[styles.link, {backgroundColor: colors.panel, borderColor: hexAlpha(colors.gold, 0.2)}]}
            onPress={() => navigation.navigate(screen as never)}
        >
            <Ionicons name={icon} size={18} color={colors.text} />
            <Text style={[styles.linkText, {color: colors.text}]}>{label}</Text>
            <Ionicons name="chevron-forward" size={18} color={colors.muted} />
        </TouchableOpacity>
    );

    return (
        <SafeAreaView style={styles.safe} edges={['top']}>
            <ScrollView contentContainerStyle={styles.content}>
                <Text style={[styles.title, {color: colors.text}]}>{t('arcana.title')}</Text>
                <View style={styles.tagline}>
                    <Ionicons name="flash" size={26} color={colors.gold} />
                    <Text style={[styles.taglineText, {color: colors.text}]}>{t('arcana.tagline')}</Text>
                </View>

                {resumableMatch ? (
                    <TouchableOpacity style={[styles.resume, {backgroundColor: colors.gold}]} onPress={resume}>
                        <Ionicons name="arrow-forward-circle" size={20} color={colors.bg} />
                        <Text style={[styles.resumeText, {color: colors.bg}]}>{t('arcana.resume')}</Text>
                    </TouchableOpacity>
                ) : null}

                <Text style={[styles.sectionTitle, {color: colors.gold}]}>{t('arcana.chooseHero')}</Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.heroRow}>
                    {heroChip(null, t('arcana.randomHero'), null, null)}
                    {(catalog?.heroes ?? []).map((item) =>
                        heroChip(item.id, heroName(item.id), heroCardId(item.id), item.hp),
                    )}
                </ScrollView>

                <TouchableOpacity
                    style={[styles.battle, {backgroundColor: colors.accent, opacity: catalog ? 1 : 0.5}]}
                    disabled={!catalog}
                    onPress={() => battle(selectedSlug)}
                >
                    <Text style={styles.battleText}>{t('arcana.battle')}</Text>
                </TouchableOpacity>

                {catalogFailed && !catalog ? (
                    <>
                        <Text style={[styles.error, {color: colors.danger}]}>{t('arcana.connectionFailed')}</Text>
                        <TouchableOpacity onPress={() => loadCatalog()}>
                            <Text style={[styles.retry, {color: colors.gold}]}>{t('arcana.retry')}</Text>
                        </TouchableOpacity>
                    </>
                ) : null}

                <View style={styles.links}>
                    {link(t('arcana.rules'), 'book-outline', 'ArcanaRules')}
                    {link(t('arcana.heroes'), 'people-outline', 'ArcanaHeroes')}
                    {link(t('arcana.history'), 'time-outline', 'ArcanaMatchHistory')}
                </View>
            </ScrollView>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {flex: 1},
    content: {padding: 20, gap: 16},
    title: {fontSize: 30, fontWeight: '800'},
    tagline: {flexDirection: 'row', gap: 12, alignItems: 'flex-start'},
    taglineText: {flex: 1, fontSize: 15, lineHeight: 21},
    sectionTitle: {fontSize: 16, fontWeight: '700'},
    heroRow: {gap: 12, paddingVertical: 4},
    heroChip: {width: 104, borderRadius: 14, borderWidth: 1, padding: 8, alignItems: 'center', gap: 6},
    heroArt: {width: 84, height: 126, borderRadius: 8},
    heroName: {fontSize: 12, fontWeight: '700', textAlign: 'center'},
    heroHp: {flexDirection: 'row', alignItems: 'center', gap: 3},
    heroHpText: {fontSize: 11, fontWeight: '600'},
    battle: {borderRadius: 16, paddingVertical: 16, alignItems: 'center'},
    battleText: {color: '#fff', fontSize: 18, fontWeight: '700'},
    resume: {flexDirection: 'row', gap: 10, alignItems: 'center', justifyContent: 'center', borderRadius: 16, paddingVertical: 14},
    resumeText: {fontSize: 16, fontWeight: '700'},
    error: {fontSize: 14, lineHeight: 20},
    retry: {fontSize: 15, fontWeight: '700', textAlign: 'center'},
    links: {gap: 10},
    link: {flexDirection: 'row', alignItems: 'center', gap: 12, borderRadius: 14, borderWidth: 1, padding: 16},
    linkText: {flex: 1, fontSize: 16, fontWeight: '600'},
});
