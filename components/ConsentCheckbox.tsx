import React from 'react';
import {StyleSheet, TouchableOpacity, View} from 'react-native';
import {Ionicons} from '@expo/vector-icons';

type Props = {
    checked: boolean;
    onToggle: () => void;
    children: React.ReactNode;
};

export const ConsentCheckbox = ({checked, onToggle, children}: Props) => {
    return (
        <View style={styles.row}>
            <TouchableOpacity
                onPress={onToggle}
                accessibilityRole="checkbox"
                accessibilityState={{checked}}
                hitSlop={8}
            >
                <Ionicons
                    name={checked ? 'checkbox' : 'square-outline'}
                    size={22}
                    color={checked ? '#6c5ce7' : 'rgba(247,244,234,0.7)'}
                />
            </TouchableOpacity>
            <View style={styles.label}>{children}</View>
        </View>
    );
};

const styles = StyleSheet.create({
    row: {
        flexDirection: 'row',
        alignItems: 'flex-start',
        gap: 10,
        paddingVertical: 8,
    },
    label: {
        flex: 1,
    },
});
