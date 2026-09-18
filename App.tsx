import React, {useEffect, useState} from 'react';
import {ActivityIndicator, StatusBar, useColorScheme, View} from 'react-native';
import {
    DarkTheme as NavigationDarkTheme,
    DefaultTheme as NavigationDefaultTheme,
    NavigationContainer,
    type LinkingOptions,
} from '@react-navigation/native';
import {createNativeStackNavigator} from '@react-navigation/native-stack';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {useTranslation} from 'react-i18next';
import {SettingsProvider, useSettings} from './providers/SettingsProvider';
import {HistoryProvider} from './providers/HistoryProvider';
import {AuthProvider, useAuth} from './providers/AuthProvider';
import {DeckShopProvider, useAppColors} from './providers/DeckShopProvider';
import {ArcanaProvider} from './providers/ArcanaProvider';
import type {AppColors} from './theme/appColors';
import {DisclaimerScreen} from './screens/DisclaimerScreen';
import {AuthScreen} from './screens/AuthScreen';
import {VerifyEmailScreen} from './screens/VerifyEmailScreen';
import {ForgotPasswordScreen} from './screens/ForgotPasswordScreen';
import {LegalDocumentScreen} from './screens/LegalDocumentScreen';
import {HomeScreen} from './screens/HomeScreen';
import {ArcanaHomeScreen} from './screens/ArcanaHomeScreen';
import {ArcanaBattleScreen} from './screens/ArcanaBattleScreen';
import {
    ArcanaHeroesScreen,
    ArcanaMatchHistoryScreen,
    ArcanaRulesScreen,
} from './screens/ArcanaInfoScreens';
import {SpreadCatalogScreen} from './screens/SpreadCatalogScreen';
import {DeckGalleryScreen} from './screens/DeckGalleryScreen';
import {HistoryListScreen} from './screens/HistoryListScreen';
import {SettingsScreen} from './screens/SettingsScreen';
import {ReadingScreen} from './screens/ReadingScreen';
import {InterpretationScreen} from './screens/InterpretationScreen';
import {Ionicons} from '@expo/vector-icons';
import {AppBackground} from './components/AppBackground';
import {GestureHandlerRootView} from 'react-native-gesture-handler';
import {I18n} from './i18n';
import {VideoSplash} from './components/VideoSplash';
import {AppTabsParamList, ArcanaStackParamList, HomeStackParamList, RootStackParamList} from './navigation/types';
import * as WebBrowser from 'expo-web-browser';

WebBrowser.maybeCompleteAuthSession();

const RootStack = createNativeStackNavigator<RootStackParamList>();
const HomeStack = createNativeStackNavigator<HomeStackParamList>();
const Tab = createBottomTabNavigator<AppTabsParamList>();
const ArcanaStack = createNativeStackNavigator<ArcanaStackParamList>();

// Deep links, e.g. mediarisetarot://deck/japanese opens the Decks tab
// scrolled to that deck's promo card. Only resolves once the user has
// passed the disclaimer/auth gate and Main is on screen.
const linking: LinkingOptions<RootStackParamList> = {
    prefixes: ['mediarisetarot://'],
    config: {
        screens: {
            Main: {
                screens: {
                    Decks: 'deck/:slug?',
                },
            },
            // react-navigation's PathConfigMap can't express a nested tab
            // navigator's param list through RootStackParamList without
            // threading generics through every level; the shape above is
            // valid at runtime, so cast past the type-checker here.
        } as never,
    },
};

const legalScreenOptions = (t: (key: string) => string, colors: AppColors) => {
    return ({route}: {route: {params: RootStackParamList['LegalDocument']}}) => ({
        headerShown: true,
        title: route.params.doc === 'privacy' ? t('legal.privacyTitle') : t('legal.termsTitle'),
        headerTintColor: colors.text,
        headerStyle: {backgroundColor: colors.bg},
        headerShadowVisible: false,
    });
};

const stackHeader = (colors: AppColors) => ({
    headerTintColor: colors.text,
    headerStyle: {backgroundColor: colors.bg},
    headerShadowVisible: false,
    headerTitleStyle: {color: colors.gold, fontWeight: '700' as const},
    contentStyle: {backgroundColor: 'transparent'},
});

const HomeStackNavigator = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    return (
        <HomeStack.Navigator screenOptions={stackHeader(colors)}>
            <HomeStack.Screen name="Home" component={HomeScreen} options={{headerShown: false}} />
            <HomeStack.Screen
                name="SpreadCatalog"
                component={SpreadCatalogScreen}
                options={{title: t('nav.spreads')}}
            />
        </HomeStack.Navigator>
    );
};

// Arcana Clash: the lobby and its reference screens, with the battlefield
// pushed on top (it hides the tab bar, like Reading does).
const ArcanaStackNavigator = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    return (
        <ArcanaStack.Navigator screenOptions={stackHeader(colors)}>
            <ArcanaStack.Screen name="ArcanaHome" component={ArcanaHomeScreen} options={{headerShown: false}} />
            <ArcanaStack.Screen name="ArcanaRules" component={ArcanaRulesScreen} options={{title: t('arcana.rules')}} />
            <ArcanaStack.Screen name="ArcanaHeroes" component={ArcanaHeroesScreen} options={{title: t('arcana.heroes')}} />
            <ArcanaStack.Screen
                name="ArcanaMatchHistory"
                component={ArcanaMatchHistoryScreen}
                options={{title: t('arcana.history')}}
            />
        </ArcanaStack.Navigator>
    );
};

