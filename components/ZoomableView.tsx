import React, {useEffect, useState} from 'react';
import {LayoutChangeEvent, StyleSheet, Text, TouchableOpacity, View, type StyleProp, type ViewStyle} from 'react-native';
import {Gesture, GestureDetector} from 'react-native-gesture-handler';
import Animated, {
    runOnJS,
    useAnimatedReaction,
    useAnimatedStyle,
    useSharedValue,
    withTiming,
} from 'react-native-reanimated';

const MIN_SCALE = 1;
const MAX_SCALE = 3.5;
const ZOOM_EPSILON = 1.04;

function clampNumber(value: number, min: number, max: number) {
    'worklet';
    return Math.min(Math.max(value, min), max);
}

type Props = {
    enabled: boolean;
    resetKey: string;
    children: React.ReactNode;
    style?: StyleProp<ViewStyle>;
    hint?: string;
    resetLabel?: string;
};

export const ZoomableView = ({enabled, resetKey, children, style, hint, resetLabel}: Props) => {
    const scale = useSharedValue(1);
    const translateX = useSharedValue(0);
    const translateY = useSharedValue(0);
    const savedScale = useSharedValue(1);
    const savedX = useSharedValue(0);
    const savedY = useSharedValue(0);
    const layoutW = useSharedValue(0);
    const layoutH = useSharedValue(0);
    const pinchFocalX = useSharedValue(0);
    const pinchFocalY = useSharedValue(0);
    const [isZoomed, setIsZoomed] = useState(false);

    const applyReset = () => {
        scale.value = withTiming(1);
        translateX.value = withTiming(0);
        translateY.value = withTiming(0);
        savedScale.value = 1;
        savedX.value = 0;
        savedY.value = 0;
    };

    useEffect(() => {
        scale.value = 1;
        translateX.value = 0;
        translateY.value = 0;
        savedScale.value = 1;
        savedX.value = 0;
        savedY.value = 0;
        setIsZoomed(false);
        // Shared values are stable refs; reset when the table is redrawn.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [resetKey]);

    useAnimatedReaction(
        () => scale.value > ZOOM_EPSILON,
        (zoomed, previous) => {
            if (zoomed !== previous) {
                runOnJS(setIsZoomed)(zoomed);
            }
        },
    );

    const pinch = Gesture.Pinch()
        .enabled(enabled)
        .onStart((event) => {
            savedScale.value = scale.value;
            savedX.value = translateX.value;
            savedY.value = translateY.value;
            pinchFocalX.value = event.focalX;
            pinchFocalY.value = event.focalY;
        })
        .onUpdate((event) => {
            const nextScale = clampNumber(savedScale.value * event.scale, MIN_SCALE, MAX_SCALE);
            const scaleDiff = nextScale / savedScale.value;
            const originX = pinchFocalX.value - layoutW.value / 2;
            const originY = pinchFocalY.value - layoutH.value / 2;
            translateX.value = savedX.value + originX * (1 - scaleDiff);
            translateY.value = savedY.value + originY * (1 - scaleDiff);
            scale.value = nextScale;
        })
        .onEnd(() => {
            if (scale.value < ZOOM_EPSILON) {
                scale.value = withTiming(1);
                translateX.value = withTiming(0);
                translateY.value = withTiming(0);
                savedScale.value = 1;
                savedX.value = 0;
                savedY.value = 0;
                return;
            }

            savedScale.value = scale.value;
            const maxX = (scale.value - 1) * layoutW.value;
            const maxY = (scale.value - 1) * layoutH.value;
            const nextX = clampNumber(translateX.value, -maxX, maxX);
            const nextY = clampNumber(translateY.value, -maxY, maxY);
            translateX.value = withTiming(nextX);
            translateY.value = withTiming(nextY);
            savedX.value = nextX;
            savedY.value = nextY;
        });

    const pan = Gesture.Pan()
        .enabled(enabled)
        .minDistance(12)
        .maxPointers(1)
        .onStart(() => {
            savedX.value = translateX.value;
            savedY.value = translateY.value;
        })
        .onUpdate((event) => {
            if (scale.value <= ZOOM_EPSILON) {
                return;
            }
            const maxX = (scale.value - 1) * layoutW.value;
            const maxY = (scale.value - 1) * layoutH.value;
            translateX.value = clampNumber(savedX.value + event.translationX, -maxX, maxX);
            translateY.value = clampNumber(savedY.value + event.translationY, -maxY, maxY);
        })
        .onEnd(() => {
            savedX.value = translateX.value;
            savedY.value = translateY.value;
        });

    const composed = Gesture.Simultaneous(pinch, pan);

    const animatedStyle = useAnimatedStyle(() => ({
        transform: [
            {translateX: translateX.value},
            {translateY: translateY.value},
            {scale: scale.value},
        ],
    }));

    const handleLayout = (event: LayoutChangeEvent) => {
        const {width, height} = event.nativeEvent.layout;
        layoutW.value = width;
        layoutH.value = height;
    };

    return (
        <View style={[styles.clip, style]} onLayout={handleLayout}>
            <GestureDetector gesture={composed}>
                <Animated.View
                    style={[styles.content, animatedStyle]}
                    collapsable={false}
                    shouldRasterizeIOS={false}
                    renderToHardwareTextureAndroid={false}
                >
                    {children}
                </Animated.View>
            </GestureDetector>
            {hint && !isZoomed ? (
                <Text style={styles.hint} pointerEvents="none">
                    {hint}
                </Text>
            ) : null}
            {isZoomed && resetLabel ? (
                <TouchableOpacity style={styles.resetButton} onPress={applyReset} activeOpacity={0.8}>
                    <Text style={styles.resetLabel}>{resetLabel}</Text>
                </TouchableOpacity>
            ) : null}
        </View>
    );
};

const styles = StyleSheet.create({
    clip: {
        overflow: 'hidden',
    },
    content: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
    },
    hint: {
        position: 'absolute',
        left: 12,
        right: 12,
        bottom: 10,
        color: '#f7f4ea',
        opacity: 0.55,
        fontSize: 12,
        textAlign: 'center',
    },
    resetButton: {
        position: 'absolute',
        top: 10,
        right: 10,
        paddingHorizontal: 12,
        paddingVertical: 6,
        borderRadius: 14,
        backgroundColor: 'rgba(12,10,20,0.82)',
        borderWidth: 1,
        borderColor: 'rgba(108,92,231,0.7)',
    },
    resetLabel: {
        color: '#f7f4ea',
        fontSize: 12,
        fontWeight: '600',
    },
});
