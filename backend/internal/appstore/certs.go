package appstore

import (
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"
)

// AppleRootCAG3SHA256 is the SHA-256 fingerprint of the embedded certificate,
// as published by Apple at https://www.apple.com/certificateauthority/.
// It is asserted in a test, so swapping the .cer file fails the build.
const AppleRootCAG3SHA256 = "63343abfb89a6a03ebb57e9b3f5fa7be7c4f5c756f3017b3a8c488c3653e9179"

// The trust anchor for every signed payload Apple sends us is embedded rather
// than configured. A blank or missing env value would degrade silently into "no
// certificate pinning", and that is the one failure mode that turns purchase
// verification into an unauthenticated premium grant. Apple Root CA G3 is valid
// until 2039 and has never been rotated; APPLE_ROOT_CA_FILE exists as an escape
// hatch if that ever changes.
//
//go:embed certs/AppleRootCA-G3.cer
var appleRootG3DER []byte

var embeddedRoot = sync.OnceValues(func() (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(appleRootG3DER)
	if err != nil {
		return nil, fmt.Errorf("parse embedded Apple root CA: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	if got := hex.EncodeToString(sum[:]); got != AppleRootCAG3SHA256 {
		return nil, fmt.Errorf("embedded Apple root CA fingerprint is %s, want %s", got, AppleRootCAG3SHA256)
	}
	return cert, nil
})

// EmbeddedRoot returns the pinned Apple Root CA G3.
func EmbeddedRoot() (*x509.Certificate, error) {
	return embeddedRoot()
}

// LoadRootCAFile reads a replacement trust anchor, in PEM or raw DER.
func LoadRootCAFile(path string) (*x509.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read root CA file: %w", err)
	}
	if strings.Contains(string(raw), "-----BEGIN") {
		block, _ := pem.Decode(raw)
		if block == nil {
			return nil, fmt.Errorf("root CA file %s: no PEM block", path)
		}
		raw = block.Bytes
	}
	cert, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, fmt.Errorf("parse root CA file %s: %w", path, err)
	}
	return cert, nil
}
