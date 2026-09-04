import * as Localization from 'expo-localization';
import type {LanguagePreference} from '../entities';

export const SUPPORTED_LANGUAGES: LanguagePreference[] = ['en', 'ru', 'th', 'zh'];

export const isLanguagePreference = (value: unknown): value is LanguagePreference =>
    typeof value === 'string' && SUPPORTED_LANGUAGES.includes(value as LanguagePreference);

const fromLocaleTag = (tag: string | null | undefined): LanguagePreference | null => {
    if (!tag) {
        return null;
    }
    const primary = tag.toLowerCase().replace('_', '-').split('-')[0];
    if (primary === 'zh') {
        return 'zh';
    }
    return isLanguagePreference(primary) ? primary : null;
};

export const resolveDeviceLanguage = (): LanguagePreference => {
    const locales = Localization.getLocales();
    for (const locale of locales) {
        const fromCode = fromLocaleTag(locale.languageCode);
        if (fromCode) {
            return fromCode;
        }
        const fromTag = fromLocaleTag(locale.languageTag);
        if (fromTag) {
            return fromTag;
        }
    }

    try {
        const fromIntl = fromLocaleTag(Intl.DateTimeFormat().resolvedOptions().locale);
        if (fromIntl) {
            return fromIntl;
        }
    } catch {
        // Intl may be unavailable on some runtimes
    }

    return 'en';
};
