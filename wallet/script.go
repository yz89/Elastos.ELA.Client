package wallet

import (
	"sort"
	"bytes"
	"errors"
	"math/big"

	"ELAClient/crypto"
)

type OpCode byte

const (
	PUSH0 = 0x00
	PUSH1 = 0x51

	CHECKSIG      = 0xAC
	CHECKMULTISIG = 0xAE
)

func CreateSignatureRedeemScript(publicKey *crypto.PubKey) ([]byte, error) {
	content, err := publicKey.EncodePoint(true)
	if err != nil {
		return nil, errors.New("[Contracts],CreateSignatureRedeemScript failed.")
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(content)))
	buf.Write(content)
	buf.WriteByte(byte(CHECKSIG))
	return buf.Bytes(), nil
}

func CreateMultiSignRedeemScript(publicKeys []*crypto.PubKey) ([]byte, error) {
	M := len(publicKeys)/2 + 1

	bigM := big.NewInt(int64(M))
	opCode := OpCode(byte(PUSH1) - 1 + bigM.Bytes()[0])

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(opCode))

	//sort pubkey
	sort.Sort(crypto.PubKeySlice(publicKeys))

	for _, pubkey := range publicKeys {
		content, err := pubkey.EncodePoint(true)
		if err != nil {
			return nil, errors.New("[Contracts],CreateSignatureContract failed.")
		}
		buf.WriteByte(byte(len(content)))
		buf.Write(content)
	}

	bigKeys := big.NewInt(int64(len(publicKeys)))
	opCode = OpCode(byte(PUSH1) - 1 + bigKeys.Bytes()[0])
	buf.WriteByte(byte(opCode))
	buf.WriteByte(CHECKMULTISIG)

	return buf.Bytes(), nil
}
