import React, {useEffect, useState} from 'react';
import {Pressable, StyleSheet, Text} from 'react-native';
import Animated, {
    Easing,
    interpolate,
    useAnimatedStyle,
    useSharedValue,
    withTiming,
} from 'react-native-reanimated';
import type {Card} from '../entities';
import {cardImages} from '../utils/cardImages';

type Props = {
    card: Card;
    isReversed?: boolean;
    theme?: 'light' | 'dark';
};

export const TarotCard = ({card, isReversed = false, theme = 'light'}: Props) => {
    const rotation = useSharedValue(0);
    const [flipped, setFlipped] = useState(false);
    const [visible, setVisible] = useState(false);

    const frontAnimatedStyle = useAnimatedStyle(() => ({
        transform: [{rotateY: `${interpolate(rotation.value, [0, 180], [0, 180])}deg`}],
        backfaceVisibility: 'hidden',
    }));

    const backAnimatedStyle = useAnimatedStyle(() => ({
        transform: [{rotateY: `${interpolate(rotation.value, [0, 180], [180, 360])}deg`}],
        backfaceVisibility: 'hidden',
        position: 'absolute',
        top: 0,
    }));

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
                    containerFade,
                    theme === 'dark' && {backgroundColor: '#111'},
                ]}
            >
                <Animated.View style={[styles.face, frontAnimatedStyle]}>
                    {imageSource ? (
                        <Animated.Image
                            source={imageSource}
                            style={[
                                styles.image,
                                isReversed && {transform: [{rotate: '180deg'}]},
                            ]}
                        />
                    ) : null}
                    <Text
                        style={[
                            styles.name,
                            theme === 'dark' && {color: '#fff'},
                            theme === 'light' && {color: '#000'},
                        ]}
                    >
                        {card.name} {isReversed ? '(Reversed)' : ''}
                    </Text>
                </Animated.View>

                <Animated.View style={[styles.face, backAnimatedStyle]}>
                    <Text
                        style={[
                            styles.desc,
                            theme === 'dark' && {color: '#fff'},
                            theme === 'light' && {color: '#000'},
                        ]}
                    >
                        {isReversed ? card.reversed.general : card.upright.general}
                    </Text>
                </Animated.View>
            </Animated.View>
        </Pressable>
    );
};

export default TarotCard;

const styles = StyleSheet.create({
    cardBox: {
        width: 110,
        height: 190,
        marginBottom: 35,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#fff',
        borderRadius: 12,
        shadowColor: '#000',
        shadowOpacity: 0.1,
        shadowOffset: {width: 0, height: 4},
        shadowRadius: 6,
        elevation: 5,
        padding: 8,
    },
    face: {
        position: 'absolute',
        width: '100%',
        height: '100%',
        alignItems: 'center',
        justifyContent: 'center',
    },
    image: {
        width: 95,
        height: 150,
        resizeMode: 'contain',
    },
    name: {
        fontSize: 11,
        fontWeight: 'bold',
        textAlign: 'center',
        marginTop: 6,
    },
    desc: {
        fontSize: 12,
        padding: 12,
        textAlign: 'center',
    },
});
