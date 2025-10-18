package argon2d

import (
"encoding/hex"
"testing"
)

func TestSaltValue(t *testing.T) {
salt := []byte("RandomX\x03")
t.Logf("Salt: %q", salt)
t.Logf("Salt hex: %s", hex.EncodeToString(salt))
t.Logf("Salt length: %d", len(salt))

// Expected: "RandomX\x03" = 7 letters + 1 byte = 8 bytes
if len(salt) != 8 {
t.Errorf("Salt length is %d, expected 8", len(salt))
}
}
