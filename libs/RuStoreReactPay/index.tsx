import {NativeEventEmitter, NativeModules} from "react-native";

export * from "./types";

import type {
  Product,
  PurchaseAvailability,
  ProductPurchaseResult,
  PurchaseParams,
  Purchase,
  PreferredPurchaseType,
  SdkTheme,
  PurchaseEventListener,
  PurchaseEventListenerParams,
  PurchaseEventListenerCallbackMap,
  RuStorePayError,
  RuStoreErrorCode,
} from "./types";

const VALID_ERROR_CODE: RuStoreErrorCode[] = [
  "RuStorePaymentNetworkException",
  "RuStorePaymentCommonException",
  "RuStorePayClientAlreadyExist",
  "RuStorePayClientNotCreated",
  "RuStorePayInvalidActivePurchase",
  "RuStorePayInvalidConsoleAppId",
  "RuStorePaySignatureException",
  "EmptyPaymentTokenException",
  "InvalidCardBindingIdException",
  "ApplicationSchemeWasNotProvided",
  "ProductPurchaseException",
  "ProductPurchaseCancelled",
  "RuStoreNotInstalledException",
  "RuStoreOutdatedException",
  "RuStoreUserUnauthorizedException",
  "RuStoreApplicationBannedException",
  "RuStoreUserBannedException"
];

interface RuStoreReactPayInterface {
  getPurchase(purchaseId: string): Promise<Purchase>;

  getPurchases(params?: PurchaseParams): Promise<Purchase[]>;

  getPurchaseAvailability(): Promise<PurchaseAvailability>;

  getProducts(ids: string[]): Promise<Product[]>;

  purchase(params: {
    productId: string | null;
    orderId?: string | null;
    quantity?: number | null;
    developerPayload?: string | null;
    appUserId?: string | null;
    appUserEmail?: string | null;
    preferredPurchaseType?: PreferredPurchaseType | null;
    sdkTheme?: SdkTheme | null;
    purchaseEventListener?: PurchaseEventListener | null;
  }): Promise<ProductPurchaseResult>;

  purchaseTwoStep(params: {
    productId: string | null;
    orderId?: string | null;
    quantity?: number | null;
    developerPayload?: string | null;
    appUserId?: string | null;
    appUserEmail?: string | null;
    sdkTheme?: SdkTheme | null;
    purchaseEventListener?: PurchaseEventListener | null;
  }): Promise<ProductPurchaseResult>;

  confirmTwoStepPurchase(params: {
    purchaseId: string;
    developerPayload?: string | null;
  }): Promise<void>;

  cancelTwoStepPurchase(purchaseId: string): Promise<void>;

  getUserAuthorizationStatus(): Promise<boolean>;

  isRuStoreInstalled(): Promise<boolean>;

  openRuStore(): Promise<void>;

  openRuStoreAuthorization(): Promise<void>;

  openRuStoreDownloadInstruction(): Promise<void>;

  isRuStorePayError: (error: unknown) => error is RuStorePayError;
}

class RuStoreReactPayModule implements RuStoreReactPayInterface {
  private eventEmitter: NativeEventEmitter | null = null;

  private registerPurchaseEvents (purchaseEventListener: PurchaseEventListener | null): PurchaseEventListenerCallbackMap {
    const purchaseEventListenerIds: PurchaseEventListenerCallbackMap = {};

    if (purchaseEventListener) {
      const eventEmitter = this.eventEmitter ??=
        new NativeEventEmitter(NativeModules.RuStoreReactPaySDKModule);
      Object.keys(purchaseEventListener).forEach((key) => {
        const eventName = key as keyof PurchaseEventListener;
        const callbackFn = purchaseEventListener[eventName];

        if (callbackFn) {
          purchaseEventListenerIds[eventName] = `purchase_${eventName}_${Date.now()}`;

          const subscription = eventEmitter.addListener(purchaseEventListenerIds[eventName], (params: PurchaseEventListenerParams) => {
            callbackFn(params);
            subscription.remove();
          });
        }
      });
    }

    return purchaseEventListenerIds;
  }

