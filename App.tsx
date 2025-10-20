import React, {useEffect} from 'react';
import {ActivityIndicator, useColorScheme, View} from 'react-native';
import {
    DarkTheme as NavigationDarkTheme,
    DefaultTheme as NavigationDefaultTheme,
    NavigationContainer,
} from '@react-navigation/native';
import {createNativeStackNavigator} from '@react-navigation/native-stack';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {SettingsProvider, useSettings} from './providers/SettingsProvider';
import {HistoryProvider} from './providers/HistoryProvider';
import {OnboardingScreen} from './screens/OnboardingScreen';
import {DisclaimerScreen} from './screens/DisclaimerScreen';
import {HomeScreen} from './screens/HomeScreen';
import {SpreadCatalogScreen} from './screens/SpreadCatalogScreen';
import {DeckGalleryScreen} from './screens/DeckGalleryScreen';
import {HistoryListScreen} from './screens/HistoryListScreen';
import {SettingsScreen} from './screens/SettingsScreen';
import {ReadingScreen} from './screens/ReadingScreen';
import {InterpretationScreen} from './screens/InterpretationScreen';
import {I18n} from './i18n';
import {AppTabsParamList, HomeStackParamList, RootStackParamList} from './navigation/types';

const RootStack = createNativeStackNavigator<RootStackParamList>();
const HomeStack = createNativeStackNavigator<HomeStackParamList>();
const Tab = createBottomTabNavigator<AppTabsParamList>();

const HomeStackNavigator = () => (
    <HomeStack.Navigator>
        <HomeStack.Screen name="Home" component={HomeScreen} options={{headerShown: false}} />
        <HomeStack.Screen
            name="SpreadCatalog"
            component={SpreadCatalogScreen}
            options={{title: 'Spreads'}}
        />
    </HomeStack.Navigator>
);

const MainTabs = () => (
    <Tab.Navigator screenOptions={{headerShown: false}}>
        <Tab.Screen
            name="Explore"
            component={HomeStackNavigator}
            options={{title: 'Home'}}
        />
        <Tab.Screen name="Decks" component={DeckGalleryScreen} options={{title: 'Decks'}} />
        <Tab.Screen name="History" component={HistoryListScreen} options={{title: 'History'}} />
        <Tab.Screen name="Settings" component={SettingsScreen} options={{title: 'Settings'}} />
    </Tab.Navigator>
);

const AppNavigation = () => {
    const {settings, loading} = useSettings();
    const systemScheme = useColorScheme();

    useEffect(() => {
        if (settings.language) {
            I18n.changeLanguage(settings.language).catch(() => {});
        }
    }, [settings.language]);

    const scheme =
        settings.theme === 'system' ? systemScheme ?? 'light' : settings.theme === 'dark' ? 'dark' : 'light';
    const theme = scheme === 'dark' ? NavigationDarkTheme : NavigationDefaultTheme;
    const initialRoute: keyof RootStackParamList = !settings.hasCompletedOnboarding
        ? 'Onboarding'
        : !settings.acceptedDisclaimer
        ? 'Disclaimer'
        : 'Main';

    if (loading) {
        return (
            <View style={{flex: 1, justifyContent: 'center', alignItems: 'center'}}>
                <ActivityIndicator />
            </View>
        );
    }

    return (
        <NavigationContainer theme={theme}>
            <RootStack.Navigator screenOptions={{headerShown: false}} initialRouteName={initialRoute}>
                <RootStack.Screen name="Onboarding" component={OnboardingScreen} />
                <RootStack.Screen name="Disclaimer" component={DisclaimerScreen} />
                <RootStack.Screen name="Main" component={MainTabs} />
                <RootStack.Screen
                    name="Reading"
                    component={ReadingScreen}
                    options={{headerShown: true, title: 'Reading'}}
                />
                <RootStack.Screen
                    name="Interpretation"
                    component={InterpretationScreen}
                    options={{headerShown: true, title: 'Interpretation'}}
                />
            </RootStack.Navigator>
        </NavigationContainer>
    );
};

export default function App() {
    return (
        <SettingsProvider>
            <HistoryProvider>
                <AppNavigation />
            </HistoryProvider>
        </SettingsProvider>
    );
}
