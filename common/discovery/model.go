package discovery

import (
	"encoding/json"
)

type EndpointInfo[T any] struct {
	IP       string                 `json:"ip"`
	Port     string                 `json:"port"`
	MetaData map[string]T           `json:"meta"`
}

func UnMarshal[T any](data []byte) (*EndpointInfo[T], error) {
	ed := &EndpointInfo[T]{}
	err := json.Unmarshal(data, ed)
	if err != nil {
		return nil, err
	}
	return ed, nil
}
func (edi *EndpointInfo[T]) Marshal() string {
	data, err := json.Marshal(edi)
	if err != nil {
		panic(err)
	}
	return string(data)
}