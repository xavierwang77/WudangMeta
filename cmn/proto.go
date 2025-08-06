package cmn

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
)

type ReqProto struct {
	Action string `json:"action,omitempty"`

	Sets    []string            `json:"sets,omitempty"`
	OrderBy []map[string]string `json:"orderBy,omitempty"`

	//***页码从第零页开始***
	Page     int64 `json:"page,omitempty"`
	PageSize int64 `json:"pageSize,omitempty"`

	Data   json.RawMessage `json:"data,omitempty"`
	Filter json.RawMessage `json:"filter,omitempty"`
}

type ReplyProto struct {
	//Status, 0: success, others: fault
	Status int `json:"status"`

	//Msg, Action result describe by literal
	Msg string `json:"msg,omitempty"`

	//Data, operand
	Data types.JSONText `json:"data,omitempty"`

	// RowCount, just row count
	RowCount int64 `json:"rowCount,omitempty"`

	//API, call target
	API string `json:"API,omitempty"`

	//Method, using http method
	Method string `json:"method,omitempty"`

	//SN, call order
	SN int `json:"SN,omitempty"`
}
