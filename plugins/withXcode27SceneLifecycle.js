const {withAppDelegate, withInfoPlist, withXcodeProject} = require('expo/config-plugins');

const MARKER = 'Xcode 27 UIScene lifecycle';

const SCENE_DELEGATE = `
// ${MARKER}
@objc(SceneDelegate)
class SceneDelegate: UIResponder, UIWindowSceneDelegate {
  var window: UIWindow?

  func scene(
    _ scene: UIScene,
    willConnectTo session: UISceneSession,
    options connectionOptions: UIScene.ConnectionOptions
  ) {
    guard let windowScene = scene as? UIWindowScene,
      let appDelegate = UIApplication.shared.delegate as? AppDelegate,
      let factory = appDelegate.reactNativeFactory else {
      return
    }

    let window = UIWindow(windowScene: windowScene)
    self.window = window
    appDelegate.window = window

    let browsingWebActivity = connectionOptions.userActivities.first {
      $0.activityType == NSUserActivityTypeBrowsingWeb
    }
    factory.startReactNative(
      withModuleName: "main",
      in: window,
      launchOptions: Self.launchOptions(
        url: connectionOptions.urlContexts.first?.url,
        userActivity: browsingWebActivity
      )
    )

    Self.route(urlContexts: connectionOptions.urlContexts)
    connectionOptions.userActivities.forEach { Self.route(userActivity: $0) }
  }

  func sceneDidDisconnect(_ scene: UIScene) {
    window = nil
  }

  func sceneDidBecomeActive(_ scene: UIScene) {
    ExpoAppDelegateSubscriberManager.applicationDidBecomeActive(UIApplication.shared)
  }

  func sceneWillResignActive(_ scene: UIScene) {
    ExpoAppDelegateSubscriberManager.applicationWillResignActive(UIApplication.shared)
  }

  func sceneWillEnterForeground(_ scene: UIScene) {
    ExpoAppDelegateSubscriberManager.applicationWillEnterForeground(UIApplication.shared)
  }

  func sceneDidEnterBackground(_ scene: UIScene) {
    ExpoAppDelegateSubscriberManager.applicationDidEnterBackground(UIApplication.shared)
  }

  func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
    Self.route(urlContexts: URLContexts)
  }

  func scene(_ scene: UIScene, continue userActivity: NSUserActivity) {
    Self.route(userActivity: userActivity)
  }

  private static func launchOptions(
    url: URL?,
    userActivity: NSUserActivity?
  ) -> [UIApplication.LaunchOptionsKey: Any]? {
    var launchOptions: [UIApplication.LaunchOptionsKey: Any] = [:]
    if let url {
      launchOptions[UIApplication.LaunchOptionsKey(
        rawValue: "UIApplicationLaunchOptionsURLKey"
      )] = url
    }
    if let userActivity {
      launchOptions[UIApplication.LaunchOptionsKey(
        rawValue: "UIApplicationLaunchOptionsUserActivityDictionaryKey"
      )] = [
        "UIApplicationLaunchOptionsUserActivityTypeKey": userActivity.activityType,
        "UIApplicationLaunchOptionsUserActivityKey": userActivity,
      ]
    }
    return launchOptions.isEmpty ? nil : launchOptions
  }

  private static func route(urlContexts: Set<UIOpenURLContext>) {
    for context in urlContexts {
      let options = openURLOptions(from: context.options)
      _ = ExpoAppDelegateSubscriberManager.application(
        UIApplication.shared,
        open: context.url,
        options: options
      )
      RCTLinkingManager.application(
        UIApplication.shared,
        open: context.url,
        options: options
      )
    }
  }

  private static func route(userActivity: NSUserActivity) {
    _ = ExpoAppDelegateSubscriberManager.application(
      UIApplication.shared,
      continue: userActivity,
      restorationHandler: { _ in }
    )
    RCTLinkingManager.application(
      UIApplication.shared,
      continue: userActivity,
      restorationHandler: { _ in }
    )
  }

  private static func openURLOptions(
    from sceneOptions: UIScene.OpenURLOptions
  ) -> [UIApplication.OpenURLOptionsKey: Any] {
    var options: [UIApplication.OpenURLOptionsKey: Any] = [:]
    if let sourceApplication = sceneOptions.sourceApplication {
      options[.sourceApplication] = sourceApplication
    }
    if let annotation = sceneOptions.annotation {
      options[.annotation] = annotation
    }
    options[.openInPlace] = sceneOptions.openInPlace
    return options
  }
}

`;

const withXcode27SceneLifecycle = (config) => {
    config = withInfoPlist(config, (mod) => {
        mod.modResults.UIApplicationSceneManifest = {
            UIApplicationSupportsMultipleScenes: false,
            UISceneConfigurations: {
                UIWindowSceneSessionRoleApplication: [
                    {
                        UISceneConfigurationName: 'Default Configuration',
                        UISceneDelegateClassName: '$(PRODUCT_MODULE_NAME).SceneDelegate',
                    },
                ],
            },
        };
        return mod;
    });

    config = withAppDelegate(config, (mod) => {
        if (mod.modResults.language !== 'swift') {
            throw new Error('withXcode27SceneLifecycle requires a Swift AppDelegate');
        }

        let {contents} = mod.modResults;
        if (!contents.includes('internal import ExpoModulesCore')) {
            contents = contents.replace(
                'import ReactAppDependencyProvider',
                'import ReactAppDependencyProvider\ninternal import ExpoModulesCore',
            );
        }

        contents = contents.replace(
            /#if os\(iOS\) \|\| os\(tvOS\)\n\s*window = UIWindow\(frame: UIScreen\.main\.bounds\)[\s\S]*?#endif\n/,
            `// The window and React Native root are created by SceneDelegate (${MARKER}).\n`,
        );

        if (!contents.includes(`// ${MARKER}`)) {
            if (!contents.includes('class ReactNativeDelegate:')) {
                throw new Error('withXcode27SceneLifecycle: ReactNativeDelegate anchor not found');
            }
            contents = contents.replace('class ReactNativeDelegate:', `${SCENE_DELEGATE}class ReactNativeDelegate:`);
        }

        mod.modResults.contents = contents;
        return mod;
    });

    return withXcodeProject(config, (mod) => {
        const version = config.version;
        const buildNumber = config.ios?.buildNumber;
        const teamId = config.ios?.appleTeamId;
        const configurations = mod.modResults.pbxXCBuildConfigurationSection();

        for (const key of Object.keys(configurations)) {
            const item = configurations[key];
            if (!item?.buildSettings) {
                continue;
            }
            if (version) {
                item.buildSettings.MARKETING_VERSION = version;
            }
            if (buildNumber) {
                item.buildSettings.CURRENT_PROJECT_VERSION = buildNumber;
            }
            if (teamId && item.buildSettings.PRODUCT_BUNDLE_IDENTIFIER) {
                item.buildSettings.DEVELOPMENT_TEAM = teamId;
            }
        }
        return mod;
    });
};

module.exports = withXcode27SceneLifecycle;
