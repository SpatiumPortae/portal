package password

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"

	"github.com/SpatiumPortae/portal/data"
	"golang.org/x/exp/slices"
)

const passwordWordLength = 3

// GeneratePassword generates a random password prefixed with the supplied id.
func Generate(id int) string {
	var words []string
	hitlistSize := len(data.SpaceWordList)

	// generate three unique words
	for len(words) != passwordWordLength {
		candidateWord := data.SpaceWordList[rand.Intn(hitlistSize)]
		if !slices.Contains(words, candidateWord) {
			words = append(words, candidateWord)
		}
	}
	return formatPassword(id, words)
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
