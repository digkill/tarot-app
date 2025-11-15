require('dotenv').config();

module.exports = {
  expo: {
    name: 'Tarot',
    slug: 'tarot',
    version: '1.1.0',
    orientation: 'portrait',
    icon: './assets/icon.png',
    userInterfaceStyle: 'light',
    newArchEnabled: true,
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
      versionCode: 1,
      permissions: [],
      adaptiveIcon: {
        foregroundImage: './assets/adaptive-icon.png',
        backgroundColor: '#ffffff',
      },
      edgeToEdgeEnabled: true,
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
    },
    runtimeVersion: {
      policy: 'appVersion',
    },
    updates: {
      url: 'https://u.expo.dev/2b61bc02-2377-4775-a6a3-2e7461cd14c6',
    },
  },
};

