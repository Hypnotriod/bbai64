package command

import "encoding/json"

type CommandType string

const (
	SetServoValues CommandType = "setServoValues"
)

type Command struct {
	Type   CommandType `json:"type"`
	Values []float64   `json:"values,omitempty"`
}

func Unmarshal(raw []byte) (cmd *Command, err error) {
	cmd = &Command{}
	err = json.Unmarshal(raw, cmd)
	return
}
