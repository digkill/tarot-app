const fs = require('node:fs');
const path = require('node:path');
const {withDangerousMod} = require('expo/config-plugins');

function configureReleaseScheme(platformProjectRoot) {
    const schemePath = path.join(
        platformProjectRoot,
        'Tarot.xcodeproj/xcshareddata/xcschemes/Tarot.xcscheme',
    );
    const contents = fs.readFileSync(schemePath, 'utf8');
    const launchAction = /<LaunchAction\b[^>]*>/;
    const configuration = /\bbuildConfiguration\s*=\s*"[^"]*"/;
    const launchTag = contents.match(launchAction)?.[0];
    if (!launchTag || !configuration.test(launchTag)) {
        throw new Error(`Missing Run build configuration in ${schemePath}`);
    }
    const updated = contents.replace(launchAction, (tag) =>
        tag.replace(configuration, 'buildConfiguration = "Release"'),
    );
    if (updated !== contents) {
        fs.writeFileSync(schemePath, updated);
    }
}

module.exports = (config) => withDangerousMod(config, ['ios', async (mod) => {
    // Xcode Run must embed production JS instead of launching the Metro client.
    configureReleaseScheme(mod.modRequest.platformProjectRoot);
    return mod;
}]);

// Apply to an existing generated project without running a full prebuild.
if (require.main === module) {
    if (!process.argv[2]) {
        throw new Error('Usage: node plugins/withIosReleaseScheme.js <ios-directory>');
    }
    configureReleaseScheme(path.resolve(process.argv[2]));
}
