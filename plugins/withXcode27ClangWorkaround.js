const {withPodfile, withXcodeProject} = require('expo/config-plugins');

const MARKER = 'Xcode 27 Apple Clang 21 index-store ICE';

const POD_POST_INSTALL = `
    # ${MARKER}: compiling RNScreens RNSTabBarController.mm crashes clang
    # in WrappingIndexRecordAction::EndSourceFileAction. Expo CLI already
    # passes COMPILER_INDEX_STORE_ENABLE=NO; Xcode GUI does not.
    installer.pods_project.root_object.attributes['LastUpgradeCheck'] = '2700'
    installer.pods_project.targets.each do |target|
      target.build_configurations.each do |config|
        config.build_settings['COMPILER_INDEX_STORE_ENABLE'] = 'NO'
      end
    end
`;

const withXcode27ClangWorkaround = (config) => {
    config = withPodfile(config, (mod) => {
        let {contents} = mod.modResults;
        if (!contents.includes(MARKER)) {
            if (!contents.includes('react_native_post_install(')) {
                throw new Error('withXcode27ClangWorkaround: Podfile has no react_native_post_install');
            }
            contents = contents.replace(/react_native_post_install\([\s\S]*?\)\n/, (match) => `${match}${POD_POST_INSTALL}`);
            mod.modResults.contents = contents;
        }
        return mod;
    });

    return withXcodeProject(config, (mod) => {
        const project = mod.modResults;
        const configs = project.pbxXCBuildConfigurationSection();
        for (const key of Object.keys(configs)) {
            const item = configs[key];
            if (item && item.buildSettings) {
                item.buildSettings.COMPILER_INDEX_STORE_ENABLE = 'NO';
            }
        }
        const projects = project.pbxProjectSection();
        for (const key of Object.keys(projects)) {
            const item = projects[key];
            if (item && item.attributes) {
                item.attributes.LastUpgradeCheck = 2700;
            }
        }
        return mod;
    });
};

module.exports = withXcode27ClangWorkaround;
