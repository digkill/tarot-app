const {withProjectBuildGradle} = require('expo/config-plugins');

const MARKER = 'TAROT_NDK_VERSION_PIN';
const NDK_VERSION = '27.2.12479018';

/**
 * expo-root-project pins ndkVersion to whatever Expo bundles by default
 * (currently 27.1.12297006), but that copy is broken/incomplete in this
 * machine's SDK (missing source.properties) — see .cursor/memory/android-release.md.
 * Override rootProject.ext.ndkVersion right after expo-root-project sets it
 * and before com.facebook.react.rootproject reads it, since that second
 * plugin configures :app synchronously during `apply` — setting the override
 * after both `apply` lines is too late.
 */
const withAndroidNdkVersion = (config) =>
    withProjectBuildGradle(config, (mod) => {
        if (mod.modResults.contents.includes(MARKER)) {
            return mod;
        }
        const anchor = 'apply plugin: "expo-root-project"';
        if (!mod.modResults.contents.includes(anchor)) {
            throw new Error('withAndroidNdkVersion: expo-root-project anchor not found in android/build.gradle');
        }
        mod.modResults.contents = mod.modResults.contents.replace(
            anchor,
            `${anchor}\n\n// ${MARKER}\next.ndkVersion = "${NDK_VERSION}"`,
        );
        return mod;
    });

module.exports = withAndroidNdkVersion;
