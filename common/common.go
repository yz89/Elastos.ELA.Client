package common

import (
	"io"
	"os"
	"bytes"
	"errors"
	"encoding/hex"
	"crypto/sha256"
	"encoding/binary"

	"golang.org/x/crypto/ripemd160"
)

const (
	SignTypeSingle = 1
	SignTypeMulti  = 2
)

func ToScriptHash(code []byte, signType int) (*Uint160, error) {
	temp := sha256.Sum256(code)
	md := ripemd160.New()
	io.WriteString(md, string(temp[:]))
	f := md.Sum(nil)

	if signType == SignTypeSingle {
		f = append([]byte{33}, f...)
	} else if signType == SignTypeMulti {
		f = append([]byte{18}, f...)
	}

	hash, err := Uint160ParseFromBytes(f)
	if err != nil {
		return nil, errors.New("[Common] ,ToScriptHash error")
	}
	return hash, nil
}

func BytesToInt16(b []byte) int16 {
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int16
	binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	return int16(tmp)
}

func IsEqualBytes(b1 []byte, b2 []byte) bool {
	len1 := len(b1)
	len2 := len(b2)
	if len1 != len2 {
		return false
	}

	for i := 0; i < len1; i++ {
		if b1[i] != b2[i] {
			return false
		}
	}
	return true
}

func BytesToHexString(data []byte) string {
	return hex.EncodeToString(data)
}

func HexStringToBytes(value string) ([]byte, error) {
	return hex.DecodeString(value)
}

func BytesReverse(u []byte) []byte {
	for i, j := 0, len(u)-1; i < j; i, j = i+1, j-1 {
		u[i], u[j] = u[j], u[i]
	}
	return u
}

func HexStringToBytesReverse(value string) ([]byte, error) {
	u, err := hex.DecodeString(value)
	if err != nil {
		return u, err
	}
	return BytesReverse(u), err
}

func ClearBytes(arr []byte, len int) {
	for i := 0; i < len; i++ {
		arr[i] = 0
	}
}

func FileExisted(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
