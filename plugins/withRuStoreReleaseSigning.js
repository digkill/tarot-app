const {withAppBuildGradle} = require('expo/config-plugins');

const MARKER = 'TAROT_RUSTORE_RELEASE_SIGNING';
const PROPERTIES_PATH = '/Users/digkill/Projects/Android/tarot/keystore.properties';

const LOAD_PROPS = `
def tarotKeystorePropertiesFile = file("${PROPERTIES_PATH}")
def tarotKeystoreProperties = new Properties()
if (tarotKeystorePropertiesFile.exists()) {
    tarotKeystoreProperties.load(new FileInputStream(tarotKeystorePropertiesFile))
}

`;

const RELEASE_SIGNING = `
        // ${MARKER}
        release {
            if (tarotKeystorePropertiesFile.exists()) {
                keyAlias tarotKeystoreProperties['keyAlias']
                keyPassword tarotKeystoreProperties['keyPassword']
                storeFile file(tarotKeystoreProperties['storeFile'])
                storePassword tarotKeystoreProperties['storePassword']
            }
        }`;

const withRuStoreReleaseSigning = (config) =>
    withAppBuildGradle(config, (mod) => {
        let {contents} = mod.modResults;
        if (contents.includes(MARKER)) {
            return mod;
        }

        if (!contents.includes('signingConfigs {')) {
            throw new Error('withRuStoreReleaseSigning: signingConfigs block not found');
        }

        if (!contents.includes('def tarotKeystorePropertiesFile')) {
            contents = contents.replace(/\nandroid \{\n/, `\n${LOAD_PROPS}android {\n`);
        }

        contents = contents.replace(
            /signingConfigs \{\s*\n(\s*)debug \{[\s\S]*?keyPassword 'android'\s*\n\s*\}/,
            (match) => `${match}${RELEASE_SIGNING}`,
        );

        contents = contents.replace(
            /(release \{\s*\n(?:\s*\/\/[^\n]*\n)*\s*)signingConfig signingConfigs\.debug/,
            `$1signingConfig tarotKeystorePropertiesFile.exists() ? signingConfigs.release : signingConfigs.debug`,
        );

        if (!contents.includes(MARKER)) {
            throw new Error('withRuStoreReleaseSigning: failed to inject release signingConfig');
        }

        mod.modResults.contents = contents;
        return mod;
    });

module.exports = withRuStoreReleaseSigning;