  async getPurchase (purchaseId: string): Promise<Purchase> {
    return await NativeModules.RuStoreReactPaySDKModule.getPurchase(
        purchaseId
    );
  }

  async getPurchases (params?: PurchaseParams): Promise<Purchase[]> {
    return await NativeModules.RuStoreReactPaySDKModule.getPurchases(
      params?.productType ?? null,
      params?.purchaseStatus ?? null,
    );
  }

  async getPurchaseAvailability (): Promise<PurchaseAvailability> {
    return await NativeModules.RuStoreReactPaySDKModule.getPurchaseAvailability();
  }

  async getProducts (ids: string[]): Promise<Product[]> {
    return await NativeModules.RuStoreReactPaySDKModule.getProducts(ids);
  }

  async purchase ({
    productId,
    orderId = null,
    quantity = null,
    developerPayload = null,
    appUserId = null,
    appUserEmail = null,
    preferredPurchaseType = null,
    sdkTheme = null,
    purchaseEventListener = null,
  }: Parameters<RuStoreReactPayInterface['purchase']>[0]): Promise<ProductPurchaseResult> {
    const purchaseEventListenerIds = this.registerPurchaseEvents(purchaseEventListener);

    return await NativeModules.RuStoreReactPaySDKModule.purchase(
      productId,
      orderId,
      quantity,
      developerPayload,
      appUserId,
      appUserEmail,
      preferredPurchaseType,
      sdkTheme,
      purchaseEventListenerIds,
    );
  }

  async purchaseTwoStep ({
    productId,
    orderId = null,
    quantity = null,
    developerPayload = null,
    appUserId = null,
    appUserEmail = null,
    sdkTheme = null,
    purchaseEventListener = null,
  }: Parameters<RuStoreReactPayInterface['purchaseTwoStep']>[0]): Promise<ProductPurchaseResult> {
    const purchaseEventListenerIds = this.registerPurchaseEvents(purchaseEventListener);

    return await NativeModules.RuStoreReactPaySDKModule.purchaseTwoStep(
      productId,
      orderId,
      quantity,
      developerPayload,
      appUserId,
      appUserEmail,
      sdkTheme,
      purchaseEventListenerIds,
    );
  }

  async confirmTwoStepPurchase ({ purchaseId, developerPayload = null }: Parameters<RuStoreReactPayInterface['confirmTwoStepPurchase']>[0])  {
    return await NativeModules.RuStoreReactPaySDKModule.confirmTwoStepPurchase(
      purchaseId,
      developerPayload,
    );
  }

  async cancelTwoStepPurchase (purchaseId: string): Promise<void> {
    return await NativeModules.RuStoreReactPaySDKModule.cancelTwoStepPurchase(
        purchaseId,
    );
  }

  async getUserAuthorizationStatus () {
    return await NativeModules.RuStoreReactPaySDKModule.getUserAuthorizationStatus();
  }

  async isRuStoreInstalled () {
    return await NativeModules.RuStoreReactPaySDKModule.isRuStoreInstalled();
  }

  async openRuStore () {
    return await NativeModules.RuStoreReactPaySDKModule.openRuStore();
  }

  async openRuStoreAuthorization () {
    return await NativeModules.RuStoreReactPaySDKModule.openRuStoreAuthorization();
  }

  async openRuStoreDownloadInstruction () {
    return await NativeModules.RuStoreReactPaySDKModule.openRuStoreDownloadInstruction();
  }

  isRuStorePayError (error: unknown): error is RuStorePayError {
    return (
      typeof error === "object" &&
      error !== null &&
      "code" in error &&
      "message" in error &&
      typeof (error as RuStorePayError).code === "string" &&
      typeof (error as RuStorePayError).message === "string" &&
      VALID_ERROR_CODE.includes((error as RuStorePayError).code)
    );
  }
}

export const RuStoreReactPay = new RuStoreReactPayModule();
