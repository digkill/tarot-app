import type {LegalDocumentId} from '../entities';

export type RootStackParamList = {
    Disclaimer: undefined;
    Auth: undefined;
    VerifyEmail: {email: string};
    ForgotPassword: {email?: string};
    LegalDocument: {doc: LegalDocumentId};
    Main: undefined;
    Reading: {
        spreadId: string;
        deckId: string;
        readingId?: string;
        quotaGranted?: boolean;
        readingKind?: 'daily' | 'bonus';
    };
    Interpretation: {readingId: string};
};

export type HomeStackParamList = {
    Home: undefined;
    SpreadCatalog: undefined;
};

export type ArcanaStackParamList = {
    ArcanaHome: undefined;
    ArcanaBattle: undefined;
    ArcanaRules: undefined;
    ArcanaHeroes: undefined;
    ArcanaMatchHistory: undefined;
};

export type AppTabsParamList = {
    Explore: undefined;
    Arcana: undefined;
    Decks: {slug?: string} | undefined;
    History: undefined;
    Settings: undefined;
};
