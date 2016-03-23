package doitserver

import (
	"crypto/sha256"
	"encoding/hex"
)

func encodeID(id, key string) string {
	h := sha256.New()
	h.Write([]byte(id + key))
	return hex.EncodeToString(h.Sum(nil))
}
