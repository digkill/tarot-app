import React, {useMemo, useState} from 'react';
import {
    Alert,
    FlatList,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {useHistory} from '../providers/HistoryProvider';
import {useSettings} from '../providers/SettingsProvider';
import {SPREADS} from '../data';
import {RootStackParamList} from '../navigation/types';

type Navigation = NativeStackNavigationProp<RootStackParamList, 'Interpretation'>;

const FILTERS = ['all', 'favorites'] as const;
type Filter = (typeof FILTERS)[number];

export const HistoryListScreen = () => {
    const {readings, deleteReading, clearAll, toggleFavorite} = useHistory();
    const {settings} = useSettings();
    const navigation = useNavigation<Navigation>();
    const {t} = useTranslation();
    const [filter, setFilter] = useState<Filter>('all');

    const filtered = useMemo(() => {
        if (filter === 'favorites') {
            return readings.filter((reading) => reading.favorite);
        }
        return readings;
    }, [filter, readings]);

    const renderItem = ({item}: {item: typeof readings[number]}) => {
        const spread = SPREADS.find((spreadItem) => spreadItem.id === item.spreadId);
        return (
            <TouchableOpacity
                style={styles.card}
                onPress={() => navigation.navigate('Interpretation', {readingId: item.id})}
            >
                <View style={styles.cardHeader}>
                    <Text style={styles.cardTitle}>{spread ? t(spread.nameKey) : item.spreadId}</Text>
                    <TouchableOpacity
                        onPress={() => toggleFavorite(item.id)}
                        style={styles.favoriteBtn}
                    >
                        <Text style={styles.favoriteIcon}>{item.favorite ? '★' : '☆'}</Text>
                    </TouchableOpacity>
                </View>
                <Text style={styles.cardDate}>
                    {new Date(item.drawnAt).toLocaleString(settings.language)}
                </Text>
                <Text style={styles.cardSummary} numberOfLines={3}>
                    {item.summaryText}
                </Text>
                <TouchableOpacity
                    onPress={() => confirmDelete(item.id)}
                    style={styles.deleteBtn}
                >
                    <Text style={styles.deleteText}>{t('history.delete')}</Text>
                </TouchableOpacity>
            </TouchableOpacity>
        );
    };

    const confirmDelete = (id: string) => {
        Alert.alert(t('history.deleteTitle'), t('history.deleteDescription'), [
            {text: t('history.cancel'), style: 'cancel'},
            {
                text: t('history.confirm'),
                style: 'destructive',
                onPress: () => deleteReading(id),
            },
        ]);
    };

    const confirmClear = () => {
        Alert.alert(t('history.clearTitle'), t('history.clearDescription'), [
            {text: t('history.cancel'), style: 'cancel'},
            {
                text: t('history.confirm'),
                style: 'destructive',
                onPress: () => clearAll(),
            },
        ]);
    };

    return (
        <SafeAreaView style={styles.safe}>
            <View style={styles.filterRow}>
                {FILTERS.map((item) => (
                    <TouchableOpacity
                        key={item}
                        style={[styles.filterButton, filter === item && styles.filterButtonActive]}
                        onPress={() => setFilter(item)}
                    >
                        <Text
                            style={[styles.filterText, filter === item && styles.filterTextActive]}
                        >
                            {t(`history.filter.${item}`)}
                        </Text>
                    </TouchableOpacity>
                ))}
                <TouchableOpacity style={styles.clearButton} onPress={confirmClear}>
                    <Text style={styles.clearText}>{t('history.clearAll')}</Text>
                </TouchableOpacity>
            </View>

            <FlatList
                data={filtered}
                keyExtractor={(item) => item.id}
                renderItem={renderItem}
                contentContainerStyle={styles.listContent}
                ListEmptyComponent={() => (
                    <View style={styles.emptyState}>
                        <Text style={styles.emptyTitle}>{t('history.emptyTitle')}</Text>
                        <Text style={styles.emptySubtitle}>{t('history.emptySubtitle')}</Text>
                    </View>
                )}
            />
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    safe: {
        flex: 1,
        backgroundColor: '#040307',
    },
    filterRow: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 16,
        paddingVertical: 12,
        gap: 8,
    },
    filterButton: {
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: 16,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.3)',
    },
    filterButtonActive: {
        backgroundColor: '#6c5ce7',
        borderColor: '#6c5ce7',
    },
    filterText: {
        color: '#f7f4ea',
        fontWeight: '500',
    },
    filterTextActive: {
        color: '#fff',
    },
    clearButton: {
        marginLeft: 'auto',
    },
    clearText: {
        color: '#f67280',
        fontWeight: '600',
    },
    listContent: {
        padding: 16,
        paddingBottom: 60,
        gap: 16,
    },
    card: {
        backgroundColor: 'rgba(12,10,20,0.9)',
        borderRadius: 20,
        padding: 18,
        borderWidth: 1,
        borderColor: 'rgba(244,211,134,0.2)',
        gap: 10,
    },
    cardHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    cardTitle: {
        color: '#f4d386',
        fontSize: 18,
        fontWeight: '600',
        flex: 1,
    },
    favoriteBtn: {
        paddingHorizontal: 10,
        paddingVertical: 6,
    },
    favoriteIcon: {
        fontSize: 18,
        color: '#f4d386',
    },
    cardDate: {
        color: '#f7f4ea',
        opacity: 0.6,
        fontSize: 13,
    },
    cardSummary: {
        color: '#f7f4ea',
        lineHeight: 18,
    },
    deleteBtn: {
        alignSelf: 'flex-start',
        paddingHorizontal: 12,
        paddingVertical: 6,
        borderRadius: 14,
        backgroundColor: 'rgba(246,114,128,0.15)',
    },
    deleteText: {
        color: '#f67280',
        fontWeight: '600',
    },
    emptyState: {
        marginTop: 80,
        alignItems: 'center',
        gap: 8,
    },
    emptyTitle: {
        color: '#f7f4ea',
        fontSize: 18,
        fontWeight: '600',
    },
    emptySubtitle: {
        color: '#f7f4ea',
        opacity: 0.6,
    },
});
