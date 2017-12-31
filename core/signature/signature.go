package signature

import (
	"ELAClient/common"
	"ELAClient/core/contract/program"
	"ELAClient/crypto"
	"bytes"
	"io"
	"fmt"
	"errors"
)

//Signable describe the data need be signed.
type Signable interface {
	//Get the the Signable's program hashes
	GetProgramHashes() ([]*common.Uint160, error)

	SetPrograms([]*program.Program)

	GetPrograms() []*program.Program

	//TODO: add SerializeUnsigned
	SerializeUnsigned(io.Writer) error
}

func SignBySigner(data Signable, signer Signer) ([]byte, error) {
	fmt.Println()
	//fmt.Println("data",data)
	rtx, err := crypto.Sign(signer.PrivKey(), GetHashData(data))

	if err != nil {
		return nil, errors.New("[Signature],SignBySigner failed.")
	}
	return rtx, nil
}

func GetHashData(data Signable) []byte {
	b_buf := new(bytes.Buffer)
	data.SerializeUnsigned(b_buf)
	return b_buf.Bytes()
}
