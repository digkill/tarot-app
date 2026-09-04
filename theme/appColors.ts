export type AppColors = {
    bg: string;
    panel: string;
    accent: string;
    text: string;
    muted: string;
    gold: string;
    danger: string;
    tabBar: string;
};

export const CLASSIC_THEME: AppColors = {
    bg: '#040307',
    panel: '#1a1030',
    accent: '#6c5ce7',
    text: '#f7f4ea',
    muted: '#9a93b3',
    gold: '#d4af37',
    danger: '#ff6b6b',
    tabBar: '#0c0a14',
};

const HEX = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;

export const parseAppColors = (raw?: Partial<AppColors> | null): AppColors => {
    const pick = (value: string | undefined, fallback: string): string =>
        value && HEX.test(value.trim()) ? value.trim() : fallback;
    return {
        bg: pick(raw?.bg, CLASSIC_THEME.bg),
        panel: pick(raw?.panel, CLASSIC_THEME.panel),
        accent: pick(raw?.accent, CLASSIC_THEME.accent),
        text: pick(raw?.text, CLASSIC_THEME.text),
        muted: pick(raw?.muted, CLASSIC_THEME.muted),
        gold: pick(raw?.gold, CLASSIC_THEME.gold),
        danger: pick(raw?.danger, CLASSIC_THEME.danger),
        tabBar: pick(raw?.tabBar, CLASSIC_THEME.tabBar),
    };
};

export const RWS_SLUG = 'rws';

export const cardKeyFromImage = (image: string): string => image.replace(/\.[^.]+$/, '');

export const hexAlpha = (hex: string, alpha: number): string => {
    const raw = hex.trim().replace('#', '');
    const full =
        raw.length === 3
            ? raw
                  .split('')
                  .map((ch) => ch + ch)
                  .join('')
            : raw;
    if (full.length !== 6) {
        return hex;
    }
    const n = Number.parseInt(full, 16);
    const r = (n >> 16) & 255;
    const g = (n >> 8) & 255;
    const b = n & 255;
    return `rgba(${r},${g},${b},${alpha})`;
};
