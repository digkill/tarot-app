# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Приложение для раскладов Таро: Expo SDK 57, React Native 0.86 (новая архитектура), TypeScript 6, строгая типизация.

## Команды

Пакетный менеджер — **yarn** (в git отслеживается только `yarn.lock`; не создавать `package-lock.json`).

```bash
yarn install                  # установка зависимостей
yarn start                    # dev-сервер Expo (также: yarn ios / yarn android / yarn web)
npx tsc --noEmit              # проверка типов — основная проверка, линтера и тестов в проекте нет
npx expo-doctor               # проверка здоровья проекта (должно быть 20/20)
npx expo export --platform ios   # смоук-тест сборки бандла Metro
npx expo install <pkg>        # добавление зависимостей, совместимых с SDK (не yarn add для нативных модулей)
npx expo install --fix        # выравнивание версий под SDK
```

Сборка: EAS Build, профиль `production` (Android собирается как APK, Node 20.19.4) — см. `eas.json`.

## Конфигурация

- Конфиг только динамический — `app.config.js`. **Не создавать `app.json`** (был удалён как дубликат; при наличии обоих expo-doctor падает).
- Секреты через `.env` (dotenv): `OPENAI_API_KEY`, `OPENAI_TAROT_MODEL` → попадают в `Constants.expoConfig.extra` (`openaiApiKey`, `openaiTarotModel`).
- OTA-обновления: expo-updates с политикой `runtimeVersion: appVersion`. После обновления нативных зависимостей нельзя публиковать OTA на старые бинарники — сначала поднять `version` и пересобрать.

## Архитектура

Вход: `index.ts` → `App.tsx`. Навигация — **react-navigation 7** (native-stack + bottom-tabs). **expo-router в проекте запрещён** — несовместим с react-navigation начиная с SDK 56.

Поток экранов (`App.tsx`): RootStack выбирает стартовый экран по флагам настроек — `Onboarding` (если не пройден) → `Disclaimer` (если не принят) → `Main` (табы: Home-стек со SpreadCatalog, Decks, History, Settings). Поверх табов — экраны `Reading` и `Interpretation`. Типы всех маршрутов — в `navigation/types.ts` (`RootStackParamList`, `HomeStackParamList`, `AppTabsParamList`).

Слои (зависимости направлены сверху вниз):

- `screens/`, `components/` — UI. Экраны — именованные экспорты, функциональные компоненты.
- `providers/` — состояние через React Context: `SettingsProvider` (хук `useSettings`: тема, язык, онбординг, `hasPremium`), `HistoryProvider` (хук `useHistory`: расклады, избранное, заметки).
- `storage/` — персистентность в AsyncStorage (`tarot.readings`, настройки). Провайдеры — единственные потребители этого слоя; экраны в AsyncStorage напрямую не ходят.
- `features/` — бизнес-логика: `aiInterpretation.ts` — премиум-толкования через OpenAI SDK (ключ и модель из `expo-constants` extra), `interpretation.ts` — базовые толкования из статичных данных.
- `entities/` — доменные типы (`Reading`, настройки, `LanguagePreference`).
- `data/` — статика: `spreads.ts` (описания раскладов), `tarot_{en,ru,th}.json` (значения карт).
- `utils/` — перемешивание колоды, картинки карт; `utils/gpt.ts` — мёртвый заглушечный код с фейковым URL, не использовать.

## Локализация

i18next инициализируется в `i18n.ts`; переводы UI — `i18n/{en,ru,th,zh}.json`. Язык определяется по устройству (expo-localization) и переопределяется в настройках. **Все видимые пользователю строки — только через `t('...')`**; новый ключ добавлять во все четыре файла. Внимание: у данных карт (`data/`) языков три (en/ru/th), у UI — четыре (+zh).

## Премиум-функция

AI-толкования — платная функция (описание: `PREMIUM_FEATURE.md`). Флаг `hasPremium` в настройках, модалка `components/PremiumModal.tsx`, генерация в `features/aiInterpretation.ts`. Реальные запросы к OpenAI из тестов и проверок не делать.

## Платежи RuStore (Android)

Подписка оплачивается через RuStore Pay SDK (нативная зависимость `ru.rustore.sdk-wrapper.react-native:pay`, НЕ npm-пакет; старый BillingClient SDK мёртв с августа 2026 — не использовать). Слои:

- `plugins/withRuStorePay.js` — config-плагин: maven-репозиторий VK, gradle-зависимость, meta-data в манифесте (`console_app_id_value`, `sdk_pay_scheme_value`), регистрация `RuStoreReactPayPackage` и `processIntent` в MainActivity. Проверять изменения плагина через `npx expo prebuild --platform android --no-install --clean` (папку `android/` после проверки удалять — она в .gitignore, проект на CNG).
- `libs/RuStoreReactPay/` — официальная JS-обёртка из примера RuStore (vendored, не править без причины).
- `features/payments.ts` — сервис поверх обёртки: `purchasePremium`, `hasActivePremiumSubscription`, `isPaymentsSupported`. Нативный модуль есть только в Android-сборке; в Expo Go/iOS методы кидают `PaymentsUnavailableError`, а PremiumModal в `__DEV__` активирует премиум без оплаты.
- Конфиг: `RUSTORE_CONSOLE_APP_ID` и `RUSTORE_PREMIUM_PRODUCT_ID` в `.env` → `extra`; deeplink-схема `mediarisetarot` (должна совпадать в `scheme` и props плагина). Сборка для RuStore: `eas build -p android --profile rustore`.
- Подписки поддерживают только одностадийную оплату (`ONE_STEP`); тестировать покупки можно только на устройстве с установленным RuStore и «песочными» платежами из консоли разработчика.

## Стиль кода

Отступ 4 пробела; стили — `StyleSheet.create` внизу файла; тёмная палитра: акцент `#6c5ce7`, светлый текст `#f7f4ea`, ошибки `#ff6b6b`. Реф для скриншотов — тип `ViewShotRef` из react-native-view-shot 5.

## Субагенты

В `.claude/agents/` определены проектные агенты: `code-reviewer`, `backend-developer`, `mobile-developer`, `qa-tester` — делегировать им задачи по их профилю.
