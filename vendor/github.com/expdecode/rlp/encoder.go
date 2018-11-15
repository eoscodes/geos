package rlp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Pack interface {
	EncodeRLP(io.Writer) error
}

// --------------------------------------------------------------
// Encoder implements the EOS packing, similar to FC_BUFFER
// --------------------------------------------------------------
type Encoder struct {
	output io.Writer
	Order  binary.ByteOrder
	count  int
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		output: w,
		Order:  binary.LittleEndian,
		count:  0,
	}
}

func Encode(w io.Writer, val interface{}) error {
	encoder := NewEncoder(w)
	err := encoder.encode(val)
	if err != nil {
		return err
	}
	return nil
}

// EncodeToBytes returns the RLP encoding of val.
// Please see the documentation of Encode for the encoding rules.
func EncodeToBytes(val interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := NewEncoder(buf).encode(val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func EncodeToReader(val interface{}) (size int, r io.Reader, err error) {
	buf := new(bytes.Buffer)
	if err := NewEncoder(buf).encode(val); err != nil {
		return 0, nil, err
	}
	return buf.Len(), bytes.NewReader(buf.Bytes()), nil
}

func EocodeSize(val interface{}) (int, error) {
	buffer, err := EncodeToBytes(val)
	if err != nil {
		return 0, err
	}
	return len(buffer), nil
}

func (e *Encoder) encode(v interface{}) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	t := rv.Type()
	switch t.Kind() {
	case reflect.Array:
		l := t.Len()
		println(fmt.Sprintf("Encode: array [%T] of length: %d", v, l))

		for i := 0; i < l; i++ {
			if err = e.encode(rv.Index(i).Interface()); err != nil {
				return
			}
		}
	case reflect.Slice:
		l := rv.Len()
		if err = e.writeUVarInt(l); err != nil {
			return
		}
		println(fmt.Sprintf("Encode: slice [%T] of length: %d", v, l))

		for i := 0; i < l; i++ {
			if err = e.encode(rv.Index(i).Interface()); err != nil {
				return
			}
		}
	case reflect.Struct:
		l := rv.NumField()
		println(fmt.Sprintf("Encode: struct [%T] with %d field.", v, l))
		for i := 0; i < l; i++ {
			field := t.Field(i)
			println(fmt.Sprintf("field -> %s", field.Name))
			tag := field.Tag.Get("eos")
			if tag == "-" {
				continue
			}

			if v := rv.Field(i); t.Field(i).Name != "_" {
				if v.CanInterface() {
					isPresent := true
					if tag == "optional" {
						isPresent = !v.IsNil()
						e.writeBool(isPresent)
					}

					if isPresent {
						if err = e.encode(v.Interface()); err != nil {
							return
						}
					}
				}
			}

		}
		// 		type OptionalProducerSchedule struct {
		// 	ProducerSchedule
		// }
		// 	NewProducers     *OptionalProducerSchedule `json:"new_producers" eos:"optional"`
	case reflect.Map:
		l := rv.Len()
		if err = e.writeUVarInt(l); err != nil {
			return
		}
		println(fmt.Sprintf("Map [%T] of length: %d", v, l))
		for _, key := range rv.MapKeys() {
			value := rv.MapIndex(key)
			if err = e.encode(key.Interface()); err != nil {
				return err
			}
			if err = e.encode(value.Interface()); err != nil {
				return err
			}
		}

	case reflect.String:
		return e.writeString(rv.String())
	case reflect.Bool:
		return e.writeBool(rv.Bool())
	case reflect.Int8:
		return e.writeByte(byte(rv.Int()))
	case reflect.Int16:
		return e.writeInt16(int16(rv.Int()))
	case reflect.Int32:
		return e.writeInt32(int32(rv.Int()))
	case reflect.Int:
		return e.writeInt32(int32(rv.Int()))
	case reflect.Int64:
		return e.writeInt64(rv.Int())
	case reflect.Uint8:
		return e.writeUint8(uint8(rv.Uint()))
	case reflect.Uint16:
		return e.writeUint16(uint16(rv.Uint()))
	case reflect.Uint32:
		return e.writeUint32(uint32(rv.Uint()))
	case reflect.Uint:
		return e.writeUint32(uint32(rv.Uint()))
	case reflect.Uint64:
		return e.writeUint64(rv.Uint())

	default:
		return errors.New("Encode: unsupported type " + t.String())
	}

	return
}

func (e *Encoder) toWriter(bytes []byte) (err error) {
	e.count += len(bytes)
	println(fmt.Sprintf("    Appending : [%s] pos [%d]", hex.EncodeToString(bytes), e.count))
	_, err = e.output.Write(bytes)
	return
}

func (e *Encoder) writeByteArray(b []byte) error {
	println(fmt.Sprintf("writing byte array of len [%d]", len(b)))
	if err := e.writeUVarInt(len(b)); err != nil {
		return err
	}
	return e.toWriter(b)
}

func (e *Encoder) writeString(s string) (err error) {
	return e.writeByteArray([]byte(s))
}

func (e *Encoder) writeUVarInt(v int) (err error) {
	buf := make([]byte, 8)
	l := binary.PutUvarint(buf, uint64(v))
	return e.toWriter(buf[:l])
}

func (e *Encoder) writeByte(b byte) (err error) {
	return e.toWriter([]byte{b})
}

func (e *Encoder) writeBool(b bool) (err error) {
	var out byte
	if b {
		out = 1
	}
	return e.writeByte(out)
}

func (e *Encoder) writeUint8(i uint8) (err error) {
	return e.toWriter([]byte{byte(i)})
}

func (e *Encoder) writeUint16(i uint16) (err error) {
	buf := make([]byte, TypeSize.UInt16)
	binary.LittleEndian.PutUint16(buf, i)
	return e.toWriter(buf)
}

func (e *Encoder) writeUint32(i uint32) (err error) {
	buf := make([]byte, TypeSize.UInt32)
	binary.LittleEndian.PutUint32(buf, i)
	return e.toWriter(buf)
}

func (e *Encoder) writeUint64(i uint64) (err error) {
	buf := make([]byte, TypeSize.UInt64)
	binary.LittleEndian.PutUint64(buf, i)
	return e.toWriter(buf)
}

func (e *Encoder) writeInt8(i int8) (err error) {
	return e.writeUint8(uint8(i))
}
func (e *Encoder) writeInt16(i int16) (err error) {
	return e.writeUint16(uint16(i))
}

func (e *Encoder) writeInt32(i int32) (err error) {
	return e.writeUint32(uint32(i))
}
func (e *Encoder) writeInt64(i int64) (err error) {
	return e.writeUint64(uint64(i))
}

func MarshalBinary(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := Encode(buf, v)
	return buf.Bytes(), err
}
