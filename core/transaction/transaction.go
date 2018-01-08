package transaction

import (
	"io"
	"fmt"
	"sort"
	"bytes"
	"errors"
	"crypto/sha256"

	. "ELAClient/common"
	"ELAClient/common/serialization"
	"ELAClient/core/contract/program"
	"ELAClient/core/transaction/payload"
)

//for different transaction types with different payload format
//and transaction process methods
type TransactionType byte

const (
	CoinBase      TransactionType = 0x00
	RegisterAsset TransactionType = 0x01
	TransferAsset TransactionType = 0x02
	Record        TransactionType = 0x03
	Deploy        TransactionType = 0x04
)

func (self TransactionType) Name() string {
	switch self {
	case CoinBase:
		return "CoinBase"
	case RegisterAsset:
		return "RegisterAsset"
	case TransferAsset:
		return "TransferAsset"
	case Record:
		return "Record"
	case Deploy:
		return "Deploy"
	default:
		return "Unknown"
	}
}

const (
	InvalidTransactionSize = -1
)

const (
	// encoded public key length 0x21 || encoded public key (33 bytes) || OP_CHECKSIG(0xac)
	PublicKeyScriptLength = 35

	// signature length(0x40) || 64 bytes signature
	SignatureScriptLength = 65

	// 1byte m || 3 encoded public keys with leading 0x40 (34 bytes * 3) ||
	// 1byte n + 1byte OP_CHECKMULTISIG
	// FIXME: if want to support 1/2 multisig
	MinMultiSignCodeLength = 105
)

//Payload define the func for loading the payload data
//base on payload type which have different struture
type Payload interface {
	//  Get payload data
	Data(version byte) []byte

	//Serialize payload data
	Serialize(w io.Writer, version byte) error

	Deserialize(r io.Reader, version byte) error
}

type Transaction struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        Payload
	Attributes     []*TxAttribute
	UTXOInputs     []*UTXOTxInput
	BalanceInputs  []*BalanceTxInput
	Outputs        []*TxOutput
	LockTime       uint32
	Programs       []*program.Program

	hash *Uint256
}

func (tx *Transaction) String() string {
	tx.Hash()
	return "Transaction: {\n\t" +
		"Hash: " + tx.hash.String() + "\n\t" +
		"TxType: " + tx.TxType.Name() + "\n\t" +
		"PayloadVersion: " + fmt.Sprint(tx.PayloadVersion) + "\n\t" +
		"Payload: " + BytesToHexString(tx.Payload.Data(tx.PayloadVersion)) + "\n\t" +
		"Attributes: " + fmt.Sprint(tx.Attributes) + "\n\t" +
		"UTXOInputs: " + fmt.Sprint(tx.UTXOInputs) + "\n\t" +
		"BalanceInputs: " + fmt.Sprint(tx.BalanceInputs) + "\n\t" +
		"Outputs: " + fmt.Sprint(tx.Outputs) + "\n\t" +
		"LockTime: " + fmt.Sprint(tx.LockTime) + "\n\t" +
		"Programs: " + fmt.Sprint(tx.Programs) + "\n\t" +
		"}\n"
}

//Serialize the Transaction
func (tx *Transaction) Serialize(w io.Writer) error {

	err := tx.SerializeUnsigned(w)
	if err != nil {
		return errors.New("Transaction txSerializeUnsigned Serialize failed.")
	}
	//Serialize  Transaction's programs
	lens := uint64(len(tx.Programs))
	err = serialization.WriteVarUint(w, lens)
	if err != nil {
		return errors.New("Transaction WriteVarUint failed.")
	}
	if lens > 0 {
		for _, p := range tx.Programs {
			err = p.Serialize(w)
			if err != nil {
				return errors.New("Transaction Programs Serialize failed.")
			}
		}
	}
	return nil
}

