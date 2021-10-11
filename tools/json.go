package tools

import (
	"encoding/json"
	"fmt"
)

func prettyJSONFormat(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}

func DecodePayload(payload interface{}, target interface{}) (err error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload into bytes:%e", err)
	}
	err = json.Unmarshal(bytes, &target)
	if err != nil {
		return fmt.Errorf("faulty payload format\nexpected format:\n%s\ngot:\n%s",
			prettyJSONFormat(&target),
			prettyJSONFormat(&payload))
	}
	return nil
}
