# CLAUDE.md

Приложение раскладов Таро: **клиент Expo SDK 57 / RN 0.86** + **бэкенд Go** в `backend/`.

Актуальная память агентов: **[`.cursor/memory/README.md`](.cursor/memory/README.md)**  
(архитектура, протоколы API, конвенции, Android/RuStore). Не опирайся на `PREMIUM_FEATURE.md`.

## Команды

Пакетный менеджер — **yarn** (не создавать `package-lock.json`).

```bash
yarn install
yarn start                          # также: yarn ios / yarn android / yarn web
npx tsc --noEmit
npx expo-doctor                     # цель 20/20
npx expo export --platform ios
npx expo install <pkg>              # нативные модули, не yarn add
npx expo install --fix
```

Релиз Android/RuStore: `.cursor/memory/android-release.md`. EAS-профили в `eas.json` (`rustore`, `rustore-aab`) для облака **не** используют наш keystore.

## Конфигурация

- Только динамический `app.config.js`. **Не создавать `app.json`.**
- Секреты в `.env` (не коммитить). Клиентский `extra.apiBaseUrl` → `https://tarot.sorapure.fun`. Ключи LLM только на сервере.
- OTA: `runtimeVersion: appVersion`. После нативных изменений поднять `version` и пересобрать бинарь.
- CNG: папки `android/` `ios/` gitignored. После проверки prebuild удалять.

## Архитектура (кратко)

`index.ts` → `App.tsx`. **react-navigation 7**. **expo-router запрещён.**

Disclaimer → Auth → Main (табы Explore/Decks/History/Settings) → Reading / Interpretation.

Слои сверху вниз: `screens/` `components/` → `providers/` → `features/` `storage/` `entities/` `data/`.  
Экраны в AsyncStorage не ходят. HTTP — `features/apiClient.ts`.

Локализация UI: `i18n/{en,ru,th,zh}.json`, все строки через `t('...')`. Карты: en/ru/th.

## Стиль

Отступ 4 пробела; `StyleSheet.create` внизу; акцент `#6c5ce7`, текст `#f7f4ea`, ошибки `#ff6b6b`.

## Субагенты

`.claude/agents/`: `code-reviewer`, `backend-developer`, `mobile-developer`, `qa-tester`.
