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

Продукты: `premium_monthly` 599₽ / $7.99, `premium_yearly` 4990₽ / $59.99, `premium_lifetime` 6990₽ / $79.99 (USD — только CloudPayments, `AmountUSDCents` в `billing/catalog.go`).

- Android: RuStore Pay (`features/payments.ts`, плагин `withRuStorePay`). После покупки `POST /billing/purchases`.
- iOS: веб-шлюзы через `POST /billing/checkout {productId, provider}` + `startWebCheckout` (`features/webCheckout.ts`), return `mediarisetarot://billing/complete`. На пейволе две кнопки: «Картой российского банка» → `provider: "yookassa"` (RUB), «Иностранной картой» → `provider: "cloudpayments"` (USD). RuStore на iOS не вызывается.
  - ЮKassa: webhook `POST /billing/yookassa/webhook`. `IS_PROD=false` берёт `YOOKASSA_*_TEST`.
  - CloudPayments: счёт `orders/create` (hosted page, `InvoiceId` = id транзакции), уведомление Pay `POST /billing/cloudpayments/pay` (HMAC `X-Content-HMAC`/`Content-HMAC`, ответ `{"code":0}`), при опросе `GET /billing/checkout/{id}` сверка через `payments/find`. Премиум выдаётся один раз — при переходе транзакции `pending → paid` (`MarkPaidFromPending`), с проверкой суммы/валюты. `IS_PROD=false` берёт `CLOUDPAYMENTS_*_TEST`. В `transactions.amount_kop` для USD лежат центы, `currency='USD'`; выручка в админке разделена по валютам.
  - `premiumSource=cloudpayments` нельзя передать через `POST /billing/purchases` — только подписанный webhook.
- Закрытие Safari завершает текущую попытку без polling и сообщения «оплата ожидается»: кнопка сразу разблокируется, следующее нажатие создаёт новый checkout. Проверка статуса с ожиданием выполняется только после callback браузера; закрытие окна само по себе не отменяет платёж у ЮKassa.
- Кросс-платформа: премиум привязан к аккаунту. Если `premiumSource=rustore`, на iOS покупка ЮKassa блокируется (`premium_already_active`) и UI говорит «управляется в RuStore». Если `yookassa` или `cloudpayments` — на Android RuStore не открывает второй план и UI говорит, где оплачено.
- Статус стора: `POST /billing/subscription-status` (может отозвать только свой источник).
- Админ может выдать премиум. `__DEV__` без нативного модуля — активация без оплаты.

Не использовать старый Google BillingClient.

## Магазин колод

`GET /shop/decks` (optional auth), `GET /me/decks`, медиа `/media/decks/{slug}/{file}`.

## LLM

Только сервер. Primary Kie `gpt-5-6-luna`, fallback OpenAI `gpt-4o-mini`. С клиента реальных запросов к OpenAI не делать.
