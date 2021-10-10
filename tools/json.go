package tools

import (
	"encoding/json"
	"fmt"
)

func PrettyJSONFormat(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func DecodePayload(payload interface{}, target interface{}) (err error) {
	err = json.Unmarshal(payload.([]byte), &target)
	if err != nil {
		return fmt.Errorf("faulty payload\n expected: %v \n%e", PrettyJSONFormat(target), err)
	}
	return nil
}
