import type {LegalDocumentId} from '../entities';

export type RootStackParamList = {
    Disclaimer: undefined;
    Auth: undefined;
    VerifyEmail: {email: string};
    ForgotPassword: {email?: string};
    LegalDocument: {doc: LegalDocumentId};
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
