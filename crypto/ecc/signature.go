package ecc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eosspark/eos-go/crypto/btcsuite/btcd/btcec"
	"github.com/eosspark/eos-go/crypto/btcsuite/btcutil/base58"
)

// Signature represents a signature for some hash
type Signature struct {
	Curve CurveID
	// Content []byte // the Compact signature as bytes
	Content [65]byte `eos:"array"` // the Compact signature as bytes
}

func NewSignatureFromData(data []byte) (Signature, error) {
	if len(data) != 66 {
		return Signature{}, fmt.Errorf("data length of a signature should be 66, reveived %d", len(data))
	}

	var content [65]byte //TODO
	for i := range data {
		if i <= 64 {
			content[i] = data[i+1]
		}
	}

	signature := Signature{
		Curve:   CurveID(data[0]), // 1 byte
		Content: content,          // 65 bytes
	}

	//switch signature.Curve {
	//case CurveK1:
	//	signature.innerSignature = &innerK1Signature{}
	//case CurveR1:
	//	signature.innerSignature = &innerR1Signature{}
	//default:
	//	return Signature{}, fmt.Errorf("invalid curve  %q", signature.Curve)
	//}
	return signature, nil
}

// Verify checks the signature against the pubKey. `hash` is a sha256
// hash of the payload to verify.
func (s Signature) Verify(hash []byte, pubKey PublicKey) bool {
	if s.Curve != CurveK1 {
		fmt.Println("WARN: github.com/eosspark/eos-go/ecc library does not support the R1 curve yet")
		return false
	}

	// TODO: choose the S256 curve, based on s.Curve
	recoveredKey, _, err := btcec.RecoverCompact(btcec.S256(), s.Content[:], hash)
	if err != nil {
		return false
	}
	key, err := pubKey.Key()
	if err != nil {
		return false
	}
	if recoveredKey.IsEqual(key) {
		return true
	}
	return false
}

// PublicKey retrieves the public key, but requires the
// payload.. that's the way to validate the signature. Use Verify() if
// you only want to validate.
func (s Signature) PublicKey(hash []byte) (out PublicKey, err error) {
	if s.Curve != CurveK1 {
		return out, fmt.Errorf("WARN: github.com/eosspark/eos-go/ecc library does not support the R1 curve yet")
	}

	recoveredKey, _, err := btcec.RecoverCompact(btcec.S256(), s.Content[:], hash)
	if err != nil {
		return out, err
	}

	var data [33]byte
	temp := recoveredKey.SerializeCompressed()
	for i := range temp {
		data[i] = temp[i]
	}

	return PublicKey{
		Curve:   s.Curve,
		Content: data,
	}, nil
	// return PublicKey{
	// 	Curve:   s.Curve,
	// 	Content: recoveredKey.SerializeCompressed(),
	// }, nil
}

func (s Signature) String() string {
	checksum := Ripemd160checksumHashCurve(s.Content[:], s.Curve)
	buf := append(s.Content[:], checksum...)
	return "SIG_" + s.Curve.StringPrefix() + base58.Encode(buf)
	//return "SIG_" + base58.Encode(buf)
	//return base58.Encode(buf)
}

func NewSignature(fromText string) (Signature, error) {
	if !strings.HasPrefix(fromText, "SIG_") {
		return Signature{}, fmt.Errorf("signature should start with SIG_")
	}
	if len(fromText) < 8 {
		return Signature{}, fmt.Errorf("invalid signature length")
	}

	fromText = fromText[4:] // remove the `SIG_` prefix

	var curveID CurveID
	var curvePrefix = fromText[:3]
	switch curvePrefix {
	case "K1_":
		curveID = CurveK1
	case "R1_":
		curveID = CurveR1
	default:
		return Signature{}, fmt.Errorf("invalid curve prefix %q", curvePrefix)
	}
	fromText = fromText[3:] // strip curve ID

	sigbytes := base58.Decode(fromText)

	content := sigbytes[:len(sigbytes)-4]
	checksum := sigbytes[len(sigbytes)-4:]
	verifyChecksum := Ripemd160checksumHashCurve(content, curveID)
	if !bytes.Equal(verifyChecksum, checksum) {
		fmt.Printf("signature checksum failed, found %x expected %x", verifyChecksum, checksum)
		return Signature{}, fmt.Errorf("signature checksum failed, found %x expected %x", verifyChecksum, checksum)
	}
	var temp [65]byte
	for i := range content {
		temp[i] = content[i]
	}
	return Signature{Curve: curveID, Content: temp}, nil

}

func (a Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Signature) UnmarshalJSON(data []byte) (err error) {
	var s string
	err = json.Unmarshal(data, &s)
	if err != nil {
		return
	}

	*a, err = NewSignature(s)

	return
}

func NewSigNil() *Signature {
	return &Signature{Curve: CurveK1,
		Content: [65]byte{}}
}
