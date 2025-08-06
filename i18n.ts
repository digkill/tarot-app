import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import * as Localization from 'expo-localization';

// Импортируем файлы переводов
import en from './i18n/en.json';
import ru from './i18n/ru.json';
import th from './i18n/th.json';

// Определяем ресурсы переводов
const resources = {
    en: {
        translation: en,
    },
    ru: {
        translation: ru,
    },
    th: {
        translation: th,
    },
};

// Детектор языка
const languageDetector = {
    type: 'languageDetector',
    async: true,
    detect: (callback: (lng: string | null) => void) => {
        const locales = Localization.getLocales();
        const preferredLanguage = locales.length > 0 ? locales[0].languageCode : 'en';
        console.log('Detected locale:', preferredLanguage);
        callback(preferredLanguage);
    },
    init: () => {},
    cacheUserLanguage: () => {},
};

// Инициализация i18next
i18n
    .use(languageDetector)
    .use(initReactI18next)
    .init({
        resources,
        fallbackLng: 'en',
        debug: process.env.NODE_ENV === 'development', // Включаем debug только в разработке
        interpolation: {
            escapeValue: false, // React защищает от XSS
        },
        react: {
            useSuspense: false, // Отключаем Suspense для упрощения
        },
    })
    .catch((error) => {
        console.error('i18next initialization failed:', error);
    });

export const I18n = i18n;