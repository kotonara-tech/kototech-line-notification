package notice

import (
	"crypto/sha1" //nolint:gosec // UUID v5 の仕様で SHA-1 を使う(暗号用途ではない)
	"encoding/hex"
	"fmt"
	"strings"
)

// uuidV5 は、名前空間と名前から決まる UUID(バージョン 5)を返す。
// 標準ライブラリだけで済ませるため自前で実装している(RFC 4122)。
func uuidV5(ns [16]byte, name string) string {
	h := sha1.New() //nolint:gosec // 同上
	h.Write(ns[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)

	var u [16]byte
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x50 // バージョン 5
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 のバリアント

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// mustParseUUID は UUID 文字列を 16 バイトにする。不正ならプログラムの誤りなので panic する。
func mustParseUUID(s string) [16]byte {
	raw, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	if err != nil || len(raw) != 16 {
		panic("不正な UUID: " + s)
	}
	var u [16]byte
	copy(u[:], raw)
	return u
}
