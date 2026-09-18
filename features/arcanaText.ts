import type {TFunction} from 'i18next';
import type {LanguagePreference} from '../entities';
import {loadDeck} from '../utils/decks';
import type {ArcanaLogEntry, ArcanaOp, ArcanaSide} from './arcanaApi';

/**
 * Turns the server's declarative card effects into sentences, and match log
 * entries into lines. Numbers come from the catalog, wording from the
 * translations — so a new card needs no new client code.
 */

const nameCache: Partial<Record<LanguagePreference, Record<string, string>>> = {};

/** Card names by server card id (the image basename, e.g. `the_tower`). */
export const arcanaCardNames = (language: LanguagePreference): Record<string, string> => {
    const cached = nameCache[language];
    if (cached) {
        return cached;
    }
    const names: Record<string, string> = {};
    loadDeck(language).forEach((card) => {
        names[card.image.replace(/\.[^.]+$/, '')] = card.name;
    });
    nameCache[language] = names;
    return names;
};

export const arcanaCardImageFile = (cardId: string): string => `${cardId}.jpeg`;

export const describeArcanaOp = (op: ArcanaOp, t: TFunction): string | null => {
    const amount = op.amount ?? 0;
    const status = op.status ? t(`arcana.status.${op.status}`) : '';
    switch (op.kind) {
        case 'damage': {
            if (op.who === 'self') {
                return t('arcana.op.damageSelf', {amount});
            }
            let text = t(op.element === 'magic' ? 'arcana.op.damageMagic' : 'arcana.op.damagePhysical', {amount});
            if (op.pierce) {
                text += `, ${t('arcana.op.pierce')}`;
            }
            if (op.chance) {
                text += ` (${t('arcana.op.crit', {chance: op.chance})})`;
            }
            return text;
        }
        case 'combo_damage':
            return t('arcana.op.combo', {amount, bonus: op.bonus ?? 0, status});
        case 'lifesteal':
            return t('arcana.op.lifesteal', {amount});
        case 'heal':
            return t('arcana.op.heal', {amount});
        case 'armor':
            return t('arcana.op.armor', {amount});
        case 'status': {
            const text = t(op.who === 'self' ? 'arcana.op.statusSelf' : 'arcana.op.statusTarget', {status, amount});
            return op.turns ? t('arcana.op.statusTurns', {text, count: op.turns}) : text;
        }
        case 'cleanse':
            return t('arcana.op.cleanse');
        case 'draw':
            return t('arcana.op.draw', {amount});
        case 'mana':
            return t('arcana.op.mana', {amount});
        case 'discard_random':
            return t('arcana.op.discardRandom');
        case 'redraw_hand':
            return op.bonus ? t('arcana.op.redrawBonus', {bonus: op.bonus}) : t('arcana.op.redraw');
        case 'repeat_last':
            return t(op.who === 'opponent' ? 'arcana.op.repeatOpponent' : 'arcana.op.repeatSelf');
        case 'purge':
            return t(op.bonus ? 'arcana.op.purgeTarget' : 'arcana.op.purgeBoth', {amount});
        case 'wheel':
            return t('arcana.op.wheel');
        case 'discount_hand':
            return t('arcana.op.discountHand', {amount});
        case 'recall':
            return t('arcana.op.recall');
        case 'recycle':
            return t('arcana.op.recycle');
        case 'copy_card':
            return t('arcana.op.copyCard');
        case 'reap':
            return t('arcana.op.reap', {amount});
        case 'soul_harvest':
            return t('arcana.op.soulHarvest', {amount});
        default:
            return null;
    }
};

export const describeArcanaSide = (side: ArcanaSide, t: TFunction): string[] =>
    side.ops.map((op) => describeArcanaOp(op, t)).filter((line): line is string => Boolean(line));

export const arcanaLogLine = (
    entry: ArcanaLogEntry,
    meId: string,
    cardName: (id: string) => string,
    t: TFunction,
): string | null => {
    const player = entry.player === meId ? t('arcana.you') : t('arcana.opponent');
    switch (entry.action) {
        case 'card.play':
            return entry.card_id
                ? t('arcana.log.card', {player, card: cardName(entry.card_id)})
                : t('arcana.log.hiddenCard', {player});
        case 'hero.ability':
            return t('arcana.log.ability', {player});
        case 'hero.ultimate':
            return t('arcana.log.ultimate', {player});
        case 'turn.end':
            return t('arcana.log.endTurn', {player});
        case 'turn.timeout':
            return t('arcana.log.timeout', {player});
        case 'mulligan':
            return t('arcana.log.mulligan', {player});
        case 'surrender':
            return t('arcana.log.surrender', {player});
        case 'mulligan.timeout':
            return t('arcana.log.start');
        default:
            return null;
    }
};

/** Totals of what an action did, as "−5" / "+3" next to the log line. */
export const arcanaEventSummary = (entry: ArcanaLogEntry, meId: string): string => {
    const parts: string[] = [];
    (entry.events ?? []).forEach((event) => {
        const side = event.player === meId ? '↓' : '↑';
        if (event.kind === 'damage' && (event.amount ?? 0) > 0) {
            parts.push(`${side}−${event.amount}${event.crit ? '!' : ''}`);
        }
        if (event.kind === 'heal' && (event.amount ?? 0) > 0) {
            parts.push(`${side}+${event.amount}`);
        }
    });
    return parts.slice(0, 3).join(' ');
};

export const ARCANA_STATUS_ORDER = [
    'burn',
    'regen',
    'strength',
    'weakness',
    'vulnerable',
    'ward',
    'riposte',
    'empower',
    'fury',
    'surge',
    'exhausted',
    'confused',
    'veil',
    'bond',
] as const;

const NEGATIVE = new Set(['burn', 'weakness', 'vulnerable', 'exhausted', 'confused']);

export const isNegativeArcanaStatus = (name: string): boolean => NEGATIVE.has(name);

export const arcanaStatusIcon = (name: string): string => {
    switch (name) {
        case 'burn':
            return 'flame';
        case 'regen':
            return 'leaf';
        case 'strength':
            return 'barbell';
        case 'weakness':
            return 'arrow-down-circle';
        case 'vulnerable':
            return 'alert-circle';
        case 'ward':
            return 'shield-half';
        case 'riposte':
            return 'return-up-back';
        case 'empower':
            return 'sparkles';
        case 'fury':
            return 'flash';
        case 'surge':
            return 'water';
        case 'exhausted':
            return 'moon';
        case 'confused':
            return 'help-circle';
        case 'veil':
            return 'eye-off';
        case 'bond':
            return 'heart';
        default:
            return 'ellipse';
    }
};
