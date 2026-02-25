package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func hashRequest(payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
