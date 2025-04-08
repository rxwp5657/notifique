package hash

import (
	"crypto/md5"
	"encoding/hex"
)

func GetMd5Hash(data string) string {
	hashed := md5.Sum([]byte(data))
	return hex.EncodeToString(hashed[:])
}
