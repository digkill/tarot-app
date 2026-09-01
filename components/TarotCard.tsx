import React, {useEffect, useState} from 'react';
import {ImageBackground, Pressable, StyleSheet, Text, View} from 'react-native';
import Animated, {
    Easing,
    interpolate,
    useAnimatedStyle,
    useSharedValue,
    withTiming,
} from 'react-native-reanimated';
import type {Card} from '../entities';
import {cardImages} from '../utils/cardImages';

const GOLD = '#d4af37';
const GOLD_LIGHT = '#f4d386';
const CARD_BG = '#1a1030';
const PATTERN = require('../assets/pattern2.png');

const BASE_W = 110;
const BASE_H = 190;
const ASPECT = BASE_H / BASE_W;

type Props = {
    card: Card;
    isReversed?: boolean;
    startFaceDown?: boolean;
    width?: number;
};

const CornerOrnament = ({position}: {position: 'tl' | 'tr' | 'bl' | 'br'}) => {
    const isTop = position === 'tl' || position === 'tr';
    const isLeft = position === 'tl' || position === 'bl';
    return (
        <View
            style={[
                styles.corner,
                isTop ? {top: 4} : {bottom: 4},
                isLeft ? {left: 4} : {right: 4},
                {
                    borderTopWidth: isTop ? 2 : 0,
                    borderBottomWidth: isTop ? 0 : 2,
                    borderLeftWidth: isLeft ? 2 : 0,
                    borderRightWidth: isLeft ? 0 : 2,
                },
            ]}
        />
    );
};

const GoldFrame = () => (
    <View style={StyleSheet.absoluteFill} pointerEvents="none">
        <View style={styles.outerBorder} />
        <View style={styles.innerBorder} />
        <CornerOrnament position="tl" />
        <CornerOrnament position="tr" />
        <CornerOrnament position="bl" />
        <CornerOrnament position="br" />
    </View>
);

const CardBackSide = ({cardW, cardH}: {cardW: number; cardH: number}) => {
    const scale = cardW / BASE_W;
    return (
        <ImageBackground
            source={PATTERN}
            style={[styles.backFill, {width: cardW, height: cardH, borderRadius: 12}]}
            resizeMode="cover"
            imageStyle={{borderRadius: 12}}
        >
            <View style={styles.backOverlay} />
            <View style={styles.emblemContainer}>
                <View style={[styles.diamondOuter, {width: 32 * scale, height: 32 * scale}]}>
                    <View style={[styles.diamondInner, {width: 14 * scale, height: 14 * scale}]} />
                </View>
            </View>
            <GoldFrame />
        </ImageBackground>
    );
};

export const TarotCard = ({card, isReversed = false, startFaceDown = false, width = BASE_W}: Props) => {
    const cardW = width;
    const cardH = Math.round(cardW * ASPECT);
    const scale = cardW / BASE_W;

    const rotation = useSharedValue(0);
    const [flipped, setFlipped] = useState(false);
    const [visible, setVisible] = useState(false);

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
        opacity: withTiming(visible ? 1 : 0, {duration: 500}),
        transform: [
            {
                translateY: withTiming(visible ? 0 : 20, {
                    duration: 500,
                    easing: Easing.out(Easing.ease),
                }),
            },
        ],
    }));

    const handleFlip = () => {
        rotation.value = withTiming(flipped ? 0 : 180, {
            duration: 600,
            easing: Easing.out(Easing.ease),
        });
        setFlipped((prev) => !prev);
    };

    useEffect(() => {
        setVisible(true);
    }, []);

    const imageSource = cardImages[card.image];

    return (
        <Pressable onPress={handleFlip}>
            <Animated.View
                style={[
                    styles.cardBox,
                    {width: cardW, height: cardH, marginBottom: Math.round(35 * scale)},
                    containerFade,
                ]}
            >
                <Animated.View style={[styles.face, {width: cardW, height: cardH}, frontAnimatedStyle]}>
                    <View style={[styles.faceInner, {borderRadius: 12}]}>
                        {imageSource ? (
                            <Animated.Image
                                source={imageSource}
                                style={[
                                    styles.image,
                                    {width: Math.round(82 * scale), height: Math.round(136 * scale)},
                                    isReversed && {transform: [{rotate: '180deg'}]},
                                ]}
                            />
                        ) : null}
                        <Text style={[styles.name, {fontSize: Math.max(7, Math.round(9 * scale))}]}>
                            {card.name}
                        </Text>
                        <GoldFrame />
                    </View>
                </Animated.View>

                <Animated.View style={[styles.face, {width: cardW, height: cardH}, backAnimatedStyle]}>
                    <CardBackSide cardW={cardW} cardH={cardH} />
                </Animated.View>
            </Animated.View>
        </Pressable>
    );
};

export default TarotCard;

const styles = StyleSheet.create({
    cardBox: {
        borderRadius: 12,
        shadowColor: '#000',
        shadowOpacity: 0.45,
        shadowOffset: {width: 0, height: 6},
        shadowRadius: 10,
        elevation: 8,
    },
    face: {
        borderRadius: 12,
        overflow: 'hidden',
    },
    faceInner: {
        flex: 1,
        backgroundColor: CARD_BG,
        borderRadius: 12,
        alignItems: 'center',
        justifyContent: 'center',
        paddingHorizontal: 6,
        paddingVertical: 8,
    },
    image: {
        resizeMode: 'contain',
    },
    name: {
        fontWeight: '700',
        textAlign: 'center',
        color: GOLD_LIGHT,
        marginTop: 4,
        letterSpacing: 0.5,
    },
    outerBorder: {
        position: 'absolute',
        top: 3,
        left: 3,
        right: 3,
        bottom: 3,
        borderRadius: 10,
        borderWidth: 1.5,
        borderColor: GOLD,
    },
    innerBorder: {
        position: 'absolute',
        top: 7,
        left: 7,
        right: 7,
        bottom: 7,
        borderRadius: 7,
        borderWidth: 0.5,
        borderColor: GOLD,
    },
    corner: {
        position: 'absolute',
        width: 10,
        height: 10,
        borderColor: GOLD_LIGHT,
    },
    backFill: {
        flex: 1,
        alignItems: 'center',
        justifyContent: 'center',
    },
    backOverlay: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(10,5,25,0.38)',
        borderRadius: 12,
    },
    emblemContainer: {
        alignItems: 'center',
        justifyContent: 'center',
    },
    diamondOuter: {
        backgroundColor: 'transparent',
        borderWidth: 2,
        borderColor: GOLD_LIGHT,
        transform: [{rotate: '45deg'}],
        alignItems: 'center',
        justifyContent: 'center',
    },
    diamondInner: {
        backgroundColor: GOLD,
    },
});
