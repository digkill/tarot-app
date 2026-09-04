import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useState,
} from 'react';
import type {ReactNode} from 'react';
import type {ImageSourcePropType} from 'react-native';
import {useTranslation} from 'react-i18next';
import {useSettings} from './SettingsProvider';
import {useAuth} from './AuthProvider';
import {cardImages} from '../utils/cardImages';
import {CLASSIC_THEME, RWS_SLUG, cardKeyFromImage, parseAppColors, type AppColors} from '../theme/appColors';
import {
    absMediaUrl,
    fetchMyDeckSlugs,
    fetchShopDecks,
    localizeShopText,
    type ShopDeck,
} from '../features/shopApi';
import {reportPurchaseRequest} from '../features/billingApi';
import {
    PurchaseCancelledError,
    RuStoreMissingError,
    isPaymentsSupported,
    purchaseStoreProduct,
} from '../features/payments';

const CARD_BACK = require('../assets/cards/card_back.jpeg');

const builtinRws = (t: (k: string) => string): ShopDeck => ({
    slug: RWS_SLUG,
    titles: {
        en: t('deck.classicName'),
        ru: t('deck.classicName'),
        th: t('deck.classicName'),
        zh: t('deck.classicName'),
    },
    descriptions: {
        en: t('deck.classicHint'),
        ru: t('deck.classicHint'),
        th: t('deck.classicHint'),
        zh: t('deck.classicHint'),
    },
    theme: CLASSIC_THEME,
    priceKop: 0,
    productId: '',
    isFree: true,
    owned: true,
    cardCount: 78,
    hasBack: true,
    coverUrl: '',
    backUrl: '',
    cards: {},
    bundled: true,
});

type DeckShopValue = {
    colors: AppColors;
    decks: ShopDeck[];
    selectedSlug: string;
    selected: ShopDeck;
    loading: boolean;
    isOwned: (slug: string) => boolean;
    selectDeck: (slug: string) => Promise<void>;
    purchaseDeck: (slug: string) => Promise<void>;
    faceSource: (imageFile: string, deckId?: string) => ImageSourcePropType | undefined;
    backSource: (deckId?: string) => ImageSourcePropType;
    titleOf: (deck: ShopDeck) => string;
};

const DeckShopContext = createContext<DeckShopValue | null>(null);

const uniq = (items: string[]): string[] => [...new Set(items.filter(Boolean))];

