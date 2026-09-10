import React, {useEffect, useMemo, useState} from 'react';
import {Modal, Pressable, ScrollView, StyleSheet, Text, TouchableOpacity, View} from 'react-native';
import {Image as ExpoImage} from 'expo-image';
import {useTranslation} from 'react-i18next';
import type {Card} from '../entities';
import {useDeckShop} from '../providers/DeckShopProvider';
import {hexAlpha} from '../theme/appColors';

type Props = {
    visible: boolean;
    card: Card | null;
    isReversed?: boolean;
    positionTitle?: string;
    artDeckId?: string;
    onClose: () => void;
};

export const CardMeaningSheet = ({
    visible,
    card,
    isReversed = false,
    positionTitle,
    artDeckId,
    onClose,
}: Props) => {
    const {t} = useTranslation();
    const {colors, faceSource} = useDeckShop();
    const [showReversed, setShowReversed] = useState(isReversed);

    useEffect(() => {
        if (visible) {
            setShowReversed(isReversed);
        }
    }, [isReversed, visible, card?.id]);

    const side = useMemo(() => {
        if (!card) {
            return null;
        }
        return showReversed ? card.reversed : card.upright;
    }, [card, showReversed]);

    const source = card ? faceSource(card.image, artDeckId) : undefined;

    return (
        <Modal visible={visible && !!card} animationType="fade" transparent onRequestClose={onClose}>
            <Pressable style={styles.overlay} onPress={onClose}>
                <Pressable
                    style={[styles.sheet, {backgroundColor: colors.tabBar, borderColor: hexAlpha(colors.gold, 0.35)}]}
                    onPress={(event) => event.stopPropagation()}
                >
                    {card && side ? (
                        <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
                            <View style={[styles.artFrame, {borderColor: hexAlpha(colors.gold, 0.45), backgroundColor: colors.bg}]}>
                                {source ? (
                                    <ExpoImage
                                        source={source}
                                        style={[styles.art, showReversed && styles.artReversed]}
                                        contentFit="contain"
                                        allowDownscaling={false}
                                        cachePolicy="memory-disk"
                                    />
                                ) : null}
                            </View>
                            {positionTitle ? (
                                <Text style={[styles.position, {color: colors.muted}]}>{positionTitle}</Text>
                            ) : null}
                            <Text style={[styles.name, {color: colors.gold}]}>{card.name}</Text>
                            <View style={styles.toggleRow}>
                                <TouchableOpacity
                                    style={[
                                        styles.toggle,
                                        {borderColor: hexAlpha(colors.gold, 0.35)},
                                        !showReversed && {backgroundColor: colors.accent, borderColor: colors.accent},
                                    ]}
                                    onPress={() => setShowReversed(false)}
                                >
                                    <Text style={[styles.toggleText, {color: colors.text}]}>
                                        {t('cardMeaning.upright')}
                                    </Text>
                                </TouchableOpacity>
                                <TouchableOpacity
                                    style={[
                                        styles.toggle,
                                        {borderColor: hexAlpha(colors.gold, 0.35)},
                                        showReversed && {backgroundColor: colors.accent, borderColor: colors.accent},
                                    ]}
                                    onPress={() => setShowReversed(true)}
                                >
                                    <Text style={[styles.toggleText, {color: colors.text}]}>
                                        {t('cardMeaning.reversed')}
                                    </Text>
                                </TouchableOpacity>
                            </View>
                            {side.keywords.length ? (
                                <View style={styles.keywords}>
                                    {side.keywords.map((keyword) => (
                                        <View
                                            key={keyword}
                                            style={[styles.chip, {backgroundColor: hexAlpha(colors.accent, 0.18)}]}
                                        >
                                            <Text style={[styles.chipText, {color: colors.gold}]}>{keyword}</Text>
                                        </View>
                                    ))}
                                </View>
                            ) : null}
                            <Text style={[styles.section, {color: colors.gold}]}>{t('cardMeaning.general')}</Text>
                            <Text style={[styles.body, {color: colors.text}]}>{side.general}</Text>
                            {side.love ? (
                                <>
                                    <Text style={[styles.section, {color: colors.gold}]}>{t('cardMeaning.love')}</Text>
                                    <Text style={[styles.body, {color: colors.text}]}>{side.love}</Text>
                                </>
                            ) : null}
                            {side.work ? (
                                <>
                                    <Text style={[styles.section, {color: colors.gold}]}>{t('cardMeaning.work')}</Text>
                                    <Text style={[styles.body, {color: colors.text}]}>{side.work}</Text>
                                </>
                            ) : null}
                            {side.health ? (
                                <>
                                    <Text style={[styles.section, {color: colors.gold}]}>{t('cardMeaning.health')}</Text>
                                    <Text style={[styles.body, {color: colors.text}]}>{side.health}</Text>
                                </>
                            ) : null}
                            {side.advice ? (
                                <>
                                    <Text style={[styles.section, {color: colors.gold}]}>{t('cardMeaning.advice')}</Text>
                                    <Text style={[styles.body, {color: colors.text}]}>{side.advice}</Text>
                                </>
                            ) : null}
                            <TouchableOpacity
                                style={[styles.close, {backgroundColor: colors.accent}]}
                                onPress={onClose}
                            >
                                <Text style={styles.closeText}>{t('cardMeaning.close')}</Text>
                            </TouchableOpacity>
                        </ScrollView>
                    ) : null}
                </Pressable>
            </Pressable>
        </Modal>
    );
};

const styles = StyleSheet.create({
    overlay: {
        flex: 1,
        backgroundColor: 'rgba(0,0,0,0.72)',
        justifyContent: 'flex-end',
    },
    sheet: {
        maxHeight: '88%',
        borderTopLeftRadius: 24,
        borderTopRightRadius: 24,
        borderWidth: 1,
        overflow: 'hidden',
    },
    content: {
        padding: 20,
        paddingBottom: 36,
        gap: 10,
    },
    artFrame: {
        alignSelf: 'center',
        width: 168,
        height: 252,
        borderRadius: 14,
        borderWidth: 1,
        overflow: 'hidden',
        marginBottom: 4,
    },
    art: {
        width: '100%',
        height: '100%',
    },
    artReversed: {
        transform: [{rotate: '180deg'}],
    },
    position: {
        textAlign: 'center',
        fontSize: 13,
        fontWeight: '600',
    },
    name: {
        textAlign: 'center',
        fontSize: 22,
        fontWeight: '700',
    },
    toggleRow: {
        flexDirection: 'row',
        gap: 8,
        justifyContent: 'center',
        marginTop: 4,
    },
    toggle: {
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: 16,
        borderWidth: 1,
    },
    toggleText: {
        fontSize: 13,
        fontWeight: '600',
    },
    keywords: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        gap: 8,
        marginTop: 4,
    },
    chip: {
        borderRadius: 12,
        paddingHorizontal: 10,
        paddingVertical: 6,
    },
    chipText: {
        fontSize: 12,
        fontWeight: '600',
    },
    section: {
        fontSize: 14,
        fontWeight: '700',
        marginTop: 8,
    },
    body: {
        fontSize: 15,
        lineHeight: 22,
        opacity: 0.92,
    },
    close: {
        marginTop: 16,
        borderRadius: 16,
        paddingVertical: 14,
        alignItems: 'center',
    },
    closeText: {
        color: '#fff',
        fontWeight: '700',
        fontSize: 16,
    },
});
