import React, {useMemo} from 'react';
import {
    SectionList,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {CompositeNavigationProp, useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {BottomTabNavigationProp} from '@react-navigation/bottom-tabs';
import {useTranslation} from 'react-i18next';
import {SPREADS} from '../data';
import {AppTabsParamList, HomeStackParamList, RootStackParamList} from '../navigation/types';
import type {Spread} from '../entities';

type Navigation = CompositeNavigationProp<
    NativeStackNavigationProp<HomeStackParamList, 'SpreadCatalog'>,
    CompositeNavigationProp<
        BottomTabNavigationProp<AppTabsParamList>,
        NativeStackNavigationProp<RootStackParamList>
    >
>;

type Section = {
    title: string;
    data: Spread[];
};

const categoryKey = (category: Spread['category']) => `spread.category.${category}`;

export const SpreadCatalogScreen = () => {
    const navigation = useNavigation<Navigation>();
    const {t} = useTranslation();

    const sections = useMemo<Section[]>(() => {
        const grouped = SPREADS.reduce<Record<string, Spread[]>>((acc, spread) => {
            const key = categoryKey(spread.category);
            acc[key] = acc[key] ? [...acc[key], spread] : [spread];
            return acc;
        }, {});

        return Object.entries(grouped).map(([title, data]) => ({
            title,
            data: data.sort((a, b) => a.minCards - b.minCards),
        }));
    }, []);

    const startSpread = (spread: Spread) => {
        navigation.navigate('Reading', {spreadId: spread.id, deckId: 'rws'});
    };

    const renderItem = ({item}: {item: Spread}) => (
        <TouchableOpacity
            style={[styles.card, item.premium && styles.cardPremium]}
            onPress={() => startSpread(item)}
        >
            <View style={styles.cardHeader}>
                <Text style={styles.cardTitle}>{t(item.nameKey)}</Text>
                {item.premium && <Text style={styles.badge}>{t('spread.premium')}</Text>}
            </View>
            <Text style={styles.cardDescription}>{t(item.descriptionKey)}</Text>
            <View style={styles.metaRow}>
                <Text style={styles.meta}>{t('spread.cards', {count: item.maxCards})}</Text>
                <Text style={styles.meta}>{t(categoryKey(item.category))}</Text>
            </View>
        </TouchableOpacity>
    );

    return (
        <SafeAreaView style={styles.safe}>
            <SectionList
                sections={sections}
                keyExtractor={(item) => item.id}
                renderItem={renderItem}
                renderSectionHeader={({section}) => (
                    <Text style={styles.sectionTitle}>{t(section.title)}</Text>
                )}
                contentContainerStyle={styles.listContent}
            />
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    listContent: {
        padding: 20,
        paddingBottom: 40,
        gap: 16,
    },
    sectionTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#f4d386',
        marginBottom: 12,
        marginTop: 24,
    },
    card: {
        backgroundColor: 'rgba(12,10,20,0.9)',
        borderRadius: 18,
        padding: 18,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.25)',
        marginBottom: 12,
    },
    cardPremium: {
        borderColor: 'rgba(108,92,231,0.5)',
        backgroundColor: 'rgba(108,92,231,0.12)',
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 10,
    },
    cardTitle: {
        color: '#f7f4ea',
        fontSize: 18,
        fontWeight: '600',
        flex: 1,
        marginRight: 12,
    },
    cardDescription: {
        color: '#f7f4ea',
        opacity: 0.75,
        fontSize: 14,
        lineHeight: 20,
        marginBottom: 12,
    },
    metaRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
    },
    meta: {
        color: '#f4d386',
        fontSize: 12,
        textTransform: 'uppercase',
        letterSpacing: 1,
    },
    badge: {
        color: '#fff',
        backgroundColor: '#6c5ce7',
        paddingHorizontal: 10,
        paddingVertical: 4,
        borderRadius: 12,
        fontSize: 12,
        fontWeight: '700',
    },
});
