import i18n from 'i18next';
import {initReactI18next} from 'react-i18next';

// Импортируем файлы переводов
import en from './i18n/en.json';
import ru from './i18n/ru.json';
import th from './i18n/th.json';
import zh from './i18n/zh.json';
import enLegal from './i18n/legal_en.json';
import ruLegal from './i18n/legal_ru.json';
import thLegal from './i18n/legal_th.json';
import zhLegal from './i18n/legal_zh.json';
import {resolveDeviceLanguage, SUPPORTED_LANGUAGES} from './utils/locale';

// Определяем ресурсы переводов
const resources = {
    en: {
        translation: {...en, ...enLegal},
    },
    ru: {
        translation: {...ru, ...ruLegal},
    },
    th: {
        translation: {...th, ...thLegal},
    },
    zh: {
        translation: {...zh, ...zhLegal},
    },
};

i18n
    .use(initReactI18next)
    .init({
        resources,
        fallbackLng: 'en',
        supportedLngs: [...SUPPORTED_LANGUAGES],
        nonExplicitSupportedLngs: true,
        lng: resolveDeviceLanguage(),
        debug: false,
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
