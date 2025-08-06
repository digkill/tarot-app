import React from "react";
import { View, Text, Button, StyleSheet } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { RootStackParamList, TarotCard } from "../App";
import tarotData from "../data/tarot.json";
import { saveSpread } from "../storage/history";
import {ColorSchemeName} from "react-native/Libraries/Utilities/Appearance"; // 🔥 импорт

type Nav = NativeStackNavigationProp<RootStackParamList, "SelectSpread">;

function drawCards(count: number): TarotCard[] {
    const shuffled = [...tarotData].sort(() => 0.5 - Math.random());
    return shuffled.slice(0, count).map(card => ({
        ...card,
        isReversed: Math.random() > 0.5,
    }));
}

export default function SelectSpreadScreen({theme = "light"}: {theme: ColorSchemeName}) {
    const navigation = useNavigation<Nav>();

    const handleDraw = async (count: number) => {
        const cards = drawCards(count);
        await saveSpread(cards); // 💾 сохраняем в AsyncStorage
        navigation.navigate("Result", { cards }); // переход к результату
    };

    return (
        <View style={styles.container}>
            <Text style={styles.title}>Choose a spread</Text>
            <Button title="Celtic Cross (10 cards)" onPress={() => handleDraw(10)} />
            <Button title="1 Card" onPress={() => handleDraw(1)} />
            <Button title="3 Cards" onPress={() => handleDraw(3)} />
        </View>
    );
}

const styles = StyleSheet.create({
    container: { flex: 1, justifyContent: "center", alignItems: "center", gap: 20 },
    title: { fontSize: 22, fontWeight: "bold", marginBottom: 20 },
});
