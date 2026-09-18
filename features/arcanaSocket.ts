import type {ArcanaGameView, ArcanaTarget} from './arcanaApi';
import {ARCANA_PROTOCOL, arcanaSocketUrl} from './arcanaApi';

/**
 * The WebSocket side of Arcana Clash: one connection, authenticated once with
 * an access token, carrying player intents up and server state down.
 *
 * React Native's WebSocket cannot send ping frames, so a dead link is noticed
 * by silence instead: the server sends state on every action and pings every
 * 15 s, and nothing at all for SILENCE_MS means the socket is gone.
 */

export type ArcanaServerFrame = {
    type: string;
    seq?: number;
    match_id?: string;
    payload?: unknown;
};

export type ArcanaHelloPayload = {
    user_id: string;
    protocol: number;
    rules_version: number;
    active_match?: string;
    emotes: string[];
};

export type ArcanaPlayerInfo = {id: string; hero: string; deck?: string};

export type ArcanaStartedPayload = {you: ArcanaPlayerInfo; opponent: ArcanaPlayerInfo};

export type ArcanaStatePayload = {
    state: ArcanaGameView;
    server_time: number;
    deadline_at: number;
    opponent_connected: boolean;
    your_deck?: string;
    opponent_deck?: string;
};

export type ArcanaFinishedPayload = {
    winner?: string;
    reason: string;
    result: 'win' | 'loss' | 'none';
    state: ArcanaGameView;
};

export type ArcanaErrorPayload = {code: string; message: string; ref?: string};

export type ArcanaEmotePayload = {player: string; emote: string};

const SILENCE_MS = 90_000;

type Handlers = {
    onFrame: (frame: ArcanaServerFrame) => void;
    /** Called on every disconnect; `code` 4001 means another device took over. */
    onClosed: (code: number, reason: string) => void;
};

export class ArcanaSocket {
    private socket: WebSocket | null = null;
    private silenceTimer: ReturnType<typeof setTimeout> | null = null;
    private closedByUs = false;

    constructor(private readonly handlers: Handlers) {}

    connect(token: string) {
        this.close();
        this.closedByUs = false;
        const socket = new WebSocket(arcanaSocketUrl());
        this.socket = socket;

        socket.onopen = () => {
            this.send({type: 'hello', protocol: ARCANA_PROTOCOL, token});
            this.armSilenceTimer();
        };
        socket.onmessage = (event) => {
            this.armSilenceTimer();
            if (typeof event.data !== 'string') {
                return;
            }
            try {
                this.handlers.onFrame(JSON.parse(event.data) as ArcanaServerFrame);
            } catch {
                // A frame we cannot parse is not worth dropping the match for.
            }
        };
        socket.onerror = () => {
            this.drop(0, 'error');
        };
        socket.onclose = (event) => {
            this.drop(event?.code ?? 0, event?.reason ?? '');
        };
    }

    /** True while the socket is open and the hello has been sent. */
    get isOpen(): boolean {
        return this.socket?.readyState === WebSocket.OPEN;
    }

    close() {
        this.clearSilenceTimer();
        const socket = this.socket;
        this.socket = null;
        if (!socket) {
            return;
        }
        this.closedByUs = true;
        socket.onopen = null;
        socket.onmessage = null;
        socket.onerror = null;
        socket.onclose = null;
        try {
            socket.close();
        } catch {
            // Already gone.
        }
    }

    private drop(code: number, reason: string) {
        if (this.closedByUs || !this.socket) {
            return;
        }
        this.clearSilenceTimer();
        this.socket = null;
        this.handlers.onClosed(code, reason);
    }

    private armSilenceTimer() {
        this.clearSilenceTimer();
        this.silenceTimer = setTimeout(() => this.drop(0, 'silence'), SILENCE_MS);
    }

    private clearSilenceTimer() {
        if (this.silenceTimer) {
            clearTimeout(this.silenceTimer);
            this.silenceTimer = null;
        }
    }

    private send(message: Record<string, unknown>) {
        if (this.socket?.readyState !== WebSocket.OPEN) {
            return;
        }
        this.socket.send(JSON.stringify(message));
    }

    joinQueue(hero: string | null, deck: string | null) {
        this.send({type: 'queue.join', hero: hero ?? undefined, deck: deck ?? undefined});
    }

    leaveQueue() {
        this.send({type: 'queue.leave'});
    }

    resume(matchId: string) {
        this.send({type: 'match.resume', match_id: matchId});
    }

    mulligan(matchId: string, cardUids: string[]) {
        this.send({type: 'mulligan', match_id: matchId, card_uids: cardUids});
    }

    playCard(matchId: string, cardUid: string, target?: ArcanaTarget) {
        this.send({type: 'card.play', match_id: matchId, card_uid: cardUid, target});
    }

    ability(matchId: string, target?: ArcanaTarget) {
        this.send({type: 'hero.ability', match_id: matchId, target});
    }

    ultimate(matchId: string, target?: ArcanaTarget) {
        this.send({type: 'hero.ultimate', match_id: matchId, target});
    }

    endTurn(matchId: string) {
        this.send({type: 'turn.end', match_id: matchId});
    }

    surrender(matchId: string) {
        this.send({type: 'match.surrender', match_id: matchId});
    }

    emote(matchId: string, emote: string) {
        this.send({type: 'player.emote', match_id: matchId, emote});
    }
}
