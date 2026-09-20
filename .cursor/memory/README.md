# Память проекта Tarot

Читай эти файлы в начале задачи. Код — источник истины; если док расходится с репозиторием, правь док в том же PR.

| Файл | Когда |
|---|---|
| [architecture.md](architecture.md) | экраны, слои, навигация, провайдеры |
| [protocols.md](protocols.md) | HTTP API, auth, биллинг, квоты, LLM |
| [conventions.md](conventions.md) | стиль, i18n, yarn, CNG, проверки |
| [android-release.md](android-release.md) | RuStore AAB/APK, ключи, adb, NDK |

Не использовать как актуальное: `PREMIUM_FEATURE.md` (демо/клиентский OpenAI).  
`CLAUDE.md` и `AGENTS.md` — короткие указатели сюда.

**Сейчас:** оба клиента `1.3.0`, `versionCode`/`buildNumber`/`CURRENT_PROJECT_VERSION` `14`, пакет `org.mediarise.tarot`, API `https://tarot.sorapure.fun`.
Версии Android (`app.config.js`) и нативного iOS (`ios-swift/project.yml`) держим одинаковыми.
