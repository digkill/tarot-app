export type RootStackParamList = {
    Onboarding: undefined;
    Disclaimer: undefined;
    Main: undefined;
    Reading: {spreadId: string; deckId: string; readingId?: string};
    Interpretation: {readingId: string};
};

export type HomeStackParamList = {
    Home: undefined;
    SpreadCatalog: undefined;
};

export type AppTabsParamList = {
    Explore: undefined;
    Decks: undefined;
    History: undefined;
    Settings: undefined;
};
