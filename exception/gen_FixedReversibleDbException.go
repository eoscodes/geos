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

var FixedReversibleDbExceptionName = reflect.TypeOf(FixedReversibleDbException{}).Name()

type FixedReversibleDbException struct {
	_MiscException
	Elog log.Messages
}

func NewFixedReversibleDbException(parent _MiscException, message log.Message) *FixedReversibleDbException {
	return &FixedReversibleDbException{parent, log.Messages{message}}
}

func (e FixedReversibleDbException) Code() int64 {
	return 3100004
}

func (e FixedReversibleDbException) Name() string {
	return FixedReversibleDbExceptionName
}

func (e FixedReversibleDbException) What() string {
	return "Corrupted reversible block database was fixed"
}

func (e *FixedReversibleDbException) AppendLog(l log.Message) {
	e.Elog = append(e.Elog, l)
}

func (e FixedReversibleDbException) GetLog() log.Messages {
	return e.Elog
}

func (e FixedReversibleDbException) TopMessage() string {
	for _, l := range e.Elog {
		if msg := l.GetMessage(); msg != "" {
			return msg
		}
	}
	return e.String()
}

func (e FixedReversibleDbException) DetailMessage() string {
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
		buffer.WriteString("] ")
		buffer.WriteString(l.GetContext().String())
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (e FixedReversibleDbException) String() string {
	return e.DetailMessage()
}

func (e FixedReversibleDbException) MarshalJSON() ([]byte, error) {
	type Exception struct {
		Code int64  `json:"code"`
		Name string `json:"name"`
		What string `json:"what"`
	}

	except := Exception{
		Code: 3100004,
		Name: FixedReversibleDbExceptionName,
		What: "Corrupted reversible block database was fixed",
	}

	return json.Marshal(except)
}

func (e FixedReversibleDbException) Callback(f interface{}) bool {
	switch callback := f.(type) {
	case func(*FixedReversibleDbException):
		callback(&e)
		return true
	case func(FixedReversibleDbException):
		callback(e)
		return true
	default:
		return false
	}
}
