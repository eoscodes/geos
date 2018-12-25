// Code generated by gotemplate. DO NOT EDIT.

package exception

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/eosspark/eos-go/log"
)

// template type Exception(PARENT,CODE,WHAT)

var TableAccessViolationName = reflect.TypeOf(TableAccessViolation{}).Name()

type TableAccessViolation struct {
	_ContractException
	Elog log.Messages
}

func NewTableAccessViolation(parent _ContractException, message log.Message) *TableAccessViolation {
	return &TableAccessViolation{parent, log.Messages{message}}
}

func (e TableAccessViolation) Code() int64 {
	return 3160002
}

func (e TableAccessViolation) Name() string {
	return TableAccessViolationName
}

func (e TableAccessViolation) What() string {
	return "Table access violation"
}

func (e *TableAccessViolation) AppendLog(l log.Message) {
	e.Elog = append(e.Elog, l)
}

func (e TableAccessViolation) GetLog() log.Messages {
	return e.Elog
}

func (e TableAccessViolation) TopMessage() string {
	for _, l := range e.Elog {
		if msg := l.GetMessage(); msg != "" {
			return msg
		}
	}
	return e.String()
}

func (e TableAccessViolation) DetailMessage() string {
	var buffer bytes.Buffer
	buffer.WriteString(strconv.Itoa(int(e.Code())))
	buffer.WriteString(" ")
	buffer.WriteString(e.Name())
	buffer.WriteString(": ")
	buffer.WriteString(e.What())
	buffer.WriteString("\n")
	for _, l := range e.Elog {
		buffer.WriteString("[")
		buffer.WriteString(l.GetMessage())
		buffer.WriteString("]")
		buffer.WriteString("\n")
		buffer.WriteString(l.GetContext().String())
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (e TableAccessViolation) String() string {
	return e.DetailMessage()
}

func (e TableAccessViolation) MarshalJSON() ([]byte, error) {
	type Exception struct {
		Code int64  `json:"code"`
		Name string `json:"name"`
		What string `json:"what"`
	}

	except := Exception{
		Code: 3160002,
		Name: TableAccessViolationName,
		What: "Table access violation",
	}

	return json.Marshal(except)
}

func (e TableAccessViolation) Callback(f interface{}) bool {
	switch callback := f.(type) {
	case func(*TableAccessViolation):
		callback(&e)
		return true
	case func(TableAccessViolation):
		callback(e)
		return true
	default:
		return false
	}
}