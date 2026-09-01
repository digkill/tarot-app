export type ProductType =
  | "CONSUMABLE_PRODUCT"
  | "NON_CONSUMABLE_PRODUCT"
  | "SUBSCRIPTION";

export type ProductPurchaseStatus =
  | "INVOICE_CREATED"
  | "CANCELLED"
  | "PROCESSING"
  | "REJECTED"
  | "CONFIRMED"
  | "REFUNDED"
  | "REFUNDING"
  | "EXECUTING"
  | "EXPIRED"
  | "PAID"
  | "REVERSED";

export type SubscriptionPurchaseStatus =
  | "INVOICE_CREATED"
  | "CANCELLED"
  | "EXPIRED"
  | "PROCESSING"
  | "REJECTED"
  | "ACTIVE"
  | "PAUSED"
  | "TERMINATED"
  | "CLOSED";

export type PurchaseType = "ONE_STEP" | "TWO_STEP" | "UNDEFINED";

export type PreferredPurchaseType = "ONE_STEP" | "TWO_STEP";

export type SdkTheme = "LIGHT" | "DARK";

export type SubscriptionPeriod =
  | {
      type: "MAIN" | "TRIAL" | "PROMO";
      duration: string;
      currency: string;
      price: number;
    }
  | {
      type: "GRACE" | "HOLD";
      duration: string;
    };

export type SubscriptionInfo = {
  periods: SubscriptionPeriod[];
};

export type RuStoreErrorCode =
  | "RuStorePaymentNetworkException"
  | "RuStorePaymentCommonException"
  | "RuStorePayClientAlreadyExist"
  | "RuStorePayClientNotCreated"
  | "RuStorePayInvalidActivePurchase"
  | "RuStorePayInvalidConsoleAppId"
  | "RuStorePaySignatureException"
  | "EmptyPaymentTokenException"
  | "InvalidCardBindingIdException"
  | "ApplicationSchemeWasNotProvided"
  | "ProductPurchaseException"
  | "ProductPurchaseCancelled"
  | "RuStoreNotInstalledException"
  | "RuStoreOutdatedException"
  | "RuStoreUserUnauthorizedException"
  | "RuStoreApplicationBannedException"
  | "RuStoreUserBannedException";

export type Product = {
  productId: string;
  type: ProductType;
  amountLabel: string;
  price?: number;
  currency: string;
  imageUrl: string;
  title: string;
  description?: string;
  subscriptionInfo?: SubscriptionInfo;
};

export type ProductPurchase = {
  purchaseId: string;
  invoiceId: string;
  orderId?: string;
  purchaseType: PurchaseType;
  status: ProductPurchaseStatus;
  description: string;
  purchaseTime?: string;
  price: number;
  amountLabel: string;
  currency: string;
  developerPayload?: string;
  sandbox: boolean;
  productId: string;
  quantity: number;
  productType: ProductType;
};

export type SubscriptionPurchase = {
  purchaseId: string;
  invoiceId: string;
  orderId?: string;
  purchaseType: PurchaseType;
  status: SubscriptionPurchaseStatus;
  description: string;
  purchaseTime?: string;
  price: number;
  amountLabel: string;
  currency: string;
  developerPayload?: string;
  sandbox: boolean;
  productId: string;
  expirationDate: string;
  gracePeriodEnabled: boolean;
};

export type PurchaseResultWrapper = {
  productPurchase?: ProductPurchase;
  subscriptionPurchase?: SubscriptionPurchase;
};

export type PurchaseAvailability = {
  availability: boolean;
  cause?: string;
};

export type PurchaseParams = {
  productType?: ProductType;
  purchaseStatus?: ProductPurchaseStatus | SubscriptionPurchaseStatus;
};

export type Purchase = {
  productPurchase?: ProductPurchase;
  subscriptionPurchase?: SubscriptionPurchase;
}

export type ProductPurchaseResult = {
  orderId?: string;
  purchaseId: string;
  productId: string;
  invoiceId: string;
  purchaseType: PurchaseType;
  productType: ProductType;
  quantity: number;
  sandbox: boolean;
};

export type PurchaseEventListenerParams = {
  purchaseId?: string | null,
  invoiceId?: string | null
}

export type PurchaseEventListener = {
  onPurchaseCreated?: (params: PurchaseEventListenerParams) => void;
  onPaymentStarted?: (params: PurchaseEventListenerParams) => void;
  onPaymentCompleted?: (params: PurchaseEventListenerParams) => void;
  onPaymentFailed?: (params: PurchaseEventListenerParams) => void;
  onPurchaseCancelled?: (params: PurchaseEventListenerParams) => void;
};

export type PurchaseEventListenerCallbackMap = {[k in keyof PurchaseEventListener]?: string};

export type RuStorePayError = {
  code: RuStoreErrorCode;
  message: string;
};