export const DeckShopProvider = ({children}: {children: ReactNode}) => {
    const {t, i18n} = useTranslation();
    const {settings, updateSettings, setSetting} = useSettings();
    const {user} = useAuth();
    const [remote, setRemote] = useState<ShopDeck[]>([]);
    const [serverSlugs, setServerSlugs] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        let cancelled = false;
        fetchShopDecks()
            .then((res) => {
                if (!cancelled) {
                    setRemote(res.decks ?? []);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setRemote([]);
                }
            })
            .finally(() => {
                if (!cancelled) {
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [user]);

    useEffect(() => {
        if (!user) {
            setServerSlugs([]);
            return;
        }
        fetchMyDeckSlugs()
            .then((res) => setServerSlugs(res.slugs ?? []))
            .catch(() => setServerSlugs([]));
    }, [user]);

    const ownedSlugs = useMemo(() => {
        const local = settings.ownedDeckIds ?? [];
        return new Set([RWS_SLUG, ...local, ...serverSlugs]);
    }, [serverSlugs, settings.ownedDeckIds]);

    const decks = useMemo(() => {
        const rws = builtinRws(t);
        const extras = remote
            .filter((deck) => deck.slug !== RWS_SLUG)
            .map((deck) => ({
                ...deck,
                theme: parseAppColors(deck.theme),
                owned: deck.isFree || deck.owned || ownedSlugs.has(deck.slug),
                coverUrl: absMediaUrl(deck.coverUrl),
                backUrl: absMediaUrl(deck.backUrl),
                cards: Object.fromEntries(
                    Object.entries(deck.cards ?? {}).map(([key, url]) => [key, absMediaUrl(url)]),
                ),
            }));
        return [rws, ...extras];
    }, [ownedSlugs, remote, t]);

    const selectedSlug = useMemo(() => {
        const wanted = settings.selectedDeckId || RWS_SLUG;
        const found = decks.find((d) => d.slug === wanted);
        if (found && (found.owned || found.isFree || found.bundled)) {
            return found.slug;
        }
        return RWS_SLUG;
    }, [decks, settings.selectedDeckId]);

    const selected = decks.find((d) => d.slug === selectedSlug) ?? decks[0];

    const colors = selected?.theme ?? CLASSIC_THEME;

    const isOwned = useCallback(
        (slug: string) => {
            if (slug === RWS_SLUG) {
                return true;
            }
            const deck = decks.find((item) => item.slug === slug);
            return Boolean(deck?.isFree || deck?.owned || ownedSlugs.has(slug));
        },
        [decks, ownedSlugs],
    );

    const selectDeck = useCallback(
        async (slug: string) => {
            const deck = decks.find((d) => d.slug === slug);
            if (!deck || (!deck.owned && !deck.isFree && !deck.bundled)) {
                return;
            }
            await setSetting('selectedDeckId', slug);
        },
        [decks, setSetting],
    );

    const markOwned = useCallback(
        async (slug: string) => {
            const next = uniq([...(settings.ownedDeckIds ?? []), slug]);
            await updateSettings({ownedDeckIds: next, selectedDeckId: slug});
            setServerSlugs((prev) => uniq([...prev, slug]));
        },
        [settings.ownedDeckIds, updateSettings],
    );

    const purchaseDeck = useCallback(
        async (slug: string) => {
            const deck = decks.find((d) => d.slug === slug);
            if (!deck || deck.bundled || deck.isFree) {
                await selectDeck(slug);
                return;
            }
            if (isOwned(slug)) {
                await selectDeck(slug);
                return;
            }

            const reportAndUnlock = async (extra?: {
                invoiceId?: string;
                purchaseId?: string;
                orderId?: string;
                sandbox?: boolean;
                source?: 'rustore' | 'dev';
            }) => {
                if (user && deck.productId) {
                    try {
                        await reportPurchaseRequest({
                            productId: deck.productId,
                            source: extra?.source ?? 'rustore',
                            invoiceId: extra?.invoiceId,
                            purchaseId: extra?.purchaseId,
                            orderId: extra?.orderId,
                            sandbox: extra?.sandbox,
                        });
                    } catch (error) {
                        console.warn('[shop] report deck purchase failed', error);
                    }
                }
                await markOwned(slug);
            };

            if (!isPaymentsSupported() || !deck.productId) {
                if (__DEV__) {
                    await reportAndUnlock({source: 'dev'});
                    return;
                }
                throw new Error('payments_unavailable');
            }

            const result = await purchaseStoreProduct(deck.productId, 'DARK');
            await reportAndUnlock({
                invoiceId: result.invoiceId,
                purchaseId: result.purchaseId,
                orderId: result.orderId,
                sandbox: result.sandbox,
                source: 'rustore',
            });
        },
        [decks, isOwned, markOwned, selectDeck, user],
    );

    const faceSource = useCallback(
        (imageFile: string, deckId?: string): ImageSourcePropType | undefined => {
            const slug = deckId || selectedSlug;
            const deck = decks.find((d) => d.slug === slug);
            const key = cardKeyFromImage(imageFile);
            const remoteUrl = deck?.cards[key];
            if (remoteUrl && !deck?.bundled) {
                return {uri: remoteUrl};
            }
            return cardImages[imageFile];
        },
        [decks, selectedSlug],
    );

    const backSource = useCallback(
        (deckId?: string): ImageSourcePropType => {
            const slug = deckId || selectedSlug;
            const deck = decks.find((d) => d.slug === slug);
            if (deck?.backUrl && !deck.bundled) {
                return {uri: deck.backUrl};
            }
            return CARD_BACK;
        },
        [decks, selectedSlug],
    );

    const titleOf = useCallback(
        (deck: ShopDeck) => localizeShopText(deck.titles, i18n.language, deck.slug),
        [i18n.language],
    );

    const value = useMemo(
        () => ({
            colors,
            decks,
            selectedSlug,
            selected,
            loading,
            isOwned,
            selectDeck,
            purchaseDeck,
            faceSource,
            backSource,
            titleOf,
        }),
        [
            backSource,
            colors,
            decks,
            faceSource,
            isOwned,
            loading,
            purchaseDeck,
            selectDeck,
            selected,
            selectedSlug,
            titleOf,
        ],
    );

    return <DeckShopContext.Provider value={value}>{children}</DeckShopContext.Provider>;
};

export const useDeckShop = (): DeckShopValue => {
    const ctx = useContext(DeckShopContext);
    if (!ctx) {
        throw new Error('useDeckShop must be used within DeckShopProvider');
    }
    return ctx;
};

export const useAppColors = (): AppColors => useDeckShop().colors;

export {PurchaseCancelledError, RuStoreMissingError};
