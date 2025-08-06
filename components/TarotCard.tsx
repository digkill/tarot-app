import React, {useEffect, useState} from "react";
import {Image, Text, StyleSheet, Pressable} from "react-native";
import Animated, {
    useSharedValue,
    useAnimatedStyle,
    withTiming,
    interpolate,
    Easing,
} from "react-native-reanimated";
import type {TarotCard} from "../App";
import {cardImages} from "../utils/cardImages";

type Props = {
    card: TarotCard;
    theme?: "light" | "dark"; // для поддержки тем
};

export default function TarotCardComponent({card, theme = "light"}: Props) {
    const rotation = useSharedValue(0);
    const [flipped, setFlipped] = useState(false);
    const [visible, setVisible] = useState(false);

    const frontAnimatedStyle = useAnimatedStyle(() => ({
        transform: [{rotateY: `${interpolate(rotation.value, [0, 180], [0, 180])}deg`}],
        backfaceVisibility: "hidden",
    }));

    const backAnimatedStyle = useAnimatedStyle(() => ({
        transform: [{rotateY: `${interpolate(rotation.value, [0, 180], [180, 360])}deg`}],
        backfaceVisibility: "hidden",
        position: "absolute",
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
        setFlipped(!flipped);
    };

    useEffect(() => {
        setVisible(true);
    }, []);


    return (
        <Pressable onPress={handleFlip}>

            <Animated.View style={[
                styles.cardBox,
                containerFade,
                theme === "dark" && {backgroundColor: "#111"}
            ]}>

                <Animated.View style={[styles.face, frontAnimatedStyle]}>
                    <Animated.Image
                        source={cardImages[card.image]}
                        style={[
                            styles.image,
                            card.isReversed && {transform: [{rotate: "180deg"}]},
                        ]}
                    />
                    <Text
                        style={[
                            styles.name,
                            theme === "dark" && {color: "#fff"},
                            theme === "light" && {color: "#000"},
                        ]}
                    >
                        {card.name} {card.isReversed ? "(Reversed)" : ""}
                    </Text>
                </Animated.View>

                <Animated.View style={[styles.face, backAnimatedStyle]}>
                    <Text
                        style={[
                            styles.desc,
                            theme === "dark" && {color: "#fff"},
                            theme === "light" && {color: "#000"},
                        ]}
                    >
                        {card.isReversed ? card.reversed : card.upright}
                    </Text>
                </Animated.View>
            </Animated.View>
        </Pressable>
    );
}

const styles = StyleSheet.create({
    cardBox: {
        width: 100,
        height: 180,
        marginBottom: 35,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: "#fff", // по умолчанию, затем переопределим в theme
        borderRadius: 12,
        shadowColor: "#000",
        shadowOpacity: 0.1,
        shadowOffset: {width: 0, height: 4},
        shadowRadius: 6,
        elevation: 5, // работает на Android
    },
    face: {
        position: "absolute",
        width: "100%",
        height: "100%",
        alignItems: "center",
        justifyContent: "center",
    },
    image: {
        width: 90,
        height: 150,
        resizeMode: "contain",
    },
    name: {
        fontSize: 10,
        fontWeight: "bold",
        textAlign: "center",
        marginTop: 4,
    },
    desc: {
        fontSize: 12,
        padding: 12,
        textAlign: "center",
    },
});
