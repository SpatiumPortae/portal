package tools

import (
	"encoding/json"
	"fmt"
)

func prettyJSONFormat(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func DecodePayload(payload interface{}, target interface{}) (err error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload into bytes:%e", err)
	}
	err = json.Unmarshal(bytes, &target)
	if err != nil {
		return fmt.Errorf("faulty payload\nexpected: %v\n%e", prettyJSONFormat(&target), err)
	}
	return nil
}
