const fs = require('fs');
const path = require('path');
const {withGradleProperties, withDangerousMod} = require('expo/config-plugins');

const CANDIDATES = [
    process.env.JAVA_HOME,
    '/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home',
    '/Library/Java/JavaVirtualMachines/temurin-17.jdk/Contents/Home',
    '/Library/Java/JavaVirtualMachines/zulu-17.jdk/Contents/Home',
].filter(Boolean);

function isJdk17(home) {
    const release = path.join(home, 'release');
    if (!fs.existsSync(path.join(home, 'bin/java')) || !fs.existsSync(release)) {
        return false;
    }
    return /JAVA_VERSION="17/.test(fs.readFileSync(release, 'utf8'));
}

function resolveJdk17Home() {
    return CANDIDATES.find((home) => isJdk17(home));
}

const withDaemonJvm17 = (config) =>
    withDangerousMod(config, [
        'android',
        async (mod) => {
            const file = path.join(mod.modRequest.platformProjectRoot, 'gradle/gradle-daemon-jvm.properties');
            fs.mkdirSync(path.dirname(file), {recursive: true});
            fs.writeFileSync(
                file,
                '# Forced to 17: JDK 24+ breaks AGP CMake/prefab (react-native-worklets)\ntoolchainVersion=17\n',
            );
            return mod;
        },
    ]);

/**
 * AGP воспринимает предупреждение JDK 24+ «restricted method in java.lang.System»
 * как ошибку CMake (react-native-worklets configureCMake*). Фиксируем Gradle на JDK 17.
 */
const withGradleJdk17 = (config) => {
    const jdkHome = resolveJdk17Home();
    let next = config;

    if (jdkHome) {
        next = withGradleProperties(next, (mod) => {
            const without = mod.modResults.filter(
                (item) => !(item.type === 'property' && item.key === 'org.gradle.java.home'),
            );
            without.push({
                type: 'property',
                key: 'org.gradle.java.home',
                value: jdkHome,
            });
            mod.modResults = without;
            return mod;
        });
    }

    return withDaemonJvm17(next);
};

module.exports = withGradleJdk17;
