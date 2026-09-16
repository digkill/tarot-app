require('dotenv').config({quiet: true});

module.exports = {
  expo: {
    name: 'Tarot',
    slug: 'tarot',
    version: '1.1.9',
    scheme: 'mediarisetarot',
    orientation: 'portrait',
    icon: './assets/icon.png',
    userInterfaceStyle: 'dark',
    newArchEnabled: true,
    plugins: [
      'expo-asset',
      'expo-font',
      'expo-image',
      [
        'expo-localization',
        {
          supportedLocales: {
            ios: ['en', 'ru', 'th', 'zh-Hans', 'zh-Hant'],
            android: ['en', 'ru', 'th', 'zh', 'zh-Hans', 'zh-Hant'],
          },
        },
      ],
      'expo-sharing',
      [
        'expo-splash-screen',
        {
          image: './assets/icon.png',
          imageWidth: 200,
          resizeMode: 'contain',
          backgroundColor: '#0B1220',
        },
      ],
      './plugins/withSplashBranding',
      'expo-status-bar',
      'expo-video',
      'expo-web-browser',
      [
        './plugins/withRuStorePay',
        {
          consoleApplicationId: process.env.RUSTORE_CONSOLE_APP_ID || '',
          scheme: 'mediarisetarot',
        },
      ],
      './plugins/withGradleJdk17',
      './plugins/withAndroidNdkVersion',
      './plugins/withAndroidCleanCxx',
      './plugins/withDebugInstallGuard',
      './plugins/withRuStoreReleaseSigning',
      './plugins/withXcode27ClangWorkaround',
      './plugins/withXcode27SceneLifecycle',
      './plugins/withIosReleaseScheme',
    ],
    splash: {
      image: './assets/icon.png',
      resizeMode: 'contain',
      backgroundColor: '#0B1220',
    },
    ios: {
      supportsTablet: true,
      icon: './assets/icon-composer/tarot-ios27-default.png',
      bundleIdentifier: 'org.mediarise.tarot',
      appleTeamId: '39CP3623CD',
      buildNumber: '12',
      infoPlist: {
        ITSAppUsesNonExemptEncryption: false,
      },
    },
    android: {
      package: 'org.mediarise.tarot',
      versionCode: 12,
      permissions: [],
      adaptiveIcon: {
        foregroundImage: './assets/adaptive-icon.png',
        backgroundColor: '#0B1220',
      },
    },
    web: {
      favicon: './assets/favicon.png',
    },
    extra: {
      eas: {
        projectId: '2b61bc02-2377-4775-a6a3-2e7461cd14c6',
      },
      rustorePremiumProductId: process.env.RUSTORE_PREMIUM_PRODUCT_ID || 'premium_monthly',
      rustorePremiumProductIds: ['premium_monthly', 'premium_yearly', 'premium_lifetime'],
      apiBaseUrl: process.env.API_BASE_URL || 'https://tarot.sorapure.fun',
    },
    runtimeVersion: {
      policy: 'appVersion',
    },
    updates: {
      url: 'https://u.expo.dev/2b61bc02-2377-4775-a6a3-2e7461cd14c6',
    },
  },
};

