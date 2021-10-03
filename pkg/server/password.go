package server

import (
	"fmt"
	"math/rand"
	"sync"

	"www.github.com/ZinoKader/portal/data"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/tools"
)

const PasswordWordLength = 3

// generate an untaken password for the target map
func GeneratePassword(target *sync.Map) models.Password {
	var words []string
	hitlistSize := len(data.SpaceWordList)

	// generate three unique words
	for len(words) != PasswordWordLength {
		candidateWord := data.SpaceWordList[rand.Intn(hitlistSize)]
		if !tools.Contains(words, candidateWord) {
			words = append(words, candidateWord)
		}
	}

	var password string

	// find a non-colliding prefix to prepend
	prefixIndex := 1
	for {
		password = formatPassword(prefixIndex, words)
		_, isTaken := target.Load(password)
		if !isTaken {
			break
		}
		prefixIndex++
	}

	return models.Password(password)
}

func formatPassword(prefixIndex int, words []string) string {
	return fmt.Sprintf("%d-%s-%s-%s", prefixIndex, words[0], words[1], words[2])
}
