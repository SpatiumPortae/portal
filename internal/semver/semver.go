package semver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const pattern = `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`

var ParseError = errors.New("provided string cannot be parsed into semver")

type Version struct {
	major, minor, patch int
}

// Parse parses the the provided string into a semver representation.
func Parse(s string) (Version, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Version{}, fmt.Errorf("compiling regex: %w", err)
	}
	if !re.MatchString(s) {
		return Version{}, ParseError
	}
	split := strings.Split(s[1:], ".")
	ver := Version{}
	ver.major, err = strconv.Atoi(split[0])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Major to int: %w", err)
	}
	ver.minor, err = strconv.Atoi(split[1])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Minor to int: %w", err)
	}
	ver.patch, err = strconv.Atoi(split[2])
	if err != nil {
		return Version{}, fmt.Errorf("parsing Patch to int: %w", err)
	}

	return ver, nil
}

// String returns a string representation of the semver.
func (sv Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.major, sv.minor, sv.patch)
}

// Compare compares the semver against the provided oracle statement.
// Return -1 if the semver is less than the oracle statement, 1 if
// the oracle statement is larger than the semver and 0 if they are equal.
func (sv Version) Compare(oracle Version) int {
	if sv.major < oracle.major {
		return -1
	}
	if sv.major > oracle.major {
		return 1
	}
	if sv.minor < oracle.minor {
		return -1
	}
	if sv.minor > oracle.minor {
		return 1
	}
	if sv.patch < oracle.patch {
		return -1
	}
	if sv.patch > oracle.patch {
		return 1
	}
	return 0
}

func (sv Version) Major() int {
	return sv.major
}

func (sv Version) Minor() int {
	return sv.minor
}

func (sv Version) Patch() int {
	return sv.patch
}

func GetPortalLatest() (Version, error) {
	r, err := http.Get("https://api.github.com/repos/SpatiumPortae/portal/tags")
	if err != nil {
		return Version{}, fmt.Errorf("fetching the latest tag from github: %w", err)
	}
	type tag struct {
		Name string `json:"name"`
	}
	var tags []tag
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		return Version{}, fmt.Errorf("decoding response from github: %w", err)
	}
	if len(tags) < 1 {
		return Version{}, fmt.Errorf("no tags returned from github: %w", err)
	}
	log.Println(tags)
	vers := make([]Version, len(tags))
	for i := range tags {
		v, err := Parse(tags[i].Name)
		if err != nil {
			return Version{}, fmt.Errorf("unable to parse tag to semver: %w", err)
		}
		vers[i] = v
	}
	return vers[0], nil
}
