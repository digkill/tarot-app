import {isPaymentsSupported, readStorePremiumStatus} from './payments';
import {reportSubscriptionStatus} from './billingApi';

export const syncStorePremiumWithServer = async (): Promise<void> => {
    if (!isPaymentsSupported()) {
        return;
    }
    try {
        const status = await readStorePremiumStatus();
        await reportSubscriptionStatus({
            active: status.active,
            productId: status.productId,
            expiresAt: status.expiresAt,
        });
    } catch {
        // RuStore or network failed — keep the last server entitlement.
    }
};
