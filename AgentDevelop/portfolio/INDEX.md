---
tagline: Мобильное таро с AI-толкованиями, магазином колод и премиум-квотами.
audience: Пользователи, которым нужен ритуал расклада + связный текст, а не голый генератор карт.
purpose: Приложение раскладов с серверным AI, биллингом и лимитами — без ключей модели в клиенте.
problems_solved: Клиентский OpenAI в демо утекал ключами и квотами. Нет нормального магазина колод и дневных лимитов. Нужны RuStore и ЮKassa, не только Google Play Billing.
functions: Расклады (1 / 3 / Кельтский крест + премиум), карта дня, AI-толкование, магазин колод, RuStore Pay / ЮKassa, i18n ru/en/th/zh.
achievements: Клиент 1.1.8 (build 10), пакет org.mediarise.tarot, API tarot.sorapure.fun, админка, PII шифруется at rest.
---

# Tarot — портфолио

Клиент на **Expo / React Native**, API на **Go** (`backend/`). Прод: [tarot.sorapure.fun](https://tarot.sorapure.fun).

## Для бизнеса

Подписка и разовые продукты (`premium_monthly` 599₽, `yearly` 4990₽, `lifetime` 6990₽) плюс магазин колод. Бесплатный слой держит привычку (карта дня, базовые расклады), премиум открывает AI и расширенные схемы. Реклама — анлок интерпретации для free.

Юридически важно: толкования — развлечение, дисклеймер на входе. LLM только на сервере (Kie primary, OpenAI fallback).

## Что умеет

- Расклады: one-card, three-card, celtic-cross; премиум love / horseshoe / weekly / year-wheel.
- Колоды: встроенная RWS + `/shop/decks`; тема UI из колоды.
- Квоты по TZ устройства: карта дня 1/20, AI 0/50, рекламные анлоки 5 у free.
- Android: RuStore AAB, локальная подпись `sign`. iOS/web: ЮKassa.

## Скрины

Нужны: сплэш/дисклеймер, расклад, толкование, магазин колод, экран премиум. [`screenshots/`](screenshots/).
