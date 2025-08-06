import React, { useRef, useState } from "react";
import { View, Text, Button, ScrollView, StyleSheet } from "react-native";
import ViewShot from "react-native-view-shot";
import * as MediaLibrary from "expo-media-library";
import * as Sharing from "expo-sharing";
import TarotCardComponent from "../components/TarotCard";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../App";
import { generateTarotAnalysis } from "../utils/gpt";
import {ColorSchemeName} from "react-native/Libraries/Utilities/Appearance";

type Props = NativeStackScreenProps<RootStackParamList, "Result">;
type PropsWithTheme = Props & { theme?: ColorSchemeName };

export default function ResultScreen({ route, theme = "light" }: PropsWithTheme) {
    const cards = route.params.cards;
    const viewRef = useRef<any>(null);
    const [analysis, setAnalysis] = useState("");

    const handleScreenshot = async () => {
        if (!viewRef.current) {
            console.warn("ViewShot ref is not available yet");
            return;
        }

        try {
            const uri = await viewRef.current.capture();
            const { status } = await MediaLibrary.requestPermissionsAsync();
            if (status === "granted") {
                await MediaLibrary.saveToLibraryAsync(uri);
            }
            await Sharing.shareAsync(uri);
        } catch (err) {
            console.error("Screenshot failed:", err);
        }
    };

    const handleAI = async () => {
        const names = cards.map((c) => c.name + (c.isReversed ? " (reversed)" : ""));
        const result = await generateTarotAnalysis(names);
        setAnalysis(result);
    };

    return (
        <ScrollView contentContainerStyle={styles.container}>
            <ViewShot ref={viewRef} options={{ format: "png", quality: 1 }}>
                {cards.map((card, idx) => (
                    <TarotCardComponent key={idx} card={card} />
                ))}
            </ViewShot>

            <Button title="📸 Save & Share Spread" onPress={handleScreenshot} />
            <Button title="🔮 Get AI Reading" onPress={handleAI} />

            {analysis && (
                <View style={styles.analysisBox}>
                    <Text style={styles.analysis}>{analysis}</Text>
                </View>
            )}
        </ScrollView>
    );
}

const styles = StyleSheet.create({
    container: { padding: 20 },
    analysisBox: {
        marginTop: 20,
        padding: 15,
        backgroundColor: "#f0f0f0",
        borderRadius: 10,
    },
    analysis: {
        fontSize: 16,
        fontStyle: "italic",
    },
});
