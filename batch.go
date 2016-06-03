package objects

import "encoding/json"

type batch struct {
	Collection string          `json:"collection"`
	WriteKey   string          `json:"write_key"`
	Objects    json.RawMessage `json:"objects"`
}
