import {apiRequest, getApiBaseUrl} from './apiClient';

/**
 * Arcana Clash lives on its own Go service, mounted under the main API host
 * at /api/v1/arcana. The client only ever renders what the server sends: it
 * computes no damage, mana or card draws of its own.
 */
export const ARCANA_PROTOCOL = 1;

export type ArcanaTargetKind = 'enemy' | 'self' | 'hand_card' | 'discard_card' | 'none';

export type ArcanaTarget = {
    kind?: ArcanaTargetKind;
    cardUid?: string;
};

export type ArcanaOp = {
    kind: string;
    who?: string;
    amount?: number;
    bonus?: number;
    turns?: number;
    chance?: number;
    status?: string;
    element?: string;
    pierce?: boolean;
};

export type ArcanaSide = {
    targets?: ArcanaTargetKind[];
    ops: ArcanaOp[];
};

export type ArcanaCardDef = {
    id: string;
    suit: string;
    cost: number;
    upright: ArcanaSide;
    reversed: ArcanaSide;
};

export type ArcanaHeroDef = {
    id: string;
    hp: number;
    passive: string;
    ability: {cost: number; chargeWithSouls?: boolean; side: ArcanaSide};
    ultimate: {cost: number; chargeWithSouls?: boolean; side: ArcanaSide};
};

export type ArcanaRules = {
    deck_size: number;
    start_hand: number;
    hand_limit: number;
    mana_cap: number;
    mulligan_max: number;
    turn_seconds: number;
    mulligan_seconds: number;
    reconnect_seconds: number;
};

export type ArcanaCatalog = {
    protocol_version: number;
    rules_version: number;
    rules: ArcanaRules;
    heroes: ArcanaHeroDef[];
    cards: ArcanaCardDef[];
    emotes: string[];
};

export type ArcanaCardView = {
    uid: string;
    card_id?: string;
    reversed?: boolean;
    cost: number;
    base_cost: number;
    hidden?: boolean;
    playable?: boolean;
    targets?: ArcanaTargetKind[];
};

export type ArcanaStatusView = {
    name: string;
    amount: number;
    turns?: number;
};

export type ArcanaHeroView = {
    id: string;
    hp: number;
    max_hp: number;
    armor: number;
    charge: number;
    ultimate_cost: number;
    souls: number;
    ultimate_ready: boolean;
};

export type ArcanaEvent = {
    kind: string;
    player?: string;
    amount?: number;
    absorbed?: number;
    crit?: boolean;
    element?: string;
    status?: string;
    card_id?: string;
    card_uid?: string;
    reason?: string;
};

export type ArcanaLogEntry = {
    seq: number;
    player?: string;
    action: string;
    card_uid?: string;
    card_id?: string;
    reversed?: boolean;
    hidden?: boolean;
    events?: ArcanaEvent[];
};

export type ArcanaGameView = {
    rules_version: number;
    seq: number;
    phase: 'mulligan' | 'playing' | 'finished';
    turn: number;
    active_player: string;
    you: {
        id: string;
        hero: ArcanaHeroView;
        mana: number;
        max_mana: number;
        hand: ArcanaCardView[];
        deck_count: number;
        discard: ArcanaCardView[];
        statuses: ArcanaStatusView[];
        ability_cost: number;
        ability_usable: boolean;
        ability_targets?: ArcanaTargetKind[];
        ultimate_usable: boolean;
        ultimate_targets?: ArcanaTargetKind[];
        mulliganed: boolean;
        timeout_streak: number;
    };
    opponent: {
        id: string;
        hero: ArcanaHeroView;
        mana: number;
        max_mana: number;
        hand_count: number;
        deck_count: number;
        discard: ArcanaCardView[];
        statuses: ArcanaStatusView[];
        mulliganed: boolean;
    };
    log: ArcanaLogEntry[];
    winner?: string;
    reason?: string;
};

export type ArcanaMatchSummary = {
    id: string;
    opponent_id?: string;
    hero: string;
    opponent_hero: string;
    deck?: string;
    opponent_deck?: string;
    result: 'win' | 'loss' | 'none';
    reason: string;
    turns: number;
    started_at: string;
    finished_at: string;
    duration_ms: number;
};

/** The hero's portrait and name come from the tarot card it is drawn from. */
export const heroCardId = (heroId: string): string => (heroId === 'strength' ? 'the_strength' : heroId);

export const fetchArcanaCatalog = (): Promise<ArcanaCatalog> =>
    apiRequest<ArcanaCatalog>('/api/v1/arcana/catalog', {auth: false});

export const fetchArcanaMatches = (limit = 30): Promise<ArcanaMatchSummary[]> =>
    apiRequest<{matches: ArcanaMatchSummary[] | null}>(`/api/v1/arcana/matches?limit=${limit}`).then(
        (response) => response.matches ?? [],
    );

export const arcanaSocketUrl = (): string =>
    `${getApiBaseUrl().replace(/^http/, 'ws')}/api/v1/arcana/ws`;
