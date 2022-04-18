package crypt_test

import (
	"testing"

	"github.com/ZinoKader/portal/pkg/crypt"
	"github.com/schollz/pake/v3"
	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	msg := []byte("A frog walks into a bank...")
	weakkey := []byte("Normie")
	t.Run("Encryption", func(t *testing.T) {

		// generate crypt struct
		c1, err := crypt.New(weakkey)
		assert.NoError(t, err)

		// encrypt decrypt same struct
		enc, err := c1.Encrypt(msg)
		assert.NoError(t, err)
		dec, err := c1.Decrypt(enc)
		assert.NoError(t, err)
		assert.Equal(t, dec, msg)

		// decrypt using new struct (same salt)
		c2, err := crypt.New(weakkey, c1.Salt)
		assert.NoError(t, err)
		dec, err = c2.Decrypt(enc)
		assert.NoError(t, err)
		assert.Equal(t, dec, msg)
	})

	t.Run("PAKE2 + Encryption", func(t *testing.T) {
		// initialize PAKE2 curves
		A, err := pake.InitCurve(weakkey, 0, "p256")
		assert.NoError(t, err)
		B, err := pake.InitCurve(weakkey, 1, "p256")
		assert.NoError(t, err)

		// send A to B
		err = B.Update(A.Bytes())
		assert.NoError(t, err)

		// send B to A
		err = A.Update(B.Bytes())
		assert.NoError(t, err)

		//Generate sessionkey.
		kA, err := A.SessionKey()
		assert.NoError(t, err)
		kB, err := B.SessionKey()
		assert.NoError(t, err)

		assert.Equal(t, kA, kB)

		// encrypt and decrypt
		cA, _ := crypt.New(kA)
		cB, _ := crypt.New(kB, cA.Salt)

		enc, _ := cA.Encrypt(msg)
		dec, _ := cB.Decrypt(enc)

		assert.Equal(t, dec, msg)
	})
}
