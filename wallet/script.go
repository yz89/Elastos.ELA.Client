package wallet

import (
	"sort"
	"bytes"
	"errors"
	"math/big"

	"ELAClient/crypto"
	tx "ELAClient/core/transaction"
)

type OpCode byte

func CreateStandardRedeemScript(publicKey *crypto.PublicKey) ([]byte, error) {
	content, err := publicKey.EncodePoint(true)
	if err != nil {
		return nil, errors.New("[Wallet],CreateSignatureRedeemScript failed.")
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(content)))
	buf.Write(content)
	buf.WriteByte(byte(tx.CHECKSIG))

	return buf.Bytes(), nil
}

func CreateMultiSignRedeemScript(publicKeys []*crypto.PublicKey) ([]byte, error) {
	M := len(publicKeys)/2 + 1

	// Write M
	bigM := big.NewInt(int64(M))
	opCode := OpCode(byte(tx.PUSH1) - 1 + bigM.Bytes()[0])
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(opCode))

	//sort pubkey
	sort.Sort(crypto.PubKeySlice(publicKeys))

	// Write public keys
	for _, pubkey := range publicKeys {
		content, err := pubkey.EncodePoint(true)
		if err != nil {
			return nil, errors.New("[Wallet],CreateSignatureContract failed.")
		}
		buf.WriteByte(byte(len(content)))
		buf.Write(content)
	}

	// Write N
	bigKeys := big.NewInt(int64(len(publicKeys)))
	opCode = OpCode(byte(tx.PUSH1) - 1 + bigKeys.Bytes()[0])
	buf.WriteByte(byte(opCode))
	buf.WriteByte(tx.CHECKMULTISIG)

	return buf.Bytes(), nil
}
