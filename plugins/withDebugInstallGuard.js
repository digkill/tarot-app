const fs = require('fs');
const path = require('path');
const {withAppBuildGradle, withDangerousMod} = require('expo/config-plugins');

const MARKER = 'TAROT_DEBUG_INSTALL_GUARD';

const DEBUG_STRINGS = `<?xml version="1.0" encoding="utf-8"?>
<resources>
  <string name="app_name">Tarot Debug</string>
</resources>
`;

/**
 * Android Studio Run always installs the debug variant. Keep it from replacing
 * the production package (org.mediarise.tarot) used for RuStore / signed APK.
 */
const withDebugInstallGuard = (config) => {
    config = withAppBuildGradle(config, (mod) => {
        if (mod.modResults.contents.includes(MARKER)) {
            return mod;
        }
        mod.modResults.contents = mod.modResults.contents.replace(
            /debug\s*\{\s*\n(\s*)signingConfig signingConfigs\.debug\s*\n(\s*)debuggable true/,
            `debug {\n$1// ${MARKER}\n$1isDefault false\n$1signingConfig signingConfigs.debug\n$2debuggable true\n$2applicationIdSuffix ".debug"\n$2versionNameSuffix "-debug"`,
        );
        if (!mod.modResults.contents.includes('isDefault true')) {
            mod.modResults.contents = mod.modResults.contents.replace(
                /release\s*\{\s*\n(\s*)signingConfig/,
                'release {\n$1isDefault true\n$1signingConfig',
            );
        }
        return mod;
    });

    return withDangerousMod(config, [
        'android',
        async (mod) => {
            const dir = path.join(
                mod.modRequest.platformProjectRoot,
                'app/src/debug/res/values',
            );
            fs.mkdirSync(dir, {recursive: true});
            fs.writeFileSync(path.join(dir, 'strings.xml'), DEBUG_STRINGS);
            return mod;
        },
    ]);
};

module.exports = withDebugInstallGuard;
