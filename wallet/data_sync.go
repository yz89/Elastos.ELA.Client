package wallet

import (
	"fmt"
	. "ELAClient/rpc"
	. "ELAClient/common"
	tx "ELAClient/core/transaction"
)

type DataSync interface {
	SyncChainData()
}

type DataSyncImpl struct {
	DataStore
	addresses []*Address
}

func GetDataSync(dataStore DataStore) DataSync {
	return &DataSyncImpl{
		DataStore: dataStore,
	}
}

func (sync *DataSyncImpl) SyncChainData() {

	// Get the addresses in this wallet
	sync.addresses, _ = sync.GetAddresses()

	for chainHeight, currentHeight, needSync := sync.needSyncBlocks(); needSync; {

		for currentHeight < chainHeight {
			block, err := GetBlockByHeight(currentHeight)
			if err != nil {
				break
			}
			sync.processBlock(block)

			// Update wallet height
			currentHeight = sync.CurrentHeight(block.BlockData.Height + 1)

			fmt.Print(">")
		}
	}
	fmt.Print("\n")
}

func (sync *DataSyncImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := GetCurrentHeight()
	if err != nil {
		return 0, 0, false
	}

	currentHeight := sync.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight {
		return chainHeight, currentHeight, false
	}

	return chainHeight, currentHeight, true
}

func (sync *DataSyncImpl) containAddress(address string) (*Address, bool) {
	for _, addr := range sync.addresses {
		if addr.Address == address {
			return addr, true
		}
	}
	return nil, false
}

func (sync *DataSyncImpl) processBlock(block *BlockInfo) {
	// Add UTXO to wallet address from transaction outputs
	for _, txn := range block.Transactions {
		for index, output := range txn.Outputs {
			if addr, ok := sync.containAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytesReverse(txn.Hash)
				referTxHash, _ := Uint256ParseFromBytes(txHashBytes)
				sequence := output.OutputLock
				if txn.TxType == tx.CoinBase {
					sequence = block.BlockData.Height + 100
				}
				input := &tx.UTXOTxInput{
					ReferTxID:          *referTxHash,
					ReferTxOutputIndex: uint16(index),
					Sequence:           sequence,
				}
				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &AddressUTXO{
					Input:  input,
					Amount: amount,
				}
				sync.AddAddressUTXO(addr.ProgramHash, addressUTXO)
			}
		}
	}

	// Delete UTXO from wallet address by transaction inputs
	for _, txn := range block.Transactions {
		for _, input := range txn.UTXOInputs {
			txHashBytes, _ := HexStringToBytesReverse(input.ReferTxID)
			referTxID, _ := Uint256ParseFromBytes(txHashBytes)
			txInput := &tx.UTXOTxInput{
				ReferTxID:          *referTxID,
				ReferTxOutputIndex: input.ReferTxOutputIndex,
				Sequence:           input.Sequence,
			}
			sync.DeleteUTXO(txInput)
		}
	}
}
