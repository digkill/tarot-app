import { useEffect, useState } from "react";
import { View, Text, FlatList, StyleSheet, TouchableOpacity, Alert } from "react-native";
import { getHistory, clearHistory } from "../storage/history";
import TarotCardComponent from "../components/TarotCard";
import type { TarotCard } from "../App";
import {ColorSchemeName} from "react-native/Libraries/Utilities/Appearance";

type HistoryItem = {
    id: string;
    timestamp: string;
    cards: TarotCard[];
};

export default function HistoryScreen({theme = "light"}: {theme: ColorSchemeName}) {
    const [history, setHistory] = useState<HistoryItem[]>([]);

    const loadHistory = async () => {
        const data = await getHistory();
        setHistory(data);
    };

    const handleClear = async () => {
        Alert.alert(
            "Clear History",
            "Are you sure you want to delete all spreads?",
            [
                { text: "Cancel", style: "cancel" },
                {
                    text: "Yes, delete",
                    style: "destructive",
                    onPress: async () => {
                        await clearHistory();
                        await loadHistory();
                    },
                },
            ]
        );
    };

    useEffect(() => {
        loadHistory();
    }, []);

    return (
        <View style={styles.wrapper}>
            <FlatList
                contentContainerStyle={styles.container}
                data={history}
                keyExtractor={(item) => item.id}
                ListEmptyComponent={<Text>No spreads yet...</Text>}
                renderItem={({ item }) => (
                    <View style={styles.spread}>
                        <Text style={styles.date}>
                            🕓 {new Date(item.timestamp).toLocaleString()}
                        </Text>
                        {item.cards.map((card, idx) => (
                            <TarotCardComponent key={idx} card={card} />
                        ))}
                    </View>
                )}
            />
            <TouchableOpacity style={styles.clearBtn} onPress={handleClear}>
                <Text style={styles.clearText}>🗑️ Clear History</Text>
            </TouchableOpacity>
        </View>
    );
}

const styles = StyleSheet.create({
    wrapper: { flex: 1 },
    container: { padding: 20, paddingBottom: 100 },
    spread: { marginBottom: 40 },
    date: { fontSize: 16, fontWeight: "bold", marginBottom: 10 },
    clearBtn: {
        position: "absolute",
        bottom: 20,
        alignSelf: "center",
        paddingVertical: 10,
        paddingHorizontal: 20,
        backgroundColor: "#ffdddd",
        borderRadius: 8,
    },
    clearText: {
        color: "#cc0000",
        fontWeight: "bold",
    },
});