//Serialize the Transaction data without contracts
func (tx *Transaction) SerializeUnsigned(w io.Writer) error {
	//txType
	w.Write([]byte{byte(tx.TxType)})
	//PayloadVersion
	w.Write([]byte{tx.PayloadVersion})
	//Payload
	if tx.Payload == nil {
		return errors.New("Transaction Payload is nil.")
	}
	tx.Payload.Serialize(w, tx.PayloadVersion)
	//[]*txAttribute
	err := serialization.WriteVarUint(w, uint64(len(tx.Attributes)))
	if err != nil {
		return errors.New("Transaction item txAttribute length serialization failed.")
	}
	if len(tx.Attributes) > 0 {
		for _, attr := range tx.Attributes {
			attr.Serialize(w)
		}
	}
	//[]*UTXOInputs
	err = serialization.WriteVarUint(w, uint64(len(tx.UTXOInputs)))
	if err != nil {
		return errors.New("Transaction item UTXOInputs length serialization failed.")
	}
	if len(tx.UTXOInputs) > 0 {
		for _, utxo := range tx.UTXOInputs {
			utxo.Serialize(w)
		}
	}
	// TODO BalanceInputs
	//[]*Outputs
	err = serialization.WriteVarUint(w, uint64(len(tx.Outputs)))
	if err != nil {
		return errors.New("Transaction item Outputs length serialization failed.")
	}
	if len(tx.Outputs) > 0 {
		for _, output := range tx.Outputs {
			output.Serialize(w)
		}
	}

	serialization.WriteUint32(w, tx.LockTime)

	return nil
}

//deserialize the Transaction
func (tx *Transaction) Deserialize(r io.Reader) error {
	// tx deserialize
	err := tx.DeserializeUnsigned(r)
	if err != nil {
		return errors.New("transaction Deserialize error")
	}

	// tx program
	lens, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("transaction tx program Deserialize error")
	}

	programHashes := []*program.Program{}
	if lens > 0 {
		for i := 0; i < int(lens); i++ {
			outputHashes := new(program.Program)
			outputHashes.Deserialize(r)
			programHashes = append(programHashes, outputHashes)
		}
		tx.Programs = programHashes
	}
	return nil
}

func (tx *Transaction) DeserializeUnsigned(r io.Reader) error {
	var txType [1]byte
	_, err := io.ReadFull(r, txType[:])
	if err != nil {
		return err
	}
	tx.TxType = TransactionType(txType[0])
	return tx.DeserializeUnsignedWithoutType(r)
}

func (tx *Transaction) DeserializeUnsignedWithoutType(r io.Reader) error {
	var payloadVersion [1]byte
	_, err := io.ReadFull(r, payloadVersion[:])
	tx.PayloadVersion = payloadVersion[0]
	if err != nil {
		return err
	}

	switch tx.TxType {
	case RegisterAsset:
		tx.Payload = new(payload.RegisterAsset)
	case TransferAsset:
		tx.Payload = new(payload.TransferAsset)
	case Deploy:
	default:
		return errors.New("[Transaction], invalid transaction type.")
	}
	err = tx.Payload.Deserialize(r, tx.PayloadVersion)
	if err != nil {
		return errors.New("Payload Parse error")
	}
	//attributes
	Len, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	if Len > uint64(0) {
		for i := uint64(0); i < Len; i++ {
			attr := new(TxAttribute)
			err = attr.Deserialize(r)
			if err != nil {
				return err
			}
			tx.Attributes = append(tx.Attributes, attr)
		}
	}
	//UTXOInputs
	Len, err = serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	if Len > uint64(0) {
		for i := uint64(0); i < Len; i++ {
			utxo := new(UTXOTxInput)
			err = utxo.Deserialize(r)
			if err != nil {
				return err
			}
			tx.UTXOInputs = append(tx.UTXOInputs, utxo)
		}
	}
	//TODO balanceInputs
	//Outputs
	Len, err = serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	if Len > uint64(0) {
		for i := uint64(0); i < Len; i++ {
			output := new(TxOutput)
			output.Deserialize(r)

			tx.Outputs = append(tx.Outputs, output)
		}
	}

	temp, err := serialization.ReadUint32(r)
	tx.LockTime = uint32(temp)
	if err != nil {
		return err
	}

	return nil
}

func (tx *Transaction) GetSize() int {
	var buffer bytes.Buffer
	if err := tx.Serialize(&buffer); err != nil {
		return InvalidTransactionSize
	}

	return buffer.Len()
}