const TAB_ICONS: Record<keyof AppTabsParamList, keyof typeof Ionicons.glyphMap> = {
    Explore: 'home-outline',
    Arcana: 'flash-outline',
    Decks: 'albums-outline',
    History: 'time-outline',
    Settings: 'settings-outline',
};

const MainTabs = () => {
    const {t} = useTranslation();
    const colors = useAppColors();
    return (
        <Tab.Navigator
            screenOptions={({route}) => ({
                headerShown: false,
                tabBarActiveTintColor: colors.accent,
                tabBarInactiveTintColor: colors.muted,
                tabBarStyle: {backgroundColor: colors.tabBar, borderTopColor: colors.gold},
                sceneStyle: {backgroundColor: 'transparent'},
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
            <Tab.Screen name="Arcana" component={ArcanaStackNavigator} options={{title: t('nav.arcana')}} />
            <Tab.Screen name="Decks" component={DeckGalleryScreen} options={{title: t('nav.decks')}} />
            <Tab.Screen name="History" component={HistoryListScreen} options={{title: t('nav.history')}} />
            <Tab.Screen name="Settings" component={SettingsScreen} options={{title: t('nav.settings')}} />
        </Tab.Navigator>
    );
};

const AppNavigation = () => {
    const {settings, loading: settingsLoading, setSetting} = useSettings();
    const {user, loading: authLoading} = useAuth();
    const systemScheme = useColorScheme();
    const {t} = useTranslation();

    useEffect(() => {
        if (settingsLoading) {
            return;
        }
        I18n.changeLanguage(settings.language).catch(() => {});
    }, [settings.language, settingsLoading]);

    useEffect(() => {
        if (!user) {
            return;
        }
        if (user.hasPremium === settings.hasPremium) {
            return;
        }
        setSetting('hasPremium', user.hasPremium).catch(() => {});
    }, [setSetting, settings.hasPremium, user]);

    const colors = useAppColors();
    const scheme =
        settings.theme === 'system' ? systemScheme ?? 'light' : settings.theme === 'dark' ? 'dark' : 'light';
    const baseTheme = scheme === 'dark' ? NavigationDarkTheme : NavigationDefaultTheme;
    const theme = {
        ...baseTheme,
        colors: {
            ...baseTheme.colors,
            primary: colors.accent,
            background: 'transparent',
            card: colors.tabBar,
            text: colors.text,
            border: colors.gold,
            notification: colors.danger,
        },
    };

    if (settingsLoading || authLoading) {
        return (
            <AppBackground>
                <View style={{flex: 1, justifyContent: 'center', alignItems: 'center'}}>
                    <StatusBar barStyle="light-content" backgroundColor={colors.bg} />
                    <ActivityIndicator color={colors.accent} />
                </View>
            </AppBackground>
        );
    }

    const showDisclaimer = !settings.acceptedDisclaimer;
    const showAuth = !showDisclaimer && !user;

    return (
        <AppBackground>
            <NavigationContainer theme={theme} linking={linking}>
                <StatusBar barStyle="light-content" backgroundColor={colors.bg} />
                <RootStack.Navigator screenOptions={{headerShown: false, ...stackHeader(colors)}}>
                {showDisclaimer ? (
                    <RootStack.Screen name="Disclaimer" component={DisclaimerScreen} />
                ) : showAuth ? (
                    <>
                        <RootStack.Screen name="Auth" component={AuthScreen} />
                        <RootStack.Screen name="VerifyEmail" component={VerifyEmailScreen} />
                        <RootStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
                        <RootStack.Screen
                            name="LegalDocument"
                            component={LegalDocumentScreen}
                            options={legalScreenOptions((key) => t(key), colors)}
                        />
                    </>
                ) : (
                    <>
                        <RootStack.Screen name="Main" component={MainTabs} />
                        <RootStack.Screen
                            name="ArcanaBattle"
                            component={ArcanaBattleScreen}
                            options={{gestureEnabled: false}}
                        />
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
                        <RootStack.Screen
                            name="LegalDocument"
                            component={LegalDocumentScreen}
                            options={legalScreenOptions((key) => t(key), colors)}
                        />
                    </>
                )}
                </RootStack.Navigator>
            </NavigationContainer>
        </AppBackground>
    );
};

export default function App() {
    const [videoDone, setVideoDone] = useState(false);

    return (
        <GestureHandlerRootView style={{flex: 1}}>
            <SettingsProvider>
                <AuthProvider>
                    <DeckShopProvider>
                        <HistoryProvider>
                            <ArcanaProvider>
                                {videoDone ? <AppNavigation /> : null}
                            </ArcanaProvider>
                        </HistoryProvider>
                    </DeckShopProvider>
                </AuthProvider>
            </SettingsProvider>
            {videoDone ? null : <VideoSplash onDone={() => setVideoDone(true)} />}
        </GestureHandlerRootView>
    );
}
