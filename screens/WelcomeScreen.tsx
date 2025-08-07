// screens/WelcomeScreen.tsx
import { View, Text, Button, StyleSheet } from "react-native";
import {Theme, useNavigation} from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../App";
import {ColorSchemeName} from "react-native/Libraries/Utilities/Appearance";
import {I18n} from "../i18n";
import {useState} from "react";

type Nav = NativeStackNavigationProp<RootStackParamList, "Welcome">;

export default function WelcomeScreen({theme}: {theme: ColorSchemeName}) {
    const navigation = useNavigation<Nav>();
    const s = styles(theme);

    const [language, setLanguage] = useState("en");

    const toggleLanguage = () => {
        const next = language === "en" ? "ru" : language === "ru" ? "th" : "en";
        I18n.locale = next;
        setLanguage(next);
    };

    return (
        <View style={s.container}>
            <Text style={s.title}>🔮 Tarot Reading</Text>
            <Button title="Start" onPress={() => navigation.navigate("SelectSpread")} />
            <Button title={`🌐 ${language}`} onPress={toggleLanguage} />
        </View>
    );
}

const styles = (theme: ColorSchemeName) =>
    StyleSheet.create({
        container: {
            flex: 1,
            justifyContent: "center",
            alignItems: "center",
            backgroundColor: theme === "dark" ? "#000" : "#fff",
        },
        title: {
            fontSize: 28,
            fontWeight: "bold",
            marginBottom: 20,
            color: theme === "dark" ? "#fff" : "#000",
        },
    });