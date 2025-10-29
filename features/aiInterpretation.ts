import OpenAI from 'openai';
import type {LanguagePreference, ReadingAiInsight} from '../entities';

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
    spreadName: string;
    spreadDescription?: string;
    entries: PremiumEntry[];
};

const LANGUAGE_LABEL: Record<LanguagePreference, string> = {
    en: 'English',
    ru: 'Russian',
    th: 'Thai',
    zh: 'Simplified Chinese',
};

type MaybeProcess = {
    env?: Record<string, string | undefined>;
};

const getEnv = (key: string): string | undefined => {
    const nodeProcess = (globalThis as {process?: MaybeProcess}).process;
    return nodeProcess?.env?.[key];
};

const resolveApiKey = (): string | undefined => {
    const envKey = getEnv('OPENAI_API_KEY');
    if (envKey) {
        return envKey;
    }
    const globalKey = (globalThis as {OPENAI_API_KEY?: string}).OPENAI_API_KEY;
    return globalKey;
};

export class MissingOpenAiKeyError extends Error {
    constructor() {
        super('Missing OpenAI API key');
        this.name = 'MissingOpenAiKeyError';
    }
}

let client: OpenAI | null = null;

const getClient = () => {
    const apiKey = resolveApiKey();
    if (!apiKey) {
        throw new MissingOpenAiKeyError();
    }
    if (!client) {
        client = new OpenAI({
            apiKey,
            dangerouslyAllowBrowser: true,
        });
    }
    return client;
};

const MODEL = getEnv('OPENAI_TAROT_MODEL') || 'gpt-4o-mini';

const buildPrompt = ({entries, spreadName, spreadDescription, language}: PremiumInterpretationRequest): string => {
    const languageLabel = LANGUAGE_LABEL[language] ?? 'English';
    const introLines = [
        `Spread: ${spreadName}`,
        spreadDescription ? `Spread overview: ${spreadDescription}` : null,
        `Respond in ${languageLabel}.`,
    ]
        .filter(Boolean)
        .join('\n');

    const cardLines = entries
        .map((entry) => {
            const orientation = entry.isReversed ? 'reversed' : 'upright';
            const uprightKeywords = entry.uprightKeywords.join(', ') || 'n/a';
            const reversedKeywords = entry.reversedKeywords.join(', ') || 'n/a';
            return [
                `Position ${entry.positionIndex}: ${entry.positionTitle}`,
                `Position detail: ${entry.positionDescription}`,
                `Card: ${entry.cardName}`,
                `Orientation: ${orientation}`,
                `Upright meaning: ${entry.uprightMeaning}`,
                `Reversed meaning: ${entry.reversedMeaning}`,
                `Upright keywords: ${uprightKeywords}`,
                `Reversed keywords: ${reversedKeywords}`,
            ].join('\n');
        })
        .join('\n\n');

    return `${introLines}\n\nCards:\n${cardLines}\n\nGenerate concise, empowering guidance with practical advice.`;
};

const schema = {
    name: 'tarot_interpretation',
    schema: {
        type: 'object',
        properties: {
            summary: {
                type: 'string',
                description: 'Overall insight summary for the entire spread, 3-5 sentences.',
            },
            positions: {
                type: 'array',
                description: 'Detailed guidance for each spread position.',
                items: {
                    type: 'object',
                    properties: {
                        positionIndex: {type: 'integer'},
                        positionTitle: {type: 'string'},
                        cardName: {type: 'string'},
                        orientation: {type: 'string', enum: ['upright', 'reversed']},
                        meaning: {
                            type: 'string',
                            description: '1-3 sentences with actionable advice.',
                        },
                    },
                    required: ['positionIndex', 'positionTitle', 'cardName', 'orientation', 'meaning'],
                    additionalProperties: false,
                },
            },
        },
        required: ['summary', 'positions'],
        additionalProperties: false,
    },
    strict: true,
} as const;

export const fetchPremiumInterpretation = async (
    request: PremiumInterpretationRequest,
): Promise<ReadingAiInsight> => {
    const clientInstance = getClient();
    const prompt = buildPrompt(request);

    const response = await clientInstance.responses.create({
        model: MODEL,
        input: [
            {
                role: 'system',
                content:
                    'You are a compassionate tarot expert. Provide grounded, clear guidance without disclaimers. Output only JSON following the provided schema.',
            },
            {
                role: 'user',
                content: prompt,
            },
        ],
        response_format: {type: 'json_schema', json_schema: schema},
    });

    const text = response.output_text?.trim();
    if (!text) {
        throw new Error('Empty response from OpenAI');
    }

    let parsed: {summary: string; positions: Array<{positionIndex: number; positionTitle: string; cardName: string; orientation: 'upright' | 'reversed'; meaning: string}>};
    try {
        parsed = JSON.parse(text);
    } catch (error) {
        throw new Error('Failed to parse OpenAI response');
    }

    return {
        summary: parsed.summary,
        positions: parsed.positions.map((item) => ({
            positionIndex: item.positionIndex,
            positionTitle: item.positionTitle,
            cardName: item.cardName,
            orientation: item.orientation,
            meaning: item.meaning,
        })),
        model: MODEL,
        language: request.language,
        generatedAt: Date.now(),
    };
};
