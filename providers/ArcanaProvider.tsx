import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useRef,
    useState,
    type ReactNode,
} from 'react';
import {AppState} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {socketAccessToken} from '../features/apiClient';
import {
    fetchArcanaCatalog,
    fetchArcanaMatches,
    type ArcanaCatalog,
    type ArcanaGameView,
    type ArcanaMatchSummary,
    type ArcanaTarget,
} from '../features/arcanaApi';
import {
    ArcanaSocket,
    type ArcanaErrorPayload,
    type ArcanaFinishedPayload,
    type ArcanaHelloPayload,
    type ArcanaServerFrame,
    type ArcanaStartedPayload,
    type ArcanaStatePayload,
} from '../features/arcanaSocket';
import {useAuth} from './AuthProvider';

/**
 * Arcana Clash client state. The server is authoritative: this provider
 * forwards intents, shows the state it receives, ignores any state older than
 * what it already has (by `seq`), and resumes the match after a dropped
 * connection inside the server's reconnect window.
 */

const HERO_KEY = 'arcana.hero';
const MAX_RECONNECTS = 8;

type Stage = 'lobby' | 'searching' | 'match' | 'finished';

export type ArcanaEmoteBubble = {id: number; mine: boolean; emote: string};

type ArcanaValue = {
    stage: Stage;
    online: boolean;
    catalog: ArcanaCatalog | null;
    catalogFailed: boolean;
    view: ArcanaGameView | null;
    started: ArcanaStartedPayload | null;
    matchId: string | null;
    deadlineAt: number | null;
    opponentConnected: boolean;
    opponentDeck: string | null;
    finished: ArcanaFinishedPayload | null;
    resumableMatch: string | null;
    /** The last rejected action or connection problem, with a changing id so
     * the same code twice is shown twice. */
    lastError: {code: string; id: number} | null;
    emotes: ArcanaEmoteBubble[];
    hero: string | null;
    setHero: (hero: string | null) => void;
    loadCatalog: () => Promise<void>;
    matchHistory: (limit?: number) => Promise<ArcanaMatchSummary[]>;
    enter: () => void;
    leave: () => void;
    battle: (deck: string | null) => void;
    cancelSearch: () => void;
    resume: () => void;
    mulligan: (cardUids: string[]) => void;
    playCard: (cardUid: string, target?: ArcanaTarget) => void;
    useAbility: (target?: ArcanaTarget) => void;
    useUltimate: (target?: ArcanaTarget) => void;
    endTurn: () => void;
    surrender: () => void;
    sendEmote: (emote: string) => void;
    closeResult: () => void;
};

const ArcanaContext = createContext<ArcanaValue | null>(null);

