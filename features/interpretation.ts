import type {Card, LanguagePreference, Reading, Spread, SpreadPosition} from '../entities';
import {SPREADS} from '../data';
import {I18n} from '../i18n';
import {findCardById, loadDeck} from '../utils/decks';

type Entry = {
    card: Card;
    position: SpreadPosition;
    isReversed: boolean;
};

type Translator = (key: string, vars?: Record<string, any>) => string;

export type Interpretation = {
    summary: string;
    positions: Array<{
        title: string;
        description: string;
        card: Card;
        isReversed: boolean;
        narrative: string;
    }>;
    keywords: string[];
};

const defaultTranslate: Translator = (key) => key;

const buildNarrative = (entry: Entry, t: Translator): string => {
    const cardSide = entry.isReversed ? entry.card.reversed : entry.card.upright;
    const positionLabel = t(entry.position.titleKey);
    const positionDescription = t(entry.position.descriptionKey);
    return `${positionLabel}: ${positionDescription} — ${cardSide.general}`;
};

const collectKeywords = (entries: Entry[]): string[] => {
    const keywords = entries.flatMap((entry) =>
        entry.isReversed ? entry.card.reversed.keywords : entry.card.upright.keywords,
    );
    return Array.from(new Set(keywords)).slice(0, 12);
};

export const hasBrokenSummary = (text: string | undefined): boolean =>
    Boolean(text && /\{(?:count|spread|price|model)\}/.test(text));

export const getReadingSummary = (reading: Reading, language: LanguagePreference): string => {
    if (reading.aiInsights?.summary) {
        return reading.aiInsights.summary;
    }

    const spread = SPREADS.find((item) => item.id === reading.spreadId);
    if (!spread) {
        return reading.summaryText ?? '';
    }

    const deck = loadDeck(language);
    const entries = reading.items
        .map((item) => {
            const position = spread.positions.find((pos) => pos.index === item.positionIndex);
            const card = findCardById(deck, item.cardId);
            if (!position || !card) return null;
            return {position, card, isReversed: item.isReversed};
        })
        .filter((entry): entry is Entry => entry !== null);

    if (!entries.length) {
        return reading.summaryText ?? '';
    }

    const translate: Translator = (key, vars) => I18n.t(key, {lng: language, ...(vars ?? {})});
    return generateInterpretation(spread, entries, translate).summary;
};

export const generateInterpretation = (
    spread: Spread,
    entries: Entry[],
    translate: Translator = defaultTranslate,
): Interpretation => {
    const t = translate;
    const positions = entries.map((entry) => ({
        title: t(entry.position.titleKey),
        description: t(entry.position.descriptionKey),
        card: entry.card,
        isReversed: entry.isReversed,
        narrative: buildNarrative(entry, t),
    }));

    const keywords = collectKeywords(entries);
    const spreadName = t(spread.nameKey);
    const intro = t('interpretation.summaryIntro', {
        spread: spreadName,
        count: entries.length,
    });
    const highlights = positions
        .slice(0, 3)
        .map((item) => item.narrative)
        .join(' ');
    const closing = t('interpretation.summaryClosing');

    return {
        summary: `${intro} ${highlights} ${closing}`.trim(),
        positions,
        keywords,
    };
};
