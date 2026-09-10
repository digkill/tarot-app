# Конвенции

- Пакетный менеджер: **yarn**. Не создавать `package-lock.json`. Нативные модули: `npx expo install`.
- Конфиг только `app.config.js`. **Не создавать `app.json`.**
- CNG: `/android` и `/ios` в `.gitignore`. После проверки prebuild удаляй `android/` (тяжёлая). Не копируй секреты в git.
- iOS: рабочий проект — `ios/Tarot.xcworkspace`, генерируется Expo prebuild. По указанию пользователя `ios-native/` исключён из работы: не читать, не синхронизировать и не использовать для сборки. Metro не запускать для Release.
- Xcode Run в схеме `Tarot` использует Release: `withIosReleaseScheme` закрепляет это после prebuild. Test остаётся Debug. Для дистрибуции — Product → Archive (Release) в рабочем workspace.
- Отступ 4 пробела. Экраны — именованный export FC. Стили — `StyleSheet.create` внизу.
- Палитра: фон тёмный, акцент `#6c5ce7`, текст `#f7f4ea`, ошибка `#ff6b6b`, золото `#d4af37`.
- Все пользовательские строки — `t('...')`. Новый ключ — во все `i18n/{en,ru,th,zh}.json`. Карты: en/ru/th (zh UI падает на en-колоду).
- Экраны не читают AsyncStorage: только провайдеры → `storage/`.
- HTTP только через `apiClient.ts`. Не хардкодить ключи. `.env` не коммитить.
- Проверки: `npx tsc --noEmit`. Смоук: `npx expo export --platform ios`. Doctor: `npx expo-doctor` (цель 20/20).
- OTA: `runtimeVersion: appVersion`. После нативных изменений — поднять `version` и пересобрать бинарь, не слать OTA на старый runtime.
- Xcode 27 / iOS 27 SDK требует UIScene. Не удалять `withXcode27SceneLifecycle`: он генерирует scene manifest, переносит запуск RN в `SceneDelegate` и сохраняет deep links.
- Субагенты: `.claude/agents/` (`mobile-developer`, `backend-developer`, `code-reviewer`, `qa-tester`).
