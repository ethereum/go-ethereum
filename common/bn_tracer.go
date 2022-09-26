package common

type CallFrameBN struct {
	Type        string        `json:"type"`
	From        string        `json:"from"`
	To          string        `json:"to,omitempty"`
	Value       string        `json:"value,omitempty"`
	Gas         string        `json:"gas"`
	GasUsed     string        `json:"gasUsed"`
	Input       string        `json:"input"`
	Output      string        `json:"output,omitempty"`
	Error       string        `json:"error,omitempty"`
	ErrorReason string        `json:"errorReason,omitempty"`
	Calls       []CallFrameBN `json:"calls,omitempty"`
	gasIn       uint64
	gasCost     uint64
	Time        string `json:"time,omitempty"`
}
