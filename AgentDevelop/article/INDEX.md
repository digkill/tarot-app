---
title: Как делали Tarot
---

# Статья: Tarot

Сначала толкования жили в клиенте (`PREMIUM_FEATURE.md`, ключ OpenAI). Это сломалось и по безопасности, и по квотам. Сейчас клиент — UI и локальная история, мозг — Go API.

## Биллинг как развилка сторов

Android ушёл в **RuStore Pay**, не Google BillingClient. iOS/web — ЮKassa checkout + webhook. Админ может выдать премиум вручную.

## Пример: клиентский HTTP

```ts
// features/apiClient.ts — единственная точка
// Bearer, refresh на 401, ошибки { error: { code, message } }
```

Код `quota_exceeded` открывает модалки лимита или рекламы (≥15 с, cooldown 90 с).

## Колоды

Тема приложения едет из выбранной колоды (`theme/appColors.ts`). Медиа `/media/decks/{slug}/{file}`.
