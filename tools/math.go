package tools

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	math_rand "math/rand"
)

func RandomSeed() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("failed to seed math/rand")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}
