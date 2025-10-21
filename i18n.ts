import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import * as Localization from 'expo-localization';

// Импортируем файлы переводов
import en from './i18n/en.json';
import ru from './i18n/ru.json';
import th from './i18n/th.json';
import zh from './i18n/zh.json';

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
    zh: {
        translation: zh,
    },
};

// Детектор языка
// Инициализация i18next
const locales = Localization.getLocales();
const deviceLanguageCode = locales[0]?.languageCode?.toLowerCase();
const availableLanguages = Object.keys(resources);
const initialLanguage = deviceLanguageCode && availableLanguages.includes(deviceLanguageCode) ? deviceLanguageCode : 'en';

i18n
    .use(initReactI18next)
    .init({
        resources,
        fallbackLng: 'en',
        supportedLngs: availableLanguages,
        lng: initialLanguage,
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
