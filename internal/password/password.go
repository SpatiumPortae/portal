package password

import (
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	math_rand "math/rand"
	"regexp"

	"github.com/SpatiumPortae/portal/data"
	"golang.org/x/exp/slices"
)

const Length = 3

// GeneratePassword generates a random password prefixed with the supplied id.
func Generate(id int) (string, error) {
	var words []string
	hitlistSize := len(data.SpaceWordList)

	rng, err := random()
	if err != nil {
		return "", fmt.Errorf("creating rng: %w", err)
	}

	// generate three unique words
	for len(words) != Length {
		candidateWord := data.SpaceWordList[rng.Intn(hitlistSize)]
		if !slices.Contains(words, candidateWord) {
			words = append(words, candidateWord)
		}
	}
	return formatPassword(id, words), nil
}

func IsValid(passStr string) bool {
	re := regexp.MustCompile(`^\d+-[a-z]+-[a-z]+-[a-z]+$`)
	return re.MatchString(passStr)
}

func Hashed(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

func formatPassword(prefixIndex int, words []string) string {
	return fmt.Sprintf("%d-%s-%s-%s", prefixIndex, words[0], words[1], words[2])
}

func random() (*math_rand.Rand, error) {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		return nil, err
	}
	return math_rand.New(math_rand.NewSource(int64(binary.LittleEndian.Uint64(b[:])))), nil
}
