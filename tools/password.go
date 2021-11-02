package tools

import (
	"crypto/sha256"
	"fmt"
	"math/rand"

	"www.github.com/ZinoKader/portal/data"
	"www.github.com/ZinoKader/portal/models"
)

const PasswordWordLength = 3

// GeneratePassword generates a random password prefixed with the supplied id.
func GeneratePassword(id int) models.Password {
	var words []string
	hitlistSize := len(data.SpaceWordList)

	// generate three unique words
	for len(words) != PasswordWordLength {
		candidateWord := data.SpaceWordList[rand.Intn(hitlistSize)]
		if !Contains(words, candidateWord) {
			words = append(words, candidateWord)
		}
	}
	password := formatPassword(id, words)
	return models.Password(password)
}

func formatPassword(prefixIndex int, words []string) string {
	return fmt.Sprintf("%d-%s-%s-%s", prefixIndex, words[0], words[1], words[2])
}

func HashPassword(password models.Password) string {
	h := sha256.New()
	h.Write([]byte(password))
	return string(h.Sum(nil))
}
