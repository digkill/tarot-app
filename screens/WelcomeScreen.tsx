// screens/WelcomeScreen.tsx
import { View, Text, Button, StyleSheet } from "react-native";
import {Theme, useNavigation} from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../App";
import {ColorSchemeName} from "react-native/Libraries/Utilities/Appearance";

type Nav = NativeStackNavigationProp<RootStackParamList, "Welcome">;

export default function WelcomeScreen({theme}: {theme: ColorSchemeName}) {
    const navigation = useNavigation<Nav>();
    const s = styles(theme);
    return (
        <View style={s.container}>
            <Text style={s.title}>🔮 Tarot Reading</Text>
            <Button title="Start" onPress={() => navigation.navigate("SelectSpread")} />
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