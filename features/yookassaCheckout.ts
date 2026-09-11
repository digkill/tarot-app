import * as WebBrowser from 'expo-web-browser';
import {createCheckoutRequest, fetchCheckoutRequest, type CheckoutStatus} from './billingApi';

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export const startYooKassaCheckout = async (productId: string): Promise<CheckoutStatus | null> => {
    const session = await createCheckoutRequest(productId);
    const result = await WebBrowser.openAuthSessionAsync(session.checkoutUrl, 'mediarisetarot://billing/complete');
    if (result.type !== 'success') {
        // Closing the browser ends this attempt; it does not cancel a payment at the provider.
        return null;
    }
    let last = await fetchCheckoutRequest(session.transactionId);
    for (let i = 0; i < 10 && last.status === 'pending' && !last.hasPremium; i += 1) {
        await sleep(1500);
        last = await fetchCheckoutRequest(session.transactionId);
    }
    return last;
};
