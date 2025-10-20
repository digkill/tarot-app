import React, {useRef, useState} from 'react';
import {
    Dimensions,
    FlatList,
    Image,
    ListRenderItem,
    NativeScrollEvent,
    NativeSyntheticEvent,
    StyleSheet,
    Text,
    TouchableOpacity,
    View,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import type {NativeStackScreenProps} from '@react-navigation/native-stack';
import {useTranslation} from 'react-i18next';
import {RootStackParamList} from '../navigation/types';
import {useSettings} from '../providers/SettingsProvider';

type Slide = {
    key: string;
    title: string;
    description: string;
    image: any;
};

const {width} = Dimensions.get('window');

type Props = NativeStackScreenProps<RootStackParamList, 'Onboarding'>;

export const OnboardingScreen = ({navigation}: Props) => {
    const {t} = useTranslation();
    const {setSetting} = useSettings();
    const [index, setIndex] = useState(0);
    const listRef = useRef<FlatList<Slide>>(null);

    const slides: Slide[] = [
        {
            key: 'intro',
            title: t('onboarding.slide1.title'),
            description: t('onboarding.slide1.description'),
            image: require('../assets/onboarding-1.png'),
        },
        {
            key: 'spreads',
            title: t('onboarding.slide2.title'),
            description: t('onboarding.slide2.description'),
            image: require('../assets/onboarding-2.png'),
        },
        {
            key: 'insights',
            title: t('onboarding.slide3.title'),
            description: t('onboarding.slide3.description'),
            image: require('../assets/onboarding-3.png'),
        },
    ];

    const handleScroll = (event: NativeSyntheticEvent<NativeScrollEvent>) => {
        const nextIndex = Math.round(event.nativeEvent.contentOffset.x / width);
        setIndex(nextIndex);
    };

    const goNext = () => {
        if (index < slides.length - 1) {
            listRef.current?.scrollToIndex({index: index + 1});
        } else {
            setSetting('hasCompletedOnboarding', true).catch(() => {});
            navigation.replace('Disclaimer');
        }
    };

    const skip = () => {
        setSetting('hasCompletedOnboarding', true).catch(() => {});
        navigation.replace('Disclaimer');
    };

    const renderItem: ListRenderItem<Slide> = ({item}) => (
        <View style={[styles.slide, {width}]}>
            <Image source={item.image} style={styles.image} resizeMode="cover" />
            <Text style={styles.title}>{item.title}</Text>
            <Text style={styles.description}>{item.description}</Text>
        </View>
    );

    return (
        <SafeAreaView style={styles.container}>
            <TouchableOpacity style={styles.skip} onPress={skip}>
                <Text style={styles.skipText}>{t('onboarding.skip')}</Text>
            </TouchableOpacity>
            <FlatList
                ref={listRef}
                data={slides}
                renderItem={renderItem}
                keyExtractor={(item) => item.key}
                horizontal
                showsHorizontalScrollIndicator={false}
                pagingEnabled
                onMomentumScrollEnd={handleScroll}
            />
            <View style={styles.footer}>
                <View style={styles.dots}>
                    {slides.map((slide, i) => (
                        <View
                            key={slide.key}
                            style={[styles.dot, index === i && styles.dotActive]}
                        />
                    ))}
                </View>
                <TouchableOpacity style={styles.button} onPress={goNext}>
                    <Text style={styles.buttonText}>
                        {index === slides.length - 1 ? t('onboarding.start') : t('onboarding.next')}
                    </Text>
                </TouchableOpacity>
            </View>
        </SafeAreaView>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#050508',
    },
    slide: {
        flex: 1,
        alignItems: 'center',
        justifyContent: 'center',
        paddingHorizontal: 24,
    },
    image: {
        width: width * 0.8,
        height: width * 1.0,
        borderRadius: 24,
        marginBottom: 32,
    },
    title: {
        fontSize: 26,
        fontWeight: '700',
        color: '#f4d386',
        textAlign: 'center',
        marginBottom: 12,
    },
    description: {
        fontSize: 16,
        color: '#f7f4ea',
        textAlign: 'center',
        lineHeight: 22,
    },
    footer: {
        paddingHorizontal: 24,
        paddingBottom: 24,
    },
    dots: {
        flexDirection: 'row',
        justifyContent: 'center',
        marginBottom: 24,
        gap: 8,
    },
    dot: {
        width: 10,
        height: 10,
        borderRadius: 5,
        backgroundColor: 'rgba(255,255,255,0.2)',
    },
    dotActive: {
        backgroundColor: '#f4d386',
        width: 20,
    },
    button: {
        backgroundColor: '#6c5ce7',
        paddingVertical: 14,
        borderRadius: 16,
    },
    buttonText: {
        color: '#fff',
        fontSize: 18,
        textAlign: 'center',
        fontWeight: '600',
    },
    skip: {
        alignItems: 'flex-end',
        paddingHorizontal: 24,
        paddingTop: 16,
    },
    skipText: {
        color: 'rgba(247,244,234,0.7)',
        fontWeight: '500',
    },
});
