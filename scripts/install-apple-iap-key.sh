#!/usr/bin/env bash
# Validate an App Store Connect In-App Purchase key and install it into
# backend/.env as a single-line base64 value.
#
#   scripts/install-apple-iap-key.sh ~/Downloads/AuthKey_ABCDE12345.p8 <ISSUER_ID>
#
# The Key ID is read from the filename. Nothing is printed except the key id,
# the issuer id and the key's curve — the private key itself never reaches the
# terminal. Run it again to rotate the key.
set -euo pipefail

P8="${1:-}"
ISSUER="${2:-}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT/backend/.env"

die() { printf 'error: %s\n' "$1" >&2; exit 1; }

[ -n "$P8" ] || die "usage: $0 <AuthKey_XXXXXXXXXX.p8> <ISSUER_ID>"
[ -n "$ISSUER" ] || die "the issuer id is required (App Store Connect > Users and Access > Integrations)"
[ -f "$P8" ] || die "no such file: $P8"
[ -f "$ENV_FILE" ] || die "no such file: $ENV_FILE"

# App Store Connect names an In-App Purchase key SubscriptionKey_<KEYID>.p8 and
# an App Store Connect API key AuthKey_<KEYID>.p8. Accept either spelling; the
# key id is always 10 alphanumeric characters.
BASENAME="$(basename "$P8")"
KEY_ID="${BASENAME%.p8}"
KEY_ID="${KEY_ID##*_}"
if ! printf '%s' "$KEY_ID" | grep -qE '^[A-Z0-9]{10}$'; then
    die "could not read a key id from '$BASENAME'; expected SubscriptionKey_<10 chars>.p8 or AuthKey_<10 chars>.p8"
fi
if ! printf '%s' "$ISSUER" | grep -qiE '^[0-9a-f-]{36}$'; then
    die "the issuer id should be a UUID, got '$ISSUER'"
fi

# Fail before touching .env if the file is not the key Apple should have given
# us: a PKCS#8 EC key on P-256, which is what ES256 signing requires.
command -v openssl >/dev/null || die "openssl is required"
# `|| true` matters: under `set -euo pipefail` a failing openssl would abort the
# script before the diagnostic below ever ran.
KEY_INFO="$(openssl pkey -in "$P8" -noout -text 2>/dev/null || true)"
[ -n "$KEY_INFO" ] || die "$BASENAME is not a readable private key (is it the right download?)"

CURVE="$(printf '%s' "$KEY_INFO" | sed -n 's/^ *ASN1 OID: *//p' | head -1)"
[ -n "$CURVE" ] || die "$BASENAME is not an EC key; ES256 signing needs the EC key Apple issues for In-App Purchase"
[ "$CURVE" = "prime256v1" ] || die "expected a P-256 (prime256v1) key for ES256, got '$CURVE'"

B64="$(base64 < "$P8" | tr -d '\n')"

# Rewrite the four keys idempotently, leaving every other line untouched.
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
grep -vE '^(APPLE_IAP_ISSUER_ID|APPLE_IAP_KEY_ID|APPLE_IAP_PRIVATE_KEY|APPLE_BUNDLE_ID)=' "$ENV_FILE" > "$TMP"
{
    printf '\n# Apple In-App Purchase — installed by scripts/install-apple-iap-key.sh\n'
    printf 'APPLE_BUNDLE_ID=org.mediarise.tarot\n'
    printf 'APPLE_IAP_ISSUER_ID=%s\n' "$ISSUER"
    printf 'APPLE_IAP_KEY_ID=%s\n' "$KEY_ID"
    printf 'APPLE_IAP_PRIVATE_KEY=%s\n' "$B64"
} >> "$TMP"

# Keep .env owner-only; it now holds a signing key.
cp "$TMP" "$ENV_FILE"
chmod 600 "$ENV_FILE"

printf 'installed into backend/.env\n'
printf '  APPLE_IAP_KEY_ID   %s\n' "$KEY_ID"
printf '  APPLE_IAP_ISSUER_ID %s\n' "$ISSUER"
printf '  key                 EC %s, %s bytes of base64\n' "$CURVE" "${#B64}"
printf '\nStore the .p8 somewhere safe — Apple lets you download it only once.\n'
printf 'Then set the same four variables on the server (they are not committed).\n'
