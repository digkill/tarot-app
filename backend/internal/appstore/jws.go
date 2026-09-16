package appstore

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrNotConfigured       = errors.New("appstore: not configured")
	ErrBadSignature        = errors.New("appstore: signature verification failed")
	ErrBundleMismatch      = errors.New("appstore: bundle id mismatch")
	ErrSandboxRejected     = errors.New("appstore: sandbox transactions are not accepted")
	ErrFamilyShared        = errors.New("appstore: family shared purchase")
	ErrNotFound            = errors.New("appstore: not found")
	ErrProviderUnavailable = errors.New("appstore: provider unavailable")
)

// appleMarkerOID marks a certificate as an App Store server signing leaf.
// Without this check any certificate Apple has ever issued — a developer
// certificate, for instance — would chain to the same root and be accepted.
const appleMarkerOID = "1.2.840.113635.100.6.11.1"

// futureSkew bounds how far ahead of our clock a claimed signing time may be.
const futureSkew = 5 * time.Minute

// VerifyTransaction verifies a JWSTransaction and checks it belongs to our app.
func (c *Client) VerifyTransaction(compact string) (*Transaction, error) {
	var tx Transaction
	if err := c.verifyJWS(compact, &tx); err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(tx.BundleID), c.bundleID) {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrBundleMismatch, tx.BundleID, c.bundleID)
	}
	if tx.IsSandbox() && !c.allowSandbox {
		return nil, ErrSandboxRejected
	}
	return &tx, nil
}

// VerifyRenewalInfo verifies a JWSRenewalInfo payload.
func (c *Client) VerifyRenewalInfo(compact string) (*RenewalInfo, error) {
	var info RenewalInfo
	if err := c.verifyJWS(compact, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// VerifyNotification verifies an App Store Server Notification V2 signedPayload
// and every JWS nested inside it. A TEST notification carries no data at all,
// so the bundle check is skipped for it — otherwise Apple's own "request a test
// notification" button could never succeed.
func (c *Client) VerifyNotification(signedPayload string) (*Notification, error) {
	var n Notification
	if err := c.verifyJWS(signedPayload, &n); err != nil {
		return nil, err
	}
	if n.IsTest() {
		return &n, nil
	}
	if n.Data == nil {
		// Summary-only notifications (RENEWAL_EXTENSION) still name the app.
		if n.Summary != nil && !strings.EqualFold(strings.TrimSpace(n.Summary.BundleID), c.bundleID) {
			return nil, fmt.Errorf("%w: summary bundle %q", ErrBundleMismatch, n.Summary.BundleID)
		}
		return &n, nil
	}
	if !strings.EqualFold(strings.TrimSpace(n.Data.BundleID), c.bundleID) {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrBundleMismatch, n.Data.BundleID, c.bundleID)
	}
	if c.appAppleID != 0 && n.Data.AppAppleID != 0 && n.Data.AppAppleID != c.appAppleID {
		return nil, fmt.Errorf("%w: appAppleId %d, want %d", ErrBundleMismatch, n.Data.AppAppleID, c.appAppleID)
	}
	if s := strings.TrimSpace(n.Data.SignedTransactionInfo); s != "" {
		tx, err := c.VerifyTransaction(s)
		if err != nil {
			return nil, fmt.Errorf("signedTransactionInfo: %w", err)
		}
		n.Transaction = tx
	}
	if s := strings.TrimSpace(n.Data.SignedRenewalInfo); s != "" {
		info, err := c.VerifyRenewalInfo(s)
		if err != nil {
			return nil, fmt.Errorf("signedRenewalInfo: %w", err)
		}
		n.Renewal = info
	}
	return &n, nil
}

func (c *Client) verifyJWS(compact string, dst jwt.Claims) error {
	if !c.Enabled() {
		return ErrNotConfigured
	}
	compact = strings.TrimSpace(compact)
	if compact == "" {
		return fmt.Errorf("%w: empty payload", ErrBadSignature)
	}
	// The signing time is read from the not-yet-verified payload purely to pick
	// the moment at which the certificate chain is evaluated. A legitimately old
	// JWS — a year-old transaction replayed from currentEntitlements — must not
	// fail because Apple has since rotated its leaf. This cannot be abused to
	// forge anything: the window can only widen inside the leaf's own validity,
	// and the signature itself is still checked against that leaf's key.
	chainAt := c.chainTime(peekSignedDate(compact))

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"ES256"}))
	if _, err := parser.ParseWithClaims(compact, dst, func(t *jwt.Token) (any, error) {
		return c.leafKey(t, chainAt)
	}); err != nil {
		return fmt.Errorf("%w: %v", ErrBadSignature, err)
	}
	return nil
}

