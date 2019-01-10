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

var GapInReversibleBlocksDbName = reflect.TypeOf(GapInReversibleBlocksDb{}).Name()

type GapInReversibleBlocksDb struct {
	_ReversibleBlocksException
	Elog log.Messages
}

func NewGapInReversibleBlocksDb(parent _ReversibleBlocksException, message log.Message) *GapInReversibleBlocksDb {
	return &GapInReversibleBlocksDb{parent, log.Messages{message}}
}

func (e GapInReversibleBlocksDb) Code() int64 {
	return 3180003
}

func (e GapInReversibleBlocksDb) Name() string {
	return GapInReversibleBlocksDbName
}

func (e GapInReversibleBlocksDb) What() string {
	return "Gap in the reversible blocks database"
}

func (e *GapInReversibleBlocksDb) AppendLog(l log.Message) {
	e.Elog = append(e.Elog, l)
}

func (e GapInReversibleBlocksDb) GetLog() log.Messages {
	return e.Elog
}

func (e GapInReversibleBlocksDb) TopMessage() string {
	for _, l := range e.Elog {
		if msg := l.GetMessage(); msg != "" {
			return msg
		}
	}
	return e.String()
}

func (e GapInReversibleBlocksDb) DetailMessage() string {
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

func (e GapInReversibleBlocksDb) String() string {
	return e.DetailMessage()
}

func (e GapInReversibleBlocksDb) MarshalJSON() ([]byte, error) {
	type Exception struct {
		Code int64  `json:"code"`
		Name string `json:"name"`
		What string `json:"what"`
	}

	except := Exception{
		Code: 3180003,
		Name: GapInReversibleBlocksDbName,
		What: "Gap in the reversible blocks database",
	}

	return json.Marshal(except)
}

func (e GapInReversibleBlocksDb) Callback(f interface{}) bool {
	switch callback := f.(type) {
	case func(*GapInReversibleBlocksDb):
		callback(&e)
		return true
	case func(GapInReversibleBlocksDb):
		callback(e)
		return true
	default:
		return false
	}
}
