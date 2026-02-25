package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// hashRequest returns a deterministic digest for idempotency request matching.
func hashRequest(payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
