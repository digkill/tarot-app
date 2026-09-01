require('dotenv').config({quiet: true});

module.exports = {
  expo: {
    name: 'Tarot',
    slug: 'tarot',
    version: '1.1.0',
    scheme: 'mediarisetarot',
    orientation: 'portrait',
    icon: './assets/icon.png',
    userInterfaceStyle: 'light',
    newArchEnabled: true,
    plugins: [
      'expo-asset',
      'expo-font',
      'expo-image',
      'expo-localization',
      'expo-sharing',
      'expo-splash-screen',
      'expo-status-bar',
      'expo-web-browser',
      [
        './plugins/withRuStorePay',
        {
          consoleApplicationId: process.env.RUSTORE_CONSOLE_APP_ID || '',
          scheme: 'mediarisetarot',
        },
      ],
    ],
    splash: {
      image: './assets/splash-icon.png',
      resizeMode: 'contain',
      backgroundColor: '#ffffff',
    },
    ios: {
      supportsTablet: true,
    },
    android: {
      package: 'org.mediarise.tarot',
      versionCode: 2,
      permissions: [],
      adaptiveIcon: {
        foregroundImage: './assets/adaptive-icon.png',
        backgroundColor: '#ffffff',
      },
    },
    web: {
      favicon: './assets/favicon.png',
    },
    extra: {
      eas: {
        projectId: '2b61bc02-2377-4775-a6a3-2e7461cd14c6',
      },
      openaiApiKey: process.env.OPENAI_API_KEY,
      openaiTarotModel: process.env.OPENAI_TAROT_MODEL || 'gpt-4o-mini',
      rustorePremiumProductId: process.env.RUSTORE_PREMIUM_PRODUCT_ID || 'premium_monthly',
    },
    runtimeVersion: {
      policy: 'appVersion',
    },
    updates: {
      url: 'https://u.expo.dev/2b61bc02-2377-4775-a6a3-2e7461cd14c6',
    },
  },
};

