const fs = require('fs');
const path = require('path');
const {withAndroidStyles, withDangerousMod} = require('expo/config-plugins');

const SRC = 'assets/splash_branding.png';
const DRAWABLE_NAME = 'splashscreen_branding.png';

/**
 * Android 12: иконка + branding внизу («Powered by MediaRise»).
 * На Android 13+ branding image система больше не показывает —
 * тот же экран рисует JS (VideoSplash) перед роликом.
 */
const withSplashBranding = (config) => {
    config = withDangerousMod(config, [
        'android',
        async (mod) => {
            const src = path.join(mod.modRequest.projectRoot, SRC);
            if (!fs.existsSync(src)) {
                return mod;
            }
            const destDir = path.join(
                mod.modRequest.platformProjectRoot,
                'app/src/main/res/drawable-xxhdpi',
            );
            fs.mkdirSync(destDir, {recursive: true});
            fs.copyFileSync(src, path.join(destDir, DRAWABLE_NAME));
            return mod;
        },
    ]);

    return withAndroidStyles(config, (mod) => {
        const splash = mod.modResults.resources.style?.find(
            (style) => style.$?.name === 'Theme.App.SplashScreen',
        );
        if (!splash) {
            return mod;
        }
        const items = splash.item ?? [];
        const hasBranding = items.some(
            (item) => item.$?.name === 'android:windowSplashScreenBrandingImage',
        );
        if (!hasBranding) {
            items.push({
                $: {name: 'android:windowSplashScreenBrandingImage'},
                _: '@drawable/splashscreen_branding',
            });
            splash.item = items;
        }
        return mod;
    });
};

module.exports = withSplashBranding;
