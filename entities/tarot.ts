export type Arcana = 'major' | 'minor';

export type Suit = 'wands' | 'cups' | 'swords' | 'pentacles';

export type TarotCardSide = {
    keywords: string[];
    general: string;
    love?: string;
    work?: string;
    health?: string;
    advice?: string;
};

export type Card = {
    id: string;
    deckId: string;
    name: string;
    arcana: Arcana;
    suit?: Suit;
    number?: number;
    image: string;
    upright: TarotCardSide;
    reversed: TarotCardSide;
};

export type SpreadCategory =
    | 'basic'
    | 'love'
    | 'career'
    | 'year'
    | 'weekly'
    | 'spiritual'
    | 'custom';

export type SpreadPosition = {
    index: number;
    titleKey: string;
    descriptionKey: string;
    x: number;
    y: number;
    rotation?: number;
    reversedAllowed?: boolean;
};

export type Spread = {
    id: string;
    nameKey: string;
    descriptionKey: string;
    positions: SpreadPosition[];
    minCards: number;
    maxCards: number;
    category: SpreadCategory;
    premium?: boolean;
    featured?: boolean;
};

export type ReadingCard = {
    positionIndex: number;
    cardId: string;
    isReversed: boolean;
};

export type ReadingAiInsight = {
    summary: string;
    positions: Array<{
        positionIndex: number;
        positionTitle: string;
        cardName: string;
        orientation: 'upright' | 'reversed';
        meaning: string;
    }>;
    model: string;
    language: LanguagePreference;
    generatedAt: number;
};

export type Reading = {
    id: string;
    spreadId: string;
    deckId: string;
    drawnAt: number;
    items: ReadingCard[];
    summaryText: string;
    notes?: string;
    tags?: string[];
    favorite?: boolean;
    aiInsights?: ReadingAiInsight;
};

export type ThemePreference = 'light' | 'dark' | 'system';

export type LanguagePreference = 'en' | 'ru' | 'th' | 'zh';

export type Settings = {
    language: LanguagePreference;
    theme: ThemePreference;
    disableAnimations: boolean;
    disableSounds: boolean;
    reversedChance: number;
    showMysticMode: boolean;
    hasPremium: boolean;
    dailyReminder?: boolean;
    hasCompletedOnboarding?: boolean;
    acceptedDisclaimer?: boolean;
};
