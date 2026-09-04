const {withAppBuildGradle} = require('expo/config-plugins');

const MARKER = 'TAROT_SKIP_CMAKE_CLEAN';

const SNIPPET = `
// ${MARKER}
// RN new-arch: Gradle \`clean\` removes library codegen dirs first, then CMake
// reconfigures and fails. Skip ninja clean and just delete app/.cxx.
gradle.taskGraph.whenReady {
    tasks.matching { it.name.startsWith("externalNativeBuildClean") }.configureEach {
        enabled = false
    }
}
afterEvaluate {
    tasks.named("clean").configure {
        doFirst {
            delete file("\${projectDir}/.cxx")
        }
    }
}

// TAROT_WORKLETS_PREFAB_ORDER
gradle.projectsEvaluated {
    def worklets = rootProject.subprojects.find { it.name == "react-native-worklets" }
    def reanimated = rootProject.subprojects.find { it.name == "react-native-reanimated" }
    if (worklets == null || reanimated == null) {
        return
    }
    def workletsNative = worklets.tasks.matching { task ->
        task.name == "externalNativeBuildRelease" || task.name == "prefabReleaseConfigurePackage"
    }
    reanimated.tasks.matching { task ->
        task.name.startsWith("configureCMakeRelWithDebInfo") ||
            task.name.startsWith("buildCMakeRelWithDebInfo")
    }.configureEach { task ->
        task.dependsOn(workletsNative)
    }
}
`;

/**
 * Android Studio Build > Clean Project fails on this RN/Expo app because
 * :app:externalNativeBuildCleanRelease re-runs CMake after codegen was deleted.
 */
const withAndroidCleanCxx = (config) =>
    withAppBuildGradle(config, (mod) => {
        if (!mod.modResults.contents.includes(MARKER)) {
            mod.modResults.contents = `${mod.modResults.contents.trimEnd()}\n${SNIPPET}`;
        }
        return mod;
    });

module.exports = withAndroidCleanCxx;
