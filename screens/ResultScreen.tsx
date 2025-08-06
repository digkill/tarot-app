import React, { useRef, useState } from 'react';
import {
    View,
    Text,
    Button,
    ScrollView,
    StyleSheet,
    Dimensions,
    TouchableOpacity,
    Alert,
} from 'react-native';
import ViewShot from 'react-native-view-shot';
import * as MediaLibrary from 'expo-media-library';
import * as Sharing from 'expo-sharing';
import { useTranslation } from 'react-i18next';
import TarotCardComponent from '../components/TarotCard';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import type { RootStackParamList } from '../App';
import { generateTarotAnalysis } from '../utils/gpt';
import { ColorSchemeName } from 'react-native/Libraries/Utilities/Appearance';
import { I18n } from '../i18n';

const languages = ['en', 'ru', 'th'];

const CARD_WIDTH = 100;
const CARD_HEIGHT = 160;
const SPACING = 60;

const celticCrossLabels: { [key: number]: string } = {
    1: 'present',
    2: 'challenge',
    3: 'past',
    4: 'future',
    5: 'above',
    6: 'below',
    7: 'advice',
    8: 'external',
    9: 'hopes',
    10: 'outcome',
};

type Props = NativeStackScreenProps<RootStackParamList, 'Result'>;
type PropsWithTheme = Props & { theme?: ColorSchemeName };

export default function ResultScreen({ route, theme = 'light' }: PropsWithTheme) {
    const { cards } = route.params;
    const viewRef = useRef<any>(null);
    const [analysis, setAnalysis] = useState('');
    const [langIndex, setLangIndex] = useState(0);
    const { t, i18n } = useTranslation(); // Используем хук useTranslation

    const handleScreenshot = async () => {
        if (!viewRef.current) {
            Alert.alert('Error', t('error_screenshot_not_ready'));
            return;
        }
        try {
            const uri = await viewRef.current.capture();
            const { status } = await MediaLibrary.requestPermissionsAsync();
            if (status === 'granted') {
                await MediaLibrary.saveToLibraryAsync(uri);
                Alert.alert(t('success'), t('screenshot_saved'));
            } else {
                Alert.alert(t('error'), t('permission_denied'));
            }
            await Sharing.shareAsync(uri);
        } catch (err) {
            console.error('Screenshot failed:', err);
            Alert.alert(t('error'), t('screenshot_failed'));
        }
    };

    const handleAI = async () => {
        try {
            const names = cards.map((c) => c.name + (c.isReversed ? ' (reversed)' : ''));
            const result = await generateTarotAnalysis(names);
            setAnalysis(result);
        } catch (err) {
            console.error('AI analysis failed:', err);
            Alert.alert(t('error'), t('ai_analysis_failed'));
        }
    };

    const switchLanguage = () => {
        const newLang = languages[(langIndex + 1) % languages.length];
        i18n.changeLanguage(newLang);
        setLangIndex((langIndex + 1) % languages.length);
    };

    const renderCelticCross = () => {
        const centerX = Dimensions.get('window').width / 2;
        const centerY = 300;

        const getCoords = (index: number): { left: number; top: number; rotate?: string } => {
            switch (index) {
                case 1:
                    return { left: centerX - CARD_WIDTH / 2, top: centerY };
                case 2:
                    return {
                        left: centerX - CARD_HEIGHT / 3,
                        top: centerY + CARD_WIDTH / 3 - CARD_HEIGHT / 3,
                        rotate: '90deg',
                    };
                case 3:
                    return {
                        left: centerX - CARD_WIDTH / 2,
                        top: centerY + CARD_HEIGHT + SPACING,
                    };
                case 4:
                    return {
                        left: centerX - CARD_WIDTH - SPACING - CARD_WIDTH / 2,
                        top: centerY,
                    };
                case 5:
                    return {
                        left: centerX - CARD_WIDTH / 2,
                        top: centerY - CARD_HEIGHT - SPACING,
                    };
                case 6:
                    return {
                        left: centerX + CARD_WIDTH + SPACING - CARD_WIDTH / 2,
                        top: centerY,
                    };
                case 7:
                    return {
                        left: centerX + CARD_WIDTH * 2.5,
                        top: centerY - CARD_HEIGHT * 1.5 - SPACING * 1.5,
                    };
                case 8:
                    return {
                        left: centerX + CARD_WIDTH * 2.5,
                        top: centerY - CARD_HEIGHT / 2 - SPACING / 2,
                    };
                case 9:
                    return {
                        left: centerX + CARD_WIDTH * 2.5,
                        top: centerY + CARD_HEIGHT / 2 + SPACING / 2,
                    };
                case 10:
                    return {
                        left: centerX + CARD_WIDTH * 2.5,
                        top: centerY + CARD_HEIGHT * 1.5 + SPACING * 1.5,
                    };
                default:
                    return { left: 0, top: 0 };
            }
        };

        return (
            <View style={styles.crossContainer}>
                {cards.slice(0, 10).map((card, idx) => {
                    const i = idx + 1;
                    const { left, top, rotate } = getCoords(i);

                    return (
                        <View key={i} style={{ position: 'absolute', left, top, alignItems: 'center' }}>
                            <View style={{ transform: rotate ? [{ rotate }] : [] }}>
                                <Text style={styles.label}>{t(celticCrossLabels[i])}</Text>
                                <TarotCardComponent card={card} />
                            </View>
                        </View>
                    );
                })}
            </View>
        );
    };

    return (
        <ScrollView
            contentContainerStyle={[styles.container, { backgroundColor: theme === 'dark' ? '#333' : '#fff' }]}
        >
            <TouchableOpacity onPress={switchLanguage}>
                <Text style={[styles.languageButton, { color: theme === 'dark' ? '#fff' : '#000' }]}>
                    🌐 {i18n.language}
                </Text>
            </TouchableOpacity>

            <ViewShot ref={viewRef} options={{ format: 'png', quality: 1 }}>
                {cards.length === 10 ? (
                    renderCelticCross()
                ) : (
                    cards.map((card, idx) => <TarotCardComponent key={idx} card={card} />)
                )}
            </ViewShot>

            <View style={styles.buttonContainer}>
                <Button
                    title={t('save_share')}
                    onPress={handleScreenshot}
                    color={theme === 'dark' ? '#aaa' : '#000'}
                />
                <Button
                    title={t('get_ai')}
                    onPress={handleAI}
                    color={theme === 'dark' ? '#aaa' : '#000'}
                />
            </View>

            {analysis && (
                <View style={[styles.analysisBox, { backgroundColor: theme === 'dark' ? '#444' : '#f0f0f0' }]}>
                    <Text style={[styles.analysis, { color: theme === 'dark' ? '#fff' : '#000' }]}>
                        {analysis}
                    </Text>
                </View>
            )}
        </ScrollView>
    );
}

const styles = StyleSheet.create({
    container: {
        padding: 20,
        minHeight: 1000,
    },
    crossContainer: {
        width: '100%',
        height: 1100,
        position: 'relative',
    },
    buttonContainer: {
        marginTop: 20,
        gap: 10,
    },
    analysisBox: {
        marginTop: 20,
        padding: 15,
        borderRadius: 10,
    },
    analysis: {
        fontSize: 16,
        fontStyle: 'italic',
    },
    label: {
        marginBottom: 4,
        fontSize: 10,
        fontWeight: '500',
        textAlign: 'center',
    },
    languageButton: {
        textAlign: 'center',
        marginBottom: 10,
        fontSize: 16,
        fontWeight: 'bold',
    },
});