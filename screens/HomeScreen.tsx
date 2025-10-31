import React, {useMemo} from 'react';
import {
    Image,
    ScrollView,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {CompositeNavigationProp, useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {BottomTabNavigationProp} from '@react-navigation/bottom-tabs';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useTranslation} from 'react-i18next';
import {AppTabsParamList, HomeStackParamList, RootStackParamList} from '../navigation/types';
import {SPREADS, BASIC_SPREAD_IDS} from '../data';
import {useHistory} from '../providers/HistoryProvider';
import {useSettings} from '../providers/SettingsProvider';

type Navigation = CompositeNavigationProp<
    NativeStackNavigationProp<HomeStackParamList, 'Home'>,
    CompositeNavigationProp<
        BottomTabNavigationProp<AppTabsParamList>,
        NativeStackNavigationProp<RootStackParamList>
    >
>;

const FEATURED_SPREADS = SPREADS.filter((spread) =>
    (BASIC_SPREAD_IDS as readonly string[]).includes(spread.id),
);

export const HomeScreen = () => {
    const navigation = useNavigation<Navigation>();
    const {t} = useTranslation();
    const {readings} = useHistory();
    const {settings} = useSettings();

    const recentReading = readings[0];
    const recentSpread = recentReading ? SPREADS.find((spread) => spread.id === recentReading.spreadId) : null;
    const recentSummary = recentReading?.aiInsights?.summary ?? recentReading?.summaryText;
    const featured = FEATURED_SPREADS;

    const greeting = useMemo(() => {
        const hour = new Date().getHours();
        if (hour < 12) return t('home.greeting.morning');
        if (hour < 18) return t('home.greeting.afternoon');
        return t('home.greeting.evening');
    }, [t]);

    const startSpread = (spreadId: string) => {
        navigation.navigate('Reading', {spreadId, deckId: 'rws'});
    };

    const goToCatalog = () => {
        navigation.navigate('SpreadCatalog');
    };

    const goToHistory = () => {
        navigation.navigate('History');
    };

    return (
        <SafeAreaView style={styles.safe}>
            <ScrollView contentContainerStyle={styles.container}>
                <View style={styles.hero}>
                    <Image source={require('../assets/tarot-bg.png')} style={styles.heroImage} />
                    <View style={styles.heroOverlay} />
                    <View style={styles.heroContent}>
                        <Text style={styles.heroGreeting}>{greeting}</Text>
                        <Text style={styles.heroTitle}>{t('home.hero.title')}</Text>
                        <Text style={styles.heroSubtitle}>{t('home.hero.subtitle')}</Text>
                        <TouchableOpacity style={styles.heroButton} onPress={() => startSpread('one-card')}>
                            <Text style={styles.heroButtonText}>{t('home.hero.cta')}</Text>
                        </TouchableOpacity>
                    </View>
                </View>

                <View style={styles.section}>
                    <View style={styles.sectionHeader}>
                        <Text style={styles.sectionTitle}>{t('home.quickAccess.title')}</Text>
                        <TouchableOpacity onPress={goToCatalog}>
                            <Text style={styles.sectionLink}>{t('home.quickAccess.more')}</Text>
                        </TouchableOpacity>
                    </View>
                    <View style={styles.spreadsRow}>
                        {featured.map((spread) => (
                            <TouchableOpacity
                                key={spread.id}
                                style={styles.spreadCard}
                                onPress={() => startSpread(spread.id)}
                            >
                                <Text style={styles.spreadTitle}>{t(spread.nameKey)}</Text>
                                <Text style={styles.spreadSubtitle}>
                                    {t('home.quickAccess.cardCount', {count: spread.maxCards})}
                                </Text>
                            </TouchableOpacity>
                        ))}
                    </View>
                </View>

                {recentReading && (
                    <View style={styles.section}>
                        <View style={styles.sectionHeader}>
                            <Text style={styles.sectionTitle}>{t('home.recent.title')}</Text>
                            <TouchableOpacity onPress={goToHistory}>
                                <Text style={styles.sectionLink}>{t('home.recent.viewAll')}</Text>
                            </TouchableOpacity>
                        </View>
                        <View style={styles.recentCard}>
                            <Text style={styles.recentLabel}>{t('home.recent.spread')}</Text>
                            <Text style={styles.recentValue}>
                                {recentSpread ? t(recentSpread.nameKey) : recentReading.spreadId}
                            </Text>
                            <Text style={styles.recentLabel}>{t('home.recent.date')}</Text>
                            <Text style={styles.recentValue}>
                                {new Date(recentReading.drawnAt).toLocaleString(settings.language)}
                            </Text>
                            {recentSummary ? (
                                <Text style={styles.recentSummary} numberOfLines={3}>
                                    {recentSummary}
                                </Text>
                            ) : null}
                        </View>
                    </View>
                )}

                <View style={styles.section}>
                    <Text style={styles.sectionTitle}>{t('home.education.title')}</Text>
                    <View style={styles.educationCard}>
                        <Text style={styles.educationText}>{t('home.education.body')}</Text>
                        <TouchableOpacity style={styles.educationButton} onPress={() => navigation.navigate('Decks')}>
                            <Text style={styles.educationButtonText}>{t('home.education.cta')}</Text>
                        </TouchableOpacity>
                    </View>
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
        paddingBottom: 40,
        gap: 32,
    },
    hero: {
        marginHorizontal: 20,
        height: 260,
        borderRadius: 24,
        overflow: 'hidden',
    },
    heroImage: {
        width: '100%',
        height: '100%',
    },
    heroOverlay: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(8,7,15,0.55)',
    },
    heroContent: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        padding: 24,
        justifyContent: 'space-between',
    },
    heroGreeting: {
        color: '#f7f4ea',
        fontSize: 16,
    },
    heroTitle: {
        color: '#f4d386',
        fontSize: 28,
        fontWeight: '700',
        lineHeight: 34,
    },
    heroSubtitle: {
        color: '#f7f4ea',
        fontSize: 15,
        opacity: 0.8,
        marginTop: 8,
    },
    heroButton: {
        backgroundColor: '#6c5ce7',
        alignSelf: 'flex-start',
        paddingVertical: 10,
        paddingHorizontal: 20,
        borderRadius: 20,
    },
    heroButtonText: {
        color: '#fff',
        fontWeight: '600',
        fontSize: 16,
    },
    section: {
        marginHorizontal: 20,
    },
    sectionHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 16,
    },
    sectionTitle: {
        fontSize: 20,
        fontWeight: '600',
        color: '#f7f4ea',
    },
    sectionLink: {
        color: '#6c5ce7',
        fontWeight: '600',
    },
    spreadsRow: {
        flexDirection: 'row',
        gap: 12,
        flexWrap: 'wrap',
    },
    spreadCard: {
        flexGrow: 1,
        backgroundColor: 'rgba(108,92,231,0.12)',
        borderRadius: 18,
        padding: 16,
        minWidth: '45%',
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.3)',
    },
    spreadTitle: {
        color: '#f4d386',
        fontWeight: '600',
        fontSize: 16,
        marginBottom: 6,
    },
    spreadSubtitle: {
        color: '#f7f4ea',
        opacity: 0.7,
        fontSize: 13,
    },
    recentCard: {
        backgroundColor: 'rgba(15,13,25,0.8)',
        borderRadius: 20,
        padding: 20,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.3)',
        gap: 8,
    },
    recentLabel: {
        color: '#f7f4ea',
        opacity: 0.6,
        fontSize: 12,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    recentValue: {
        color: '#f4d386',
        fontWeight: '600',
        fontSize: 16,
    },
    recentSummary: {
        color: '#f7f4ea',
        fontSize: 14,
        marginTop: 8,
        lineHeight: 20,
    },
    educationCard: {
        backgroundColor: 'rgba(108,92,231,0.12)',
        borderRadius: 18,
        padding: 20,
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.3)',
    },
    educationText: {
        color: '#f7f4ea',
        fontSize: 15,
        lineHeight: 21,
    },
    educationButton: {
        marginTop: 16,
        backgroundColor: '#f4d386',
        paddingVertical: 12,
        borderRadius: 20,
        alignSelf: 'flex-start',
        paddingHorizontal: 20,
    },
    educationButtonText: {
        color: '#08070f',
        fontWeight: '600',
    },
});
