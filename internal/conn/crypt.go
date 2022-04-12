package conn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

type crypt struct {
	Key []byte
}

// New returns a new crypt object, with a sha256 cryptographic key derived from specified,
//  sessionkey and the specified salt.
func New(sessionkey []byte, salt []byte) crypt {
	key := pbkdf2.Key(sessionkey, salt, 100, 32, sha256.New)
	crypt := crypt{
		Key: key,
	}
	return crypt
}

// Encrypt encrypts the provided message using shared key and a random nonce that is appended to the message.
func (s *crypt) Encrypt(unencrypted []byte) (encrypted []byte, err error) {
	block, err := aes.NewCipher(s.Key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("unable to generate random nonce: %v", err)
	}

	aescgm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	encrypted = aescgm.Seal(nil, nonce, unencrypted, nil)
	encrypted = append(nonce, encrypted...)
	return encrypted, nil
}

// Decrypt decrypts the provided message with the the shared key.
func (s *crypt) Decrypt(encrypted []byte) (decrypted []byte, err error) {
	block, err := aes.NewCipher(s.Key)
	if err != nil {
		return nil, err
	}

	aescgm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	decrypted, err = aescgm.Open(nil, encrypted[:12], encrypted[12:], nil)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}
