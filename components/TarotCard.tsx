import React, {useEffect, useState} from 'react';
import {Pressable, StyleSheet, View} from 'react-native';
import Animated, {
    Easing,
    interpolate,
    useAnimatedStyle,
    useSharedValue,
    withTiming,
} from 'react-native-reanimated';
import type {Card} from '../entities';
import {cardImages} from '../utils/cardImages';

const CARD_BACK = require('../assets/cards/card_back.jpeg');
const BASE_W = 110;
const BASE_H = 165;
const ASPECT = BASE_H / BASE_W;

type Props = {
    card: Card;
    isReversed?: boolean;
    startFaceDown?: boolean;
    width?: number;
};

export const TarotCard = ({card, isReversed = false, startFaceDown = false, width = BASE_W}: Props) => {
    const cardW = width;
    const cardH = Math.round(cardW * ASPECT);

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
            <Animated.View style={[styles.cardBox, {width: cardW, height: cardH}, containerFade]}>
                <Animated.View style={[styles.face, {width: cardW, height: cardH}, frontAnimatedStyle]}>
                    {imageSource ? (
                        <Animated.Image
                            source={imageSource}
                            resizeMode="cover"
                            style={[
                                styles.fullImage,
                                {width: cardW, height: cardH},
                                isReversed && {transform: [{rotate: '180deg'}]},
                            ]}
                        />
                    ) : (
                        <View style={[styles.placeholder, {width: cardW, height: cardH}]} />
                    )}
                </Animated.View>

                <Animated.View style={[styles.face, {width: cardW, height: cardH}, backAnimatedStyle]}>
                    <Animated.Image
                        source={CARD_BACK}
                        resizeMode="cover"
                        style={[styles.fullImage, {width: cardW, height: cardH}]}
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
    fullImage: {
        borderRadius: 12,
        resizeMode: 'cover',
    },
    placeholder: {
        backgroundColor: '#1a1030',
        borderRadius: 12,
    },
});
