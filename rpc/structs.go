package rpc

import (
	. "github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type PayloadInfo interface{}

type TxAttributeInfo struct {
	Usage AttributeUsage
	Data  string
}

type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
	Sequence           uint32
	Address            string
	Value              string
}

type BalanceTxInputInfo struct {
	AssetID     string
	Value       Fixed64
	ProgramHash string
}

type TxoutputInfo struct {
	AssetID    string
	Value      string
	Address    string
	OutputLock uint32
}

type ProgramInfo struct {
	Code      string
	Parameter string
}

type TxInfo struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        PayloadInfo
	Attributes     []TxAttributeInfo
	UTXOInputs     []UTXOTxInputInfo
	BalanceInputs  []BalanceTxInputInfo
	Outputs        []TxoutputInfo
	LockTime       uint32
	Programs       []ProgramInfo

	Timestamp         uint32 `json:",omitempty"`
	Confirminations   uint32 `json:",omitempty"`
	TxSize            uint32 `json:",omitempty"`
	Hash              string
}

type BlockInfo struct {
	Hash              string   `json:"hash"`
	Confirmations     uint32   `json:"confirmations"`
	Size              uint32   `json:"size"`
	Height            uint32   `json:"height"`
	Version           uint32   `json:"version"`
	Merkleroot        string   `json:"merkleroot"`
	Time              uint32   `json:"time"`
	Nonce             uint32   `json:"nonce"`
	Difficulty        string   `json:"difficulty"`
	Bits              uint32   `json:"bits"`
	Previousblockhash string   `json:"Previousblockhash"`
	Tx                []string `json:"tx"`
}
