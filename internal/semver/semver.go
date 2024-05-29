package semver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const pattern = `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`

var ErrParse = errors.New("could not parse provided string into semantic version")

type Comparison int

const (
	CompareEqual Comparison = iota
	CompareOldMajor
	CompareNewMajor
	CompareOldMinor
	CompareNewMinor
	CompareOldPatch
	CompareNewPatch
)

type Version struct {
	Major int `json:"major,omitempty"`
	Minor int `json:"minor,omitempty"`
	Patch int `json:"patch,omitempty"`
}

// Parse parses the the provided string into a semver representation.
func Parse(s string) (Version, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Version{}, fmt.Errorf("compiling regex: %w", err)
	}
	if !re.MatchString(s) {
		return Version{}, ErrParse
	}
	split := strings.Split(s[1:], ".")
	ver := Version{}
	ver.Major, err = strconv.Atoi(split[0])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Major to int: %w", err)
	}
	ver.Minor, err = strconv.Atoi(split[1])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Minor to int: %w", err)
	}
	ver.Patch, err = strconv.Atoi(split[2])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Patch to int: %w", err)
	}

	return ver, nil
}

// String returns a string representation of the semver.
func (sv Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
}

// Compare compares the semver against the provided oracle statement.
func (sv Version) Compare(oracle Version) Comparison {
	switch {
	case sv.Major < oracle.Major:
		return CompareOldMajor
	case sv.Major > oracle.Major:
		return CompareNewMajor
	case sv.Minor < oracle.Minor:
		return CompareOldMinor
	case sv.Minor > oracle.Minor:
		return CompareNewMinor
	case sv.Patch < oracle.Patch:
		return CompareOldPatch
	case sv.Patch > oracle.Patch:
		return CompareNewPatch
	default:
		return CompareEqual
	}
}

func GetRendezvousVersion(ctx context.Context, addr string) (Version, error) {
	r, err := http.Get(fmt.Sprintf("http://%s/version", addr))
	if err != nil {
		return Version{}, fmt.Errorf("fetching the latest version from relay: %w", err)
	}
	defer r.Body.Close()
	var version Version
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		return Version{}, fmt.Errorf("decoding version response from relay: %w", err)
	}
	return version, nil
}