func (tx *Transaction) GetProgramHashes() ([]*Uint168, error) {
	if tx == nil {
		return nil, errors.New("[Transaction],GetProgramHashes transaction is nil.")
	}
	hashs := []*Uint168{}
	uniqHashes := []*Uint168{}
	// add inputUTXO's transaction
	referenceWithUTXO_Output := tx.Outputs
	for _, output := range referenceWithUTXO_Output {
		programHash := output.ProgramHash
		hashs = append(hashs, &programHash)
	}
	for _, attribute := range tx.Attributes {
		if attribute.Usage == Script {
			dataHash, err := Uint160ParseFromBytes(attribute.Data)
			if err != nil {
				return nil, errors.New("[Transaction], GetProgramHashes err.")
			}
			hashs = append(hashs, dataHash)
		}
	}

	//remove dupilicated hashes
	uniq := make(map[*Uint168]bool)
	for _, v := range hashs {
		uniq[v] = true
	}
	for k := range uniq {
		uniqHashes = append(uniqHashes, k)
	}
	sort.Sort(byProgramHashes(uniqHashes))
	return uniqHashes, nil
}

func (tx *Transaction) SetPrograms(programs []*program.Program) {
	tx.Programs = programs
}

func (tx *Transaction) GetPrograms() []*program.Program {
	return tx.Programs
}

func (tx *Transaction) Hash() Uint256 {
	if tx.hash == nil {
		buf := new(bytes.Buffer)
		tx.SerializeUnsigned(buf)
		temp := sha256.Sum256([]byte(buf.Bytes()))
		f := Uint256(sha256.Sum256(temp[:]))
		tx.hash = &f
	}
	return *tx.hash

}
func (tx *Transaction) IsCoinBaseTx() bool {
	return tx.TxType == CoinBase
}

func (tx *Transaction) SetHash(hash Uint256) {
	tx.hash = &hash
}

func (tx *Transaction) Verify() error {
	//TODO: Verify()
	return nil
}

func ParseMultiSignTransactionCode(code []byte) []*Uint168 {
	if len(code) < MinMultiSignCodeLength {
		fmt.Println("short code in multisig transaction detected")
		return nil
	}

	// remove last byte CHECKMULTISIG
	code = code[:len(code)-1]
	// remove m
	code = code[1:]
	// remove n
	code = code[:len(code)-1]
	if len(code)%(PublicKeyScriptLength-1) != 0 {
		fmt.Println("invalid code in multisig transaction detected")
		return nil
	}

	var programHash []*Uint168
	i := 0
	for i < len(code) {
		script := make([]byte, PublicKeyScriptLength-1)
		copy(script, code[i:i+PublicKeyScriptLength-1])
		script = append(script, 0xac)
		i += PublicKeyScriptLength - 1
		hash, _ := ToScriptHash(script, SignTypeSingle)
		programHash = append(programHash, hash)
	}

	return programHash
}

func (tx *Transaction) ParseTransactionCode() []*Uint168 {
	// TODO: parse Programs[1:]
	code := make([]byte, len(tx.Programs[0].Code))
	copy(code, tx.Programs[0].Code)

	return ParseMultiSignTransactionCode(code)
}

func (tx *Transaction) ParseTransactionSig() (havesig, needsig int, err error) {
	if len(tx.Programs) <= 0 {
		return -1, -1, errors.New("missing transation program")
	}
	x := len(tx.Programs[0].Parameter) / SignatureScriptLength
	y := len(tx.Programs[0].Parameter) % SignatureScriptLength

	return x, y, nil
}

func (tx *Transaction) AppendSignature(sig []byte) error {
	if len(tx.Programs) <= 0 {
		return errors.New("missing transation program")
	}

	newsig := []byte{}
	newsig = append(newsig, byte(len(sig)))
	newsig = append(newsig, sig...)

	havesig, _, err := tx.ParseTransactionSig()
	if err != nil {
		return err
	}

	existedsigs := tx.Programs[0].Parameter[0: havesig*SignatureScriptLength]
	leftsigs := tx.Programs[0].Parameter[havesig*SignatureScriptLength+1:]

	tx.Programs[0].Parameter = nil
	tx.Programs[0].Parameter = append(tx.Programs[0].Parameter, existedsigs...)
	tx.Programs[0].Parameter = append(tx.Programs[0].Parameter, newsig...)
	tx.Programs[0].Parameter = append(tx.Programs[0].Parameter, leftsigs...)

	return nil
}

type byProgramHashes []*Uint168

func (a byProgramHashes) Len() int      { return len(a) }
func (a byProgramHashes) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byProgramHashes) Less(i, j int) bool {
	if a[i].CompareTo(a[j]) > 0 {
		return false
	} else {
		return true
	}
}
