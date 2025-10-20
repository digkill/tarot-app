import tarotEn from '../data/tarot_en.json';
import tarotRu from '../data/tarot_ru.json';
import tarotTh from '../data/tarot_th.json';
import type {Card, LanguagePreference, TarotCardSide} from '../entities';

type RawCard = {
    id?: string;
    name: string;
    image: string;
    upright: string | TarotCardSide;
    reversed: string | TarotCardSide;
    arcana?: Card['arcana'];
    suit?: Card['suit'];
    number?: number;
};

const DECK_ID = 'rws';

const rawDecks: Record<LanguagePreference, RawCard[]> = {
    en: tarotEn as RawCard[],
    ru: tarotRu as RawCard[],
    th: tarotTh as RawCard[],
};

const toSide = (value: string | TarotCardSide): TarotCardSide =>
    typeof value === 'string'
        ? {keywords: [], general: value}
        : {
              keywords: value.keywords ?? [],
              general: value.general,
              love: value.love,
              work: value.work,
              health: value.health,
              advice: value.advice,
          };

const normalizeId = (card: RawCard, index: number): string => {
    if (card.id) return card.id;
    const slug = card.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
    return slug ? `${DECK_ID}-${slug}` : `${DECK_ID}-${index}`;
};

export const loadDeck = (language: LanguagePreference): Card[] =>
    rawDecks[language].map((card, index) => ({
        id: normalizeId(card, index),
        deckId: DECK_ID,
        name: card.name,
        arcana: card.arcana ?? inferArcana(card.name),
        suit: card.suit,
        number: card.number,
        image: card.image,
        upright: toSide(card.upright),
        reversed: toSide(card.reversed),
    }));

const MAJOR_ARCANA_NAMES = [
    'The Fool',
    'The Magician',
    'The High Priestess',
    'The Empress',
    'The Emperor',
    'The Hierophant',
    'The Lovers',
    'The Chariot',
    'Strength',
    'Justice',
    'The Hermit',
    'Wheel of Fortune',
    'The Hanged Man',
    'Death',
    'Temperance',
    'The Devil',
    'The Tower',
    'The Star',
    'The Moon',
    'The Sun',
    'Judgement',
    'The World',
];

const inferArcana = (name: string): Card['arcana'] => {
    return MAJOR_ARCANA_NAMES.some((title) => name.toLowerCase().includes(title.toLowerCase()))
        ? 'major'
        : 'minor';
};

export const findCardById = (deck: Card[], id: string): Card | undefined =>
    deck.find((card) => card.id === id);
