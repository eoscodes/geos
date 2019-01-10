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

var MissingChainApiPluginExceptionName = reflect.TypeOf(MissingChainApiPluginException{}).Name()

type MissingChainApiPluginException struct {
	_PluginException
	Elog log.Messages
}

func NewMissingChainApiPluginException(parent _PluginException, message log.Message) *MissingChainApiPluginException {
	return &MissingChainApiPluginException{parent, log.Messages{message}}
}

func (e MissingChainApiPluginException) Code() int64 {
	return 3110001
}

func (e MissingChainApiPluginException) Name() string {
	return MissingChainApiPluginExceptionName
}

func (e MissingChainApiPluginException) What() string {
	return "Missing Chain API Plugin"
}

func (e *MissingChainApiPluginException) AppendLog(l log.Message) {
	e.Elog = append(e.Elog, l)
}

func (e MissingChainApiPluginException) GetLog() log.Messages {
	return e.Elog
}

func (e MissingChainApiPluginException) TopMessage() string {
	for _, l := range e.Elog {
		if msg := l.GetMessage(); msg != "" {
			return msg
		}
	}
	return e.String()
}

func (e MissingChainApiPluginException) DetailMessage() string {
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

func (e MissingChainApiPluginException) String() string {
	return e.DetailMessage()
}

func (e MissingChainApiPluginException) MarshalJSON() ([]byte, error) {
	type Exception struct {
		Code int64  `json:"code"`
		Name string `json:"name"`
		What string `json:"what"`
	}

	except := Exception{
		Code: 3110001,
		Name: MissingChainApiPluginExceptionName,
		What: "Missing Chain API Plugin",
	}

	return json.Marshal(except)
}

func (e MissingChainApiPluginException) Callback(f interface{}) bool {
	switch callback := f.(type) {
	case func(*MissingChainApiPluginException):
		callback(&e)
		return true
	case func(MissingChainApiPluginException):
		callback(e)
		return true
	default:
		return false
	}
}
