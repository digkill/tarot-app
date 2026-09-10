import React, {useEffect, useState} from 'react';
import {Pressable, StyleSheet, View} from 'react-native';
import {Image as ExpoImage} from 'expo-image';
import Animated, {
    Easing,
    interpolate,
    useAnimatedStyle,
    useSharedValue,
    withTiming,
} from 'react-native-reanimated';
import type {Card} from '../entities';
import {useDeckShop} from '../providers/DeckShopProvider';

const BASE_W = 110;
const BASE_H = 165;
const ASPECT = BASE_H / BASE_W;

type Props = {
    card: Card;
    isReversed?: boolean;
    startFaceDown?: boolean;
    width?: number;
    artDeckId?: string;
    interactive?: boolean;
    highlighted?: boolean;
    dimmed?: boolean;
    onPressFace?: () => void;
};

export const TarotCard = ({
    card,
    isReversed = false,
    startFaceDown = false,
    width = BASE_W,
    artDeckId,
    interactive = true,
    highlighted = false,
    dimmed = false,
    onPressFace,
}: Props) => {
    const {faceSource, backSource, colors} = useDeckShop();
    const imageSource = faceSource(card.image, artDeckId);
    const cardBack = backSource(artDeckId);
    const cardW = width;
    const cardH = Math.round(cardW * ASPECT);
    const recycleKey = `${artDeckId ?? 'rws'}:${card.id}:${card.image}`;

    const rotation = useSharedValue(0);
    const [flipped, setFlipped] = useState(false);
    const [visible, setVisible] = useState(false);

    const showingFront = startFaceDown ? flipped : !flipped;

    const frontAnimatedStyle = useAnimatedStyle(() => {
        const deg = startFaceDown
            ? interpolate(rotation.value, [0, 180], [180, 360])
            : interpolate(rotation.value, [0, 180], [0, 180]);
        return {
            transform: [{rotateY: `${deg}deg`}],
            backfaceVisibility: 'hidden',
        };
    });

    const backAnimatedStyle = useAnimatedStyle(() => {
        const deg = startFaceDown
            ? interpolate(rotation.value, [0, 180], [0, 180])
            : interpolate(rotation.value, [0, 180], [180, 360]);
        return {
            transform: [{rotateY: `${deg}deg`}],
            backfaceVisibility: 'hidden',
            position: 'absolute',
            top: 0,
        };
    });

    const containerFade = useAnimatedStyle(() => ({
        opacity: withTiming(visible ? (dimmed ? 0.38 : 1) : 0, {duration: 500}),
        transform: [
            {
                translateY: withTiming(visible ? 0 : 20, {
                    duration: 500,
                    easing: Easing.out(Easing.ease),
                }),
            },
        ],
    }));

    const handlePress = () => {
        if (!interactive) {
            return;
        }
        if (showingFront) {
            onPressFace?.();
            return;
        }
        rotation.value = withTiming(flipped ? 0 : 180, {
            duration: 600,
            easing: Easing.out(Easing.ease),
        });
        setFlipped((prev) => !prev);
        if (onPressFace) {
            setTimeout(() => {
                onPressFace();
            }, 620);
        }
    };

    useEffect(() => {
        setVisible(true);
    }, []);

    return (
        <Pressable onPress={handlePress} disabled={!interactive}>
            <Animated.View
                style={[
                    styles.cardBox,
                    {width: cardW, height: cardH, borderColor: highlighted ? colors.gold : 'transparent'},
                    highlighted && styles.highlighted,
                    containerFade,
                ]}
            >
                <Animated.View style={[styles.face, {width: cardW, height: cardH}, frontAnimatedStyle]}>
                    {imageSource ? (
                        <ExpoImage
                            source={imageSource}
                            style={[styles.fullImage, isReversed && styles.reversed]}
                            contentFit="cover"
                            allowDownscaling={false}
                            cachePolicy="memory-disk"
                            recyclingKey={recycleKey}
                        />
                    ) : (
                        <View style={[styles.placeholder, {width: cardW, height: cardH, backgroundColor: colors.panel}]} />
                    )}
                </Animated.View>

                <Animated.View style={[styles.face, {width: cardW, height: cardH}, backAnimatedStyle]}>
                    <ExpoImage
                        source={cardBack}
                        style={styles.fullImage}
                        contentFit="cover"
                        allowDownscaling={false}
                        cachePolicy="memory-disk"
                        recyclingKey={`${recycleKey}:back`}
                    />
                </Animated.View>
            </Animated.View>
        </Pressable>
    );
};

export default TarotCard;

const styles = StyleSheet.create({
    cardBox: {
        borderRadius: 12,
        overflow: 'hidden',
        shadowColor: '#000',
        shadowOpacity: 0.45,
        shadowOffset: {width: 0, height: 6},
        shadowRadius: 10,
        elevation: 8,
        borderWidth: 2,
    },
    highlighted: {
        shadowColor: '#d4af37',
        shadowOpacity: 0.85,
        shadowRadius: 14,
        elevation: 12,
    },
    face: {
        borderRadius: 12,
        overflow: 'hidden',
    },
    fullImage: {
        width: '100%',
        height: '100%',
        borderRadius: 12,
    },
    reversed: {
        transform: [{rotate: '180deg'}],
    },
    placeholder: {
        backgroundColor: '#1a1030',
        borderRadius: 12,
    },
});
