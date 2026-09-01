import React, {useEffect} from 'react';
import {ActivityIndicator, useColorScheme, View} from 'react-native';
import {
    DarkTheme as NavigationDarkTheme,
    DefaultTheme as NavigationDefaultTheme,
    NavigationContainer,
} from '@react-navigation/native';
import {createNativeStackNavigator} from '@react-navigation/native-stack';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {useTranslation} from 'react-i18next';
import {SettingsProvider, useSettings} from './providers/SettingsProvider';
import {HistoryProvider} from './providers/HistoryProvider';
import {DisclaimerScreen} from './screens/DisclaimerScreen';
import {HomeScreen} from './screens/HomeScreen';
import {SpreadCatalogScreen} from './screens/SpreadCatalogScreen';
import {DeckGalleryScreen} from './screens/DeckGalleryScreen';
import {HistoryListScreen} from './screens/HistoryListScreen';
import {SettingsScreen} from './screens/SettingsScreen';
import {ReadingScreen} from './screens/ReadingScreen';
import {InterpretationScreen} from './screens/InterpretationScreen';
import {Ionicons} from '@expo/vector-icons';
import {GestureHandlerRootView} from 'react-native-gesture-handler';
import {I18n} from './i18n';
import {AppTabsParamList, HomeStackParamList, RootStackParamList} from './navigation/types';

const RootStack = createNativeStackNavigator<RootStackParamList>();
const HomeStack = createNativeStackNavigator<HomeStackParamList>();
const Tab = createBottomTabNavigator<AppTabsParamList>();

const HomeStackNavigator = () => {
    const {t} = useTranslation();
    return (
        <HomeStack.Navigator>
            <HomeStack.Screen name="Home" component={HomeScreen} options={{headerShown: false}} />
            <HomeStack.Screen
                name="SpreadCatalog"
                component={SpreadCatalogScreen}
                options={{title: t('nav.spreads')}}
            />
        </HomeStack.Navigator>
    );
};

const TAB_ICONS: Record<keyof AppTabsParamList, keyof typeof Ionicons.glyphMap> = {
    Explore: 'home-outline',
    Decks: 'albums-outline',
    History: 'time-outline',
    Settings: 'settings-outline',
};

const MainTabs = () => {
    const {t} = useTranslation();
    return (
        <Tab.Navigator
            screenOptions={({route}) => ({
                headerShown: false,
                tabBarActiveTintColor: '#6c5ce7',
                tabBarInactiveTintColor: 'rgba(247,244,234,0.55)',
                tabBarStyle: {backgroundColor: '#0c0a14', borderTopColor: 'rgba(244,211,134,0.15)'},
                tabBarIcon: ({color, size}) => (
                    <Ionicons name={TAB_ICONS[route.name]} size={size} color={color} />
                ),
            })}
        >
            <Tab.Screen
                name="Explore"
                component={HomeStackNavigator}
                options={{title: t('nav.home')}}
            />
            <Tab.Screen name="Decks" component={DeckGalleryScreen} options={{title: t('nav.decks')}} />
            <Tab.Screen name="History" component={HistoryListScreen} options={{title: t('nav.history')}} />
            <Tab.Screen name="Settings" component={SettingsScreen} options={{title: t('nav.settings')}} />
        </Tab.Navigator>
    );
};

const AppNavigation = () => {
    const {settings, loading} = useSettings();
    const systemScheme = useColorScheme();
    const {t} = useTranslation();

    useEffect(() => {
        if (settings.language) {
            I18n.changeLanguage(settings.language).catch(() => {});
        }
    }, [settings.language]);

    const scheme =
        settings.theme === 'system' ? systemScheme ?? 'light' : settings.theme === 'dark' ? 'dark' : 'light';
    const theme = scheme === 'dark' ? NavigationDarkTheme : NavigationDefaultTheme;
    const initialRoute: keyof RootStackParamList = !settings.acceptedDisclaimer
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
                <RootStack.Screen name="Disclaimer" component={DisclaimerScreen} />
                <RootStack.Screen name="Main" component={MainTabs} />
                <RootStack.Screen
                    name="Reading"
                    component={ReadingScreen}
                    options={{headerShown: true, title: t('nav.reading')}}
                />
                <RootStack.Screen
                    name="Interpretation"
                    component={InterpretationScreen}
                    options={{headerShown: true, title: t('nav.interpretation')}}
                />
            </RootStack.Navigator>
        </NavigationContainer>
    );
};

export default function App() {
    return (
        <GestureHandlerRootView style={{flex: 1}}>
            <SettingsProvider>
                <HistoryProvider>
                    <AppNavigation />
                </HistoryProvider>
            </SettingsProvider>
        </GestureHandlerRootView>
    );
}
