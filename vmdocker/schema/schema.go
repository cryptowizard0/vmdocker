package schema

import (
	vmmSchema "github.com/hymatrix/hymx/vmm/schema"
	goarSchema "github.com/permadao/goar/schema"
)

type SpawnRequest struct {
	Pid    string           `json:"pid"`
	Owner  string           `json:"owner"`
	CuAddr string           `json:"cu_addr"`
	Data   []byte           `json:"data"`
	Tags   []goarSchema.Tag `json:"tags"`
	Evn    vmmSchema.Env    `json:"env"`
}

type ApplyRequest struct {
	Meta   vmmSchema.Meta    `json:"meta"`
	From   string            `json:"from"`
	Params map[string]string `json:"params"`
}

type OutboxResponse struct {
	Result string `json:"result"`
	Status string `json:"status"`
}
