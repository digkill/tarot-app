# Протоколы

База: `Constants.expoConfig.extra.apiBaseUrl` или `https://tarot.sorapure.fun`.  
Клиент: `features/apiClient.ts` — Bearer, refresh на 401, ошибки `{ error: { code, message } }`.

## Auth

Публично: `POST /api/v1/auth/{register,login,refresh,logout,verify-email,resend-verification,forgot-password,reset-password}`.

Сессия: `{ user, accessToken, refreshToken }` в `storage/session.ts`. TTL access ~15m, refresh ~30d.

`GET/DELETE /api/v1/me` — с auth. `user.hasPremium` синхронизируется в настройки.

## Квоты (TZ устройства, `?tz=`)

| | Free | Premium |
|---|---|---|
| Карта дня | 1 | 20 |
| AI-толкования | 0 | 50 |
| Анлок за рекламу | 5 | 0 |

`GET /usage`, `POST /usage/daily-card`, `POST /usage/ad-session`, `POST /interpretations`.  
Код `quota_exceeded` → `DailyLimitModal` / `RewardedUnlockModal`. Реклама: watch ≥15s, cooldown 90s.

## Биллинг

Продукты: `premium_monthly` 599₽, `premium_yearly` 4990₽, `premium_lifetime` 6990₽.

- Android: RuStore Pay (`features/payments.ts`, плагин `withRuStorePay`). После покупки `POST /billing/purchases`.
- iOS: только веб-шлюз ЮKassa `POST /billing/checkout` + `startYooKassaCheckout`, webhook `POST /billing/yookassa/webhook`, return `mediarisetarot://billing/complete`. RuStore на iOS не вызывается.
- Кросс-платформа: премиум привязан к аккаунту. Если `premiumSource=rustore`, на iOS покупка ЮKassa блокируется (`premium_already_active`) и UI говорит «управляется в RuStore». Если `yookassa` — на Android RuStore не открывает второй план и UI говорит «управляется в ЮKassa».
- Статус стора: `POST /billing/subscription-status` (может отозвать только свой источник).
- Админ может выдать премиум. `__DEV__` без нативного модуля — активация без оплаты.

Не использовать старый Google BillingClient.

## Магазин колод

`GET /shop/decks` (optional auth), `GET /me/decks`, медиа `/media/decks/{slug}/{file}`.

## LLM

Только сервер. Primary Kie `gpt-5-6-luna`, fallback OpenAI `gpt-4o-mini`. С клиента реальных запросов к OpenAI не делать.
