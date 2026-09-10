# Android / RuStore релиз

Пакет `org.mediarise.tarot`. Debug из Studio/Expo ставится как `org.mediarise.tarot.debug` и не должен перекрывать прод.

## Подпись

Ключи: `/Users/digkill/Projects/Android/tarot/`  
`tarot.keystore` + `keystore.properties`. Плагин `plugins/withRuStoreReleaseSigning.js`.

RuStore принимает AAB с алиасом **`sign`** (не `upload`). Не печатай пароли в чат/логи.

После релиза в этой папке держи **только текущий** `tarot-X.Y.Z-sign.aab`. Старые AAB/APK сразу удаляй.

## Сборка AAB

Нужны JDK **17**, `ANDROID_HOME`, NDK **27.2.12479018** (Expo хочет 27.1 — пини `ndkVersion` на 27.2 в сгенерированном gradle, **не клонируй** лишние NDK в SDK).

```bash
export JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home
export PATH="$JAVA_HOME/bin:$PATH"
export ANDROID_HOME="$HOME/Library/Android/sdk"
yarn expo prebuild --platform android --clean
# при необходимости pin NDK 27.2 в android/build.gradle и app/build.gradle
cd android && ./gradlew :app:bundleRelease
```

APK на устройство: `./gradlew :app:assembleRelease -PreactNativeArchitectures=arm64-v8a`

Heap Gradle: `-Xmx4096m`. Не оставляй `android/` после копирования артефакта (~3 ГБ).

EAS-профили `rustore` (APK) / `rustore-aab` для облака **не подписывают** наш keystore — для стора только локальная подпись.

## adb

```bash
adb uninstall org.mediarise.tarot
adb uninstall org.mediarise.tarot.debug   # если ставили debug
adb install -r app-release.apk
```

`-k` у uninstall сохраняет данные. `adb devices` — телефон должен быть `device`, не `unauthorized`.
