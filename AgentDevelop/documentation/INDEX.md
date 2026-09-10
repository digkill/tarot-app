---
title: Документация Tarot
---

# Документация

Память: [`.cursor/memory/README.md`](../../.cursor/memory/README.md). Навигация — **react-navigation 7**, не expo-router. Конфиг только `app.config.js`. Пакетный менеджер **yarn**.

## Клиент

`index.ts` → `App.tsx`. Поток: VideoSplash → Disclaimer → Auth → табы Main; модалки Reading / Interpretation / Legal. Провайдеры: Settings → Auth → DeckShop → History. HTTP только `features/apiClient.ts`. История раскладов **локальная**.

## API

`backend/cmd/server/main.go`: chi, pgx, goose, JWT. Квоты с `?tz=`. Биллинг: RuStore Pay на Android, ЮKassa на iOS/web. LLM на сервере.

## Релиз Android

RuStore AAB, `versionCode`/`buildNumber` 10, пакет `org.mediarise.tarot`. `/android` в gitignore (CNG).
