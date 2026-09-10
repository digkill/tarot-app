import React from 'react';
import {Image, StyleSheet, View} from 'react-native';
import {useAppColors} from '../providers/DeckShopProvider';

type Props = {
    children: React.ReactNode;
};

export const AppBackground = ({children}: Props) => {
    const colors = useAppColors();
    return (
        <View style={[styles.root, {backgroundColor: colors.bg}]}>
            <View pointerEvents="none" style={styles.patternWrap}>
                <Image
                    source={require('../assets/pattern-print.png')}
                    style={styles.pattern}
                    resizeMode="repeat"
                />
            </View>
            {children}
        </View>
    );
};

const styles = StyleSheet.create({
    root: {
        flex: 1,
    },
    patternWrap: {
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        left: 0,
    },
    pattern: {
        width: '100%',
        height: '100%',
        opacity: 0.22,
    },
});
