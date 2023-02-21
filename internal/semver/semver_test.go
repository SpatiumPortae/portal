package semver_test

import (
	"testing"

	"github.com/SpatiumPortae/portal/internal/semver"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			s := "v0.0.1"
			ver, err := semver.Parse(s)
			assert.Nil(t, err)
			assert.Equal(t, s, ver.String())
		})
		t.Run("double digits", func(t *testing.T) {
			s := "v10.24.30"
			ver, err := semver.Parse(s)
			assert.Nil(t, err)
			assert.Equal(t, s, ver.String())
		})
	})
	t.Run("negative", func(t *testing.T) {
		t.Run("no leading v", func(t *testing.T) {
			s := "0.0.1"
			_, err := semver.Parse(s)
			assert.Equal(t, semver.ErrParse, err)
		})
		t.Run("major leading 0", func(t *testing.T) {
			s := "v01.0.1"
			_, err := semver.Parse(s)
			assert.Equal(t, semver.ErrParse, err)
		})
		t.Run("minor leading 0", func(t *testing.T) {
			s := "v0.01.1"
			_, err := semver.Parse(s)
			assert.Equal(t, semver.ErrParse, err)
		})
		t.Run("patch leading 0", func(t *testing.T) {
			s := "v0.1.01"
			_, err := semver.Parse(s)
			assert.Equal(t, semver.ErrParse, err)
		})
	})
}

func TestCompare(t *testing.T) {
	sv, err := semver.Parse("v1.1.1")
	assert.Nil(t, err)
	t.Run("major", func(t *testing.T) {
		t.Run("oracle larger", func(t *testing.T) {
			oracle, err := semver.Parse("v2.0.0")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareOldMajor, sv.Compare(oracle))
		})
		t.Run("oracle less", func(t *testing.T) {
			oracle, err := semver.Parse("v0.0.0")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareNewMajor, sv.Compare(oracle))
		})
	})
	t.Run("minor", func(t *testing.T) {
		t.Run("oracle larger", func(t *testing.T) {
			oracle, err := semver.Parse("v1.2.0")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareOldMinor, sv.Compare(oracle))
		})
		t.Run("oracle less", func(t *testing.T) {
			oracle, err := semver.Parse("v1.0.0")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareNewMinor, sv.Compare(oracle))
		})
	})
	t.Run("patch", func(t *testing.T) {
		t.Run("oracle larger", func(t *testing.T) {
			oracle, err := semver.Parse("v1.1.2")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareOldPatch, sv.Compare(oracle))
		})
		t.Run("oracle less", func(t *testing.T) {
			oracle, err := semver.Parse("v1.1.0")
			assert.Nil(t, err)
			assert.Equal(t, semver.CompareNewPatch, sv.Compare(oracle))
		})
	})
	t.Run("equal", func(t *testing.T) {
		oracle, err := semver.Parse("v1.1.1")
		assert.Nil(t, err)
		assert.Equal(t, semver.CompareEqual, sv.Compare(oracle))
	})
}
