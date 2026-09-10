# Архитектура

Две части в одном репо: **Expo-клиент** (корень) и **Go-бэкенд** (`backend/`).

## Клиент

Вход: `index.ts` → `App.tsx`. Expo SDK 57, RN 0.86 (new arch), TypeScript 6, React 19.  
Навигация: **react-navigation 7**. **expo-router запрещён.**

Поток: `VideoSplash` → `Disclaimer` → `Auth`/`VerifyEmail`/`ForgotPassword` → табы `Main` → модально `Reading` / `Interpretation` / `LegalDocument`. Онбординга нет.

Провайдеры снаружи внутрь: `SettingsProvider` → `AuthProvider` → `DeckShopProvider` → `HistoryProvider`.

| Слой | Путь | Правило |
|---|---|---|
| UI | `screens/`, `components/` | именованные FC, стили внизу |
| Состояние | `providers/` | экраны не ходят в AsyncStorage |
| Сеть/домен | `features/` | `apiClient.ts` — единственный HTTP |
| Типы | `entities/` | `Reading`, `Settings`, `AuthUser` |
| Локальный стор | `storage/` | `tarot.readings`, настройки, сессия |
| Статика | `data/` | расклады + `tarot_{en,ru,th}.json` |
| i18n UI | `i18n/{en,ru,th,zh}.json` | + `legal_*` |

История раскладов **локальная**. CRUD `/api/v1/readings` на сервере есть — клиент его не использует.

Расклады: бесплатные `one-card`, `three-card`, `celtic-cross`; премиум love / horseshoe / weekly / year-wheel.

Колоды: встроенная `rws` + магазин `/shop/decks`. Тема UI из колоды (`theme/appColors.ts`). Фон-паттерн общий: `components/AppBackground.tsx` на всём дереве навигации.

## Бэкенд

`backend/cmd/server/main.go`: chi, pgx, goose, JWT. Postgres + SMTP (Beget).  
LLM: Kie.ai primary, OpenAI fallback. Ключи только на сервере.  
Админка: `https://tarot.sorapure.fun/admin`. PII (IP/UA) шифруется at rest.

## Мёртвое

- `utils/gpt.ts` — заглушка, не использовать
- OpenAI SDK / ключ в `extra` клиента — убраны; толкования через `POST /api/v1/interpretations`
