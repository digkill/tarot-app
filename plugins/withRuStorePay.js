const {
    withProjectBuildGradle,
    withAppBuildGradle,
    withAndroidManifest,
    withMainApplication,
    withMainActivity,
    AndroidConfig,
} = require('expo/config-plugins');

const MAVEN_URL = 'https://artifactory-external.vkpartner.ru/artifactory/maven-rustore-exposed/';
const DEFAULT_SDK_VERSION = '10.3.1';

const addMavenRepository = (config) =>
    withProjectBuildGradle(config, (mod) => {
        if (!mod.modResults.contents.includes(MAVEN_URL)) {
            mod.modResults.contents = mod.modResults.contents.replace(
                /allprojects\s*\{\s*\n(\s*)repositories\s*\{/,
                (match, indent) =>
                    `${match}\n${indent}    maven { url = uri("${MAVEN_URL}") }`,
            );
        }
        return mod;
    });

const addSdkDependency = (config, sdkVersion) =>
    withAppBuildGradle(config, (mod) => {
        const dependency = `implementation("ru.rustore.sdk-wrapper.react-native:pay:${sdkVersion}")`;
        if (!mod.modResults.contents.includes('ru.rustore.sdk-wrapper.react-native')) {
            mod.modResults.contents = mod.modResults.contents.replace(
                /dependencies\s*\{/,
                (match) => `${match}\n    ${dependency}`,
            );
        }
        return mod;
    });

const addManifestMetaData = (config, {consoleApplicationId, scheme}) =>
    withAndroidManifest(config, (mod) => {
        const application = AndroidConfig.Manifest.getMainApplicationOrThrow(mod.modResults);
        const setMetaData = (name, value) => {
            AndroidConfig.Manifest.removeMetaDataItemFromMainApplication(application, name);
            AndroidConfig.Manifest.addMetaDataItemToMainApplication(application, name, value);
        };
        setMetaData('console_app_id_value', consoleApplicationId);
        setMetaData('sdk_pay_scheme_value', scheme);
        setMetaData('internal_config_key', 'react-native');
        return mod;
    });

const addPackageRegistration = (config) =>
    withMainApplication(config, (mod) => {
        let contents = mod.modResults.contents;
        if (!contents.includes('RuStoreReactPayPackage')) {
            contents = contents.replace(
                /^(package .+)$/m,
                '$1\n\nimport ru.rustore.react.pay.RuStoreReactPayPackage',
            );
            // Шаблон Expo: PackageList(this).packages.apply { ... }
            contents = contents.replace(
                /(PackageList\(this\)\.packages\.apply \{)/,
                '$1\n          add(RuStoreReactPayPackage())',
            );
            // Шаблон bare RN: val packages = PackageList(this).packages
            contents = contents.replace(
                /(val packages = PackageList\(this\)\.packages\n)/,
                '$1            packages.add(RuStoreReactPayPackage())\n',
            );
        }
        mod.modResults.contents = contents;
        return mod;
    });

const addIntentProcessing = (config) =>
    withMainActivity(config, (mod) => {
        let contents = mod.modResults.contents;
        if (!contents.includes('RuStoreReactPayModule')) {
            contents = contents.replace(
                /^(package .+)$/m,
                '$1\n\nimport android.content.Intent\nimport ru.rustore.react.pay.RuStoreReactPayModule',
            );
            contents = contents.replace(
                /(super\.onCreate\(null\))/,
                '$1\n    intent?.let { RuStoreReactPayModule.processIntent(it) }',
            );
            contents = contents.replace(
                /(\n\s*override fun invokeDefaultOnBackPressed)/,
                '\n  override fun onNewIntent(intent: Intent) {\n' +
                    '    super.onNewIntent(intent)\n' +
                    '    RuStoreReactPayModule.processIntent(intent)\n' +
                    '  }\n$1',
            );
        }
        mod.modResults.contents = contents;
        return mod;
    });

/**
 * Подключает RuStore Pay SDK (in-app платежи для сборки под RuStore).
 *
 * props:
 *   consoleApplicationId — числовой ID приложения из консоли RuStore (apps/{ID}/versions)
 *   scheme               — deeplink-схема возврата после оплаты (должна совпадать с expo.scheme)
 *   sdkVersion           — версия ru.rustore.sdk-wrapper.react-native:pay (по умолчанию 10.3.1)
 */
const withRuStorePay = (config, props = {}) => {
    const consoleApplicationId = String(props.consoleApplicationId ?? '');
    const scheme = props.scheme ?? config.scheme;
    const sdkVersion = props.sdkVersion ?? DEFAULT_SDK_VERSION;

    if (!consoleApplicationId) {
        console.warn('[withRuStorePay] consoleApplicationId не задан — платежи RuStore работать не будут');
    }
    if (!scheme || typeof scheme !== 'string') {
        throw new Error('[withRuStorePay] нужна deeplink-схема: задайте expo.scheme или props.scheme');
    }

    config = addMavenRepository(config);
    config = addSdkDependency(config, sdkVersion);
    config = addManifestMetaData(config, {consoleApplicationId, scheme});
    config = addPackageRegistration(config);
    config = addIntentProcessing(config);
    return config;
};

module.exports = withRuStorePay;
