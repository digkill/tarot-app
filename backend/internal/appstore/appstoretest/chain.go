// Package appstoretest builds synthetic Apple-style signing chains for tests.
//
// It stands in for Apple's real root → intermediate → leaf chain so the
// verification code path can be exercised offline. It proves our handling, not
// the shape of Apple's actual certificates — one manual sandbox purchase before
// release remains mandatory.
package appstoretest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultBundleID matches the app's real bundle id.
const DefaultBundleID = "org.mediarise.tarot"

// markerOID is Apple's App Store server signing marker.
var markerOID = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 6, 11, 1}

type Chain struct {
	root         *x509.Certificate
	intermediate *x509.Certificate
	leaf         *x509.Certificate
	leafKey      *ecdsa.PrivateKey
}

// New builds a chain valid around now.
func New(t *testing.T, opts ...func(*x509.Certificate)) *Chain {
	t.Helper()
	return NewWindow(t, time.Now().Add(-24*time.Hour), time.Now().Add(365*24*time.Hour), opts...)
}

// NewWindow builds a chain with an explicit validity window, so a test can
// produce a genuine payload whose certificates have since expired.
func NewWindow(t *testing.T, notBefore, notAfter time.Time, opts ...func(*x509.Certificate)) *Chain {
	t.Helper()

	rootKey := genKey(t)
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	root := sign(t, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)

	interKey := genKey(t)
	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test Intermediate CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	intermediate := sign(t, interTmpl, root, &interKey.PublicKey, rootKey)

	leafKey := genKey(t)
	leafTmpl := &x509.Certificate{
		SerialNumber:    big.NewInt(3),
		Subject:         pkix.Name{CommonName: "Test App Store Signing Leaf"},
		NotBefore:       notBefore,
		NotAfter:        notAfter,
		KeyUsage:        x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{Id: markerOID, Value: []byte{0x05, 0x00}}},
	}
	for _, opt := range opts {
		opt(leafTmpl)
	}
	leaf := sign(t, leafTmpl, intermediate, &leafKey.PublicKey, interKey)

	return &Chain{root: root, intermediate: intermediate, leaf: leaf, leafKey: leafKey}
}

// WithoutMarkerOID drops the App Store signing marker from the leaf.
func WithoutMarkerOID(tmpl *x509.Certificate) {
	tmpl.ExtraExtensions = nil
}

// Root is the trust anchor to hand to appstore.Config.RootCA.
func (c *Chain) Root() *x509.Certificate { return c.root }

// LeafKey signs payloads; exposed so a test can forge one deliberately.
func (c *Chain) LeafKey() *ecdsa.PrivateKey { return c.leafKey }

// X5C is the header value Apple sends: leaf, intermediate, root, in standard
// base64 rather than JWT's URL-safe alphabet.
func (c *Chain) X5C() []string {
	return []string{
		base64.StdEncoding.EncodeToString(c.leaf.Raw),
		base64.StdEncoding.EncodeToString(c.intermediate.Raw),
		base64.StdEncoding.EncodeToString(c.root.Raw),
	}
}

// Sign produces a compact JWS the way Apple does.
func (c *Chain) Sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["x5c"] = c.X5C()
	signed, err := token.SignedString(c.leafKey)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

// TxClaims is a plausible JWSTransactionDecodedPayload for a yearly premium
// subscription. Mutators let a test change any single field.
func TxClaims(mutate ...func(jwt.MapClaims)) jwt.MapClaims {
	claims := jwt.MapClaims{
		"transactionId":         "2000000812345679",
		"originalTransactionId": "2000000812345678",
		"webOrderLineItemId":    "100000123",
		"bundleId":              DefaultBundleID,
		"productId":             "org.mediarise.tarot.premium.yearly",
		"purchaseDate":          float64(time.Now().Add(-time.Hour).UnixMilli()),
		"expiresDate":           float64(time.Now().Add(366 * 24 * time.Hour).UnixMilli()),
		"type":                  "Auto-Renewable Subscription",
		"inAppOwnershipType":    "PURCHASED",
		"signedDate":            float64(time.Now().UnixMilli()),
		"environment":           "Production",
		"transactionReason":     "PURCHASE",
		"price":                 float64(59990),
		"currency":              "USD",
		"appAccountToken":       "8f14e45f-ea1d-4b0a-9f3c-000000000001",
	}
	for _, m := range mutate {
		m(claims)
	}
	return claims
}

// NotificationClaims is a responseBodyV2DecodedPayload wrapping the given
// nested JWS payloads.
func NotificationClaims(txJWS, renewalJWS string, mutate ...func(jwt.MapClaims)) jwt.MapClaims {
	data := map[string]any{
		"appAppleId":    float64(6478901234),
		"bundleId":      DefaultBundleID,
		"bundleVersion": "13",
		"environment":   "Production",
	}
	if txJWS != "" {
		data["signedTransactionInfo"] = txJWS
	}
	if renewalJWS != "" {
		data["signedRenewalInfo"] = renewalJWS
	}
	claims := jwt.MapClaims{
		"notificationType": "DID_RENEW",
		"notificationUUID": "9ad8c4b1-0000-4000-8000-000000000002",
		"version":          "2.0",
		"signedDate":       float64(time.Now().UnixMilli()),
		"data":             data,
	}
	for _, m := range mutate {
		m(claims)
	}
	return claims
}

// TestNotificationClaims is a TEST notification, which carries no data at all.
func TestNotificationClaims(mutate ...func(jwt.MapClaims)) jwt.MapClaims {
	claims := jwt.MapClaims{
		"notificationType": "TEST",
		"notificationUUID": "9ad8c4b1-0000-4000-8000-000000000001",
		"version":          "2.0",
		"signedDate":       float64(time.Now().UnixMilli()),
	}
	for _, m := range mutate {
		m(claims)
	}
	return claims
}

func genKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func sign(t *testing.T, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, signer *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, signer)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
