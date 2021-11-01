package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

type Crypt struct {
	Key  []byte
	Salt []byte
}

// New returns a new Crypt object, with a sha256 cryptographic key and corresponding salt.
// Parameters:
// sessionkey       A sessionkey (preferably generated with PAKE2).
// salt             Salt used in generating the key. If not supplied a random salt will be generated.
func New(sessionkey []byte, salt ...[]byte) (*Crypt, error) {
	var s []byte
	if len(salt) < 1 {
		s = make([]byte, 8)
		if _, err := rand.Read(s); err != nil {
			return nil, fmt.Errorf("unable to generate random salt: %v", err)
		}
	} else {
		s = salt[0]
	}
	key := pbkdf2.Key(sessionkey, s, 100, 32, sha256.New)
	crypt := &Crypt{
		Key:  key,
		Salt: s,
	}
	return crypt, nil
}

// Encrypt encrypts the provided message using shared key and a random nonce that is appended to the message.
func (s *Crypt) Encrypt(unencrypted []byte) (encrypted []byte, err error) {
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
func (s *Crypt) Decrypt(encrypted []byte) (decrypted []byte, err error) {
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
