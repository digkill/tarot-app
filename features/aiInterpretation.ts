import type {LanguagePreference, ReadingAiInsight} from '../entities';
import {apiRequest, deviceTimezone} from './apiClient';

type PremiumEntry = {
    positionIndex: number;
    positionTitle: string;
    positionDescription: string;
    cardName: string;
    uprightMeaning: string;
    reversedMeaning: string;
    uprightKeywords: string[];
    reversedKeywords: string[];
    isReversed: boolean;
};

export type PremiumInterpretationRequest = {
    language: LanguagePreference;
    spreadId: string;
    spreadName: string;
    spreadDescription?: string;
    entries: PremiumEntry[];
};

type InterpretApiResponse = {
    summary: string;
    positions: Array<{
        positionIndex: number;
        positionTitle: string;
        cardName: string;
        orientation: string;
        meaning: string;
    }>;
};

export const fetchPremiumInterpretation = async (
    request: PremiumInterpretationRequest,
): Promise<ReadingAiInsight> => {
    const data = await apiRequest<InterpretApiResponse>(
        `/api/v1/interpretations?tz=${encodeURIComponent(deviceTimezone())}`,
        {
            method: 'POST',
            body: {
                spreadId: request.spreadId,
                spreadName: request.spreadName,
                spreadDescription: request.spreadDescription ?? '',
                language: request.language,
                cards: request.entries,
            },
        },
    );

    return {
        summary: data.summary,
        positions: data.positions.map((item) => ({
            positionIndex: item.positionIndex,
            positionTitle: item.positionTitle,
            cardName: item.cardName,
            orientation: item.orientation === 'reversed' ? 'reversed' : 'upright',
            meaning: item.meaning,
        })),
        model: '',
        language: request.language,
        generatedAt: Date.now(),
    };
};