// leafKey extracts the x5c chain, pins it to our root and returns the leaf's
// public key. golang-jwt has no x5c support, so this is done by hand.
func (c *Client) leafKey(t *jwt.Token, chainAt time.Time) (*ecdsa.PublicKey, error) {
	rawChain, ok := t.Header["x5c"].([]any)
	if !ok {
		return nil, errors.New("x5c header missing")
	}
	// Apple sends leaf, intermediate, root. Anything outside this range is not
	// a shape we know how to reason about.
	if len(rawChain) < 2 || len(rawChain) > 5 {
		return nil, fmt.Errorf("x5c has %d certificates", len(rawChain))
	}

	certs := make([]*x509.Certificate, 0, len(rawChain))
	for i, entry := range rawChain {
		encoded, ok := entry.(string)
		if !ok {
			return nil, fmt.Errorf("x5c[%d] is not a string", i)
		}
		// x5c is standard base64, not the URL-safe alphabet used by JWT itself.
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("x5c[%d]: %w", i, err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("x5c[%d]: %w", i, err)
		}
		certs = append(certs, cert)
	}

	root, err := c.root()
	if err != nil {
		return nil, err
	}
	// The pin: the roots pool holds exactly one certificate, so Verify below can
	// only succeed if the chain really terminates at Apple Root CA G3.
	roots := x509.NewCertPool()
	roots.AddCert(root)

	// When Apple includes the root in the chain it must be byte-identical to
	// ours; a different self-signed tail is an attempt to swap the anchor.
	last := certs[len(certs)-1]
	if isSelfSigned(last) && !bytes.Equal(last.Raw, root.Raw) {
		return nil, errors.New("x5c terminates at an unexpected root certificate")
	}

	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}

	leaf := certs[0]
	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   chainAt,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return nil, fmt.Errorf("verify x5c chain: %w", err)
	}
	if !hasMarkerOID(leaf) {
		return nil, fmt.Errorf("leaf certificate lacks the App Store signing marker %s", appleMarkerOID)
	}

	pub, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("leaf public key is %T, want ECDSA", leaf.PublicKey)
	}
	return pub, nil
}

func (c *Client) chainTime(signedAt time.Time) time.Time {
	now := c.now()
	if signedAt.IsZero() {
		return now
	}
	if signedAt.After(now.Add(futureSkew)) {
		return now
	}
	if signedAt.Before(now) {
		return signedAt
	}
	return now
}

func hasMarkerOID(cert *x509.Certificate) bool {
	for _, ext := range cert.Extensions {
		if ext.Id.String() == appleMarkerOID {
			return true
		}
	}
	for _, oid := range cert.UnknownExtKeyUsage {
		if oid.String() == appleMarkerOID {
			return true
		}
	}
	return false
}

func isSelfSigned(cert *x509.Certificate) bool {
	return cert.IsCA && bytes.Equal(cert.RawIssuer, cert.RawSubject)
}

// peekSignedDate reads the signedDate claim without verifying anything.
func peekSignedDate(compact string) time.Time {
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var probe struct {
		SignedDate int64 `json:"signedDate"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil || probe.SignedDate <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(probe.SignedDate).UTC()
}