export const ArcanaProvider = ({children}: {children: ReactNode}) => {
    const {user} = useAuth();
    const [stage, setStage] = useState<Stage>('lobby');
    const [online, setOnline] = useState(false);
    const [catalog, setCatalog] = useState<ArcanaCatalog | null>(null);
    const [catalogFailed, setCatalogFailed] = useState(false);
    const [view, setView] = useState<ArcanaGameView | null>(null);
    const [started, setStarted] = useState<ArcanaStartedPayload | null>(null);
    const [matchId, setMatchId] = useState<string | null>(null);
    const [deadlineAt, setDeadlineAt] = useState<number | null>(null);
    const [opponentConnected, setOpponentConnected] = useState(true);
    const [opponentDeck, setOpponentDeck] = useState<string | null>(null);
    const [finished, setFinished] = useState<ArcanaFinishedPayload | null>(null);
    const [resumableMatch, setResumableMatch] = useState<string | null>(null);
    const [lastError, setLastError] = useState<{code: string; id: number} | null>(null);
    const [emotes, setEmotes] = useState<ArcanaEmoteBubble[]>([]);
    const [hero, setHeroState] = useState<string | null>(null);

    const socketRef = useRef<ArcanaSocket | null>(null);
    const stageRef = useRef<Stage>('lobby');
    const matchRef = useRef<string | null>(null);
    const viewSeqRef = useRef(0);
    const wantsQueueRef = useRef(false);
    const deckRef = useRef<string | null>(null);
    const keepConnectedRef = useRef(false);
    const attemptsRef = useRef(0);
    const forceRefreshRef = useRef(false);
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const errorIdRef = useRef(0);
    const emoteIdRef = useRef(0);

    const setStageBoth = useCallback((next: Stage) => {
        stageRef.current = next;
        setStage(next);
    }, []);

    const raiseError = useCallback((code: string) => {
        errorIdRef.current += 1;
        setLastError({code, id: errorIdRef.current});
    }, []);

    useEffect(() => {
        AsyncStorage.getItem(HERO_KEY)
            .then((value) => setHeroState(value))
            .catch(() => {});
    }, []);

    const setHero = useCallback((next: string | null) => {
        setHeroState(next);
        if (next) {
            AsyncStorage.setItem(HERO_KEY, next).catch(() => {});
        } else {
            AsyncStorage.removeItem(HERO_KEY).catch(() => {});
        }
    }, []);

    const loadCatalog = useCallback(async () => {
        try {
            setCatalog(await fetchArcanaCatalog());
            setCatalogFailed(false);
        } catch {
            setCatalogFailed(true);
        }
    }, []);

    const clearReconnectTimer = () => {
        if (reconnectTimerRef.current) {
            clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = null;
        }
    };

    const handleFrame = useCallback(
        (frame: ArcanaServerFrame) => {
            const socket = socketRef.current;
            switch (frame.type) {
                case 'hello': {
                    const hello = frame.payload as ArcanaHelloPayload;
                    setOnline(true);
                    attemptsRef.current = 0;
                    forceRefreshRef.current = false;
                    setResumableMatch(hello.active_match ?? null);
                    setLastError((current) => (current?.code === 'connection_failed' ? null : current));
                    if (hello.active_match && (stageRef.current === 'match' || stageRef.current === 'searching')) {
                        matchRef.current = hello.active_match;
                        setMatchId(hello.active_match);
                        setStageBoth('match');
                        socket?.resume(hello.active_match);
                    } else if (stageRef.current === 'match') {
                        // The match ended while we were away.
                        setStageBoth('lobby');
                        setView(null);
                    } else if (wantsQueueRef.current && stageRef.current === 'searching') {
                        socket?.joinQueue(hero, deckRef.current);
                    }
                    break;
                }
                case 'queue.waiting':
                    setStageBoth('searching');
                    break;
                case 'queue.left':
                    if (stageRef.current === 'searching') {
                        setStageBoth('lobby');
                    }
                    break;
                case 'match.started': {
                    const payload = frame.payload as ArcanaStartedPayload;
                    wantsQueueRef.current = false;
                    matchRef.current = frame.match_id ?? null;
                    viewSeqRef.current = 0;
                    setMatchId(frame.match_id ?? null);
                    setStarted(payload);
                    setOpponentDeck(payload.opponent.deck ?? null);
                    setResumableMatch(null);
                    setView(null);
                    setFinished(null);
                    setOpponentConnected(true);
                    setStageBoth('match');
                    break;
                }
                case 'match.state': {
                    const payload = frame.payload as ArcanaStatePayload;
                    if (!matchRef.current) {
                        matchRef.current = frame.match_id ?? null;
                        setMatchId(frame.match_id ?? null);
                    }
                    if (frame.match_id !== matchRef.current) {
                        return;
                    }
                    if (payload.state.seq < viewSeqRef.current) {
                        return;
                    }
                    viewSeqRef.current = payload.state.seq;
                    setView(payload.state);
                    const offset = payload.server_time - Date.now();
                    setDeadlineAt(payload.deadline_at - offset);
                    setOpponentConnected(payload.opponent_connected);
                    setOpponentDeck(payload.opponent_deck ?? null);
                    setResumableMatch(null);
                    if (stageRef.current !== 'finished') {
                        setStageBoth('match');
                    }
                    break;
                }
                case 'match.finished': {
                    const payload = frame.payload as ArcanaFinishedPayload;
                    if (matchRef.current && frame.match_id !== matchRef.current) {
                        return;
                    }
                    setFinished(payload);
                    setView(payload.state);
                    setDeadlineAt(null);
                    setResumableMatch(null);
                    setStageBoth('finished');
                    break;
                }
                case 'player.emote': {
                    const payload = frame.payload as {player: string; emote: string};
                    emoteIdRef.current += 1;
                    const bubble = {
                        id: emoteIdRef.current,
                        mine: payload.player === user?.id,
                        emote: payload.emote,
                    };
                    setEmotes((current) => [...current, bubble]);
                    setTimeout(() => {
                        setEmotes((current) => current.filter((item) => item.id !== bubble.id));
                    }, 3000);
                    break;
                }
                case 'opponent.disconnected':
                    setOpponentConnected(false);
                    break;
                case 'opponent.reconnected':
                    setOpponentConnected(true);
                    break;
                case 'error': {
                    const payload = frame.payload as ArcanaErrorPayload;
                    if (payload.code === 'already_in_match') {
                        setStageBoth('match');
                        return;
                    }
                    raiseError(payload.code);
                    if (stageRef.current === 'searching' && payload.code === 'invalid_hero') {
                        setStageBoth('lobby');
                    }
                    break;
                }
                default:
                    break;
            }
        },
        [hero, raiseError, setStageBoth, user?.id],
    );

    const connect = useCallback(async () => {
        if (socketRef.current?.isOpen) {
            return;
        }
        const token = await socketAccessToken(forceRefreshRef.current);
        if (!token) {
            raiseError('connection_failed');
            return;
        }
        socketRef.current?.connect(token);
    }, [raiseError]);

    const scheduleReconnect = useCallback(
        (code: number, reason: string) => {
            setOnline(false);
            if (code === 4001) {
                // The same account started playing on another device.
                raiseError('session_replaced');
                if (stageRef.current === 'match') {
                    setStageBoth('lobby');
                }
                return;
            }
            if (reason === 'unauthorized') {
                forceRefreshRef.current = true;
            }
            const wanted =
                keepConnectedRef.current || stageRef.current === 'match' || stageRef.current === 'searching';
            if (!wanted) {
                return;
            }
            attemptsRef.current += 1;
            if (attemptsRef.current === 2) {
                raiseError('connection_failed');
            }
            if (attemptsRef.current > MAX_RECONNECTS) {
                if (stageRef.current === 'searching') {
                    setStageBoth('lobby');
                }
                return;
            }
            const delay = Math.min(2 ** (attemptsRef.current - 1), 8) * 1000;
            clearReconnectTimer();
            reconnectTimerRef.current = setTimeout(() => {
                connect().catch(() => {});
            }, delay);
        },
        [connect, raiseError, setStageBoth],
    );

    // One socket for the provider's lifetime; its handlers are stable.
    useEffect(() => {
        socketRef.current = new ArcanaSocket({
            onFrame: (frame) => handleFrameRef.current(frame),
            onClosed: (code, reason) => scheduleReconnectRef.current(code, reason),
        });
        return () => {
            clearReconnectTimer();
            socketRef.current?.close();
            socketRef.current = null;
        };
    }, []);

    const handleFrameRef = useRef(handleFrame);
    const scheduleReconnectRef = useRef(scheduleReconnect);
    useEffect(() => {
        handleFrameRef.current = handleFrame;
        scheduleReconnectRef.current = scheduleReconnect;
    }, [handleFrame, scheduleReconnect]);

    // iOS and Android both drop sockets in the background.
    useEffect(() => {
        const subscription = AppState.addEventListener('change', (next) => {
            if (next !== 'active') {
                return;
            }
            if (keepConnectedRef.current || stageRef.current === 'match' || stageRef.current === 'searching') {
                connect().catch(() => {});
            }
        });
        return () => subscription.remove();
    }, [connect]);

    const enter = useCallback(() => {
        keepConnectedRef.current = true;
        connect().catch(() => {});
    }, [connect]);

    const leave = useCallback(() => {
        keepConnectedRef.current = false;
        if (stageRef.current === 'lobby' || stageRef.current === 'finished') {
            clearReconnectTimer();
            socketRef.current?.close();
            setOnline(false);
        }
    }, []);

    const battle = useCallback(
        (deck: string | null) => {
            deckRef.current = deck;
            wantsQueueRef.current = true;
            setFinished(null);
            setStageBoth('searching');
            if (socketRef.current?.isOpen) {
                socketRef.current.joinQueue(hero, deck);
            } else {
                connect().catch(() => {});
            }
        },
        [connect, hero, setStageBoth],
    );

    const cancelSearch = useCallback(() => {
        wantsQueueRef.current = false;
        socketRef.current?.leaveQueue();
        setStageBoth('lobby');
    }, [setStageBoth]);

    const resume = useCallback(() => {
        const id = resumableMatch;
        if (!id) {
            return;
        }
        matchRef.current = id;
        setMatchId(id);
        setStageBoth('match');
        socketRef.current?.resume(id);
    }, [resumableMatch, setStageBoth]);

    const withMatch = useCallback((action: (socket: ArcanaSocket, id: string) => void) => {
        const socket = socketRef.current;
        const id = matchRef.current;
        if (!socket || !id) {
            return;
        }
        action(socket, id);
    }, []);

    const value = useMemo<ArcanaValue>(
        () => ({
            stage,
            online,
            catalog,
            catalogFailed,
            view,
            started,
            matchId,
            deadlineAt,
            opponentConnected,
            opponentDeck,
            finished,
            resumableMatch,
            lastError,
            emotes,
            hero,
            setHero,
            loadCatalog,
            matchHistory: (limit?: number) => fetchArcanaMatches(limit),
            enter,
            leave,
            battle,
            cancelSearch,
            resume,
            mulligan: (cardUids) => withMatch((socket, id) => socket.mulligan(id, cardUids)),
            playCard: (cardUid, target) => withMatch((socket, id) => socket.playCard(id, cardUid, target)),
            useAbility: (target) => withMatch((socket, id) => socket.ability(id, target)),
            useUltimate: (target) => withMatch((socket, id) => socket.ultimate(id, target)),
            endTurn: () => withMatch((socket, id) => socket.endTurn(id)),
            surrender: () => withMatch((socket, id) => socket.surrender(id)),
            sendEmote: (emote) => withMatch((socket, id) => socket.emote(id, emote)),
            closeResult: () => {
                setStageBoth('lobby');
                setView(null);
                setStarted(null);
                setFinished(null);
                setOpponentDeck(null);
                matchRef.current = null;
                viewSeqRef.current = 0;
                setMatchId(null);
                if (!keepConnectedRef.current) {
                    socketRef.current?.close();
                    setOnline(false);
                }
            },
        }),
        [
            battle,
            cancelSearch,
            catalog,
            catalogFailed,
            deadlineAt,
            emotes,
            enter,
            finished,
            hero,
            lastError,
            leave,
            loadCatalog,
            matchId,
            online,
            opponentConnected,
            opponentDeck,
            resumableMatch,
            resume,
            setHero,
            setStageBoth,
            stage,
            started,
            view,
            withMatch,
        ],
    );

    return <ArcanaContext.Provider value={value}>{children}</ArcanaContext.Provider>;
};

export const useArcana = (): ArcanaValue => {
    const value = useContext(ArcanaContext);
    if (!value) {
        throw new Error('useArcana must be used inside ArcanaProvider');
    }
    return value;
};
