package wallet

import (
	"fmt"
	"os"

	"github.com/elastos/Elastos.ELA.Client/rpc"
	. "github.com/elastos/Elastos.ELA.Client/common"
	tx "github.com/elastos/Elastos.ELA.Client/core/transaction"
	"github.com/cheggaaa/pb"
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

	var chainHeight uint32
	var currentHeight uint32
	var needSync bool

	for {
		chainHeight, currentHeight, needSync = sync.needSyncBlocks()
		if !needSync {
			break
		}
		bar := pb.StartNew(int(chainHeight - currentHeight + 1))
		for currentHeight <= chainHeight {
			block, err := rpc.GetBlockByHeight(currentHeight)
			if err != nil {
				fmt.Println("Get block error on height:", currentHeight, "error:", err)
				os.Exit(1)
			}
			sync.processBlock(block)

			// Update wallet height
			currentHeight = sync.CurrentHeight(block.BlockData.Height + 1)
			bar.Increment()
		}
		bar.Finish()
	}
}

func (sync *DataSyncImpl) needSyncBlocks() (uint32, uint32, bool) {

	chainHeight, err := rpc.GetChainHeight()
	if err != nil {
		return 0, 0, false
	}

	currentHeight := sync.CurrentHeight(QueryHeightCode)

	if currentHeight >= chainHeight+1 {
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

func (sync *DataSyncImpl) processBlock(block *rpc.BlockInfo) {
	for _, txn := range block.Transactions {

		// Add UTXOs to wallet address from transaction outputs
		for index, output := range txn.Outputs {
			if addr, ok := sync.containAddress(output.Address); ok {
				// Create UTXO input from output
				txHashBytes, _ := HexStringToBytesReverse(txn.Hash)
				referTxHash, _ := Uint256FromBytes(txHashBytes)
				lockTime := output.OutputLock
				if txn.TxType == tx.CoinBase {
					lockTime = block.BlockData.Height + 100
				}
				amount, _ := StringToFixed64(output.Value)
				// Save UTXO input to data store
				addressUTXO := &AddressUTXO{
					Op:       tx.NewOutPoint(*referTxHash, uint16(index)),
					Amount:   amount,
					LockTime: lockTime,
				}
				sync.AddAddressUTXO(addr.ProgramHash, addressUTXO)
			}
		}

		// Delete UTXOs from wallet by transaction inputs
		for _, input := range txn.UTXOInputs {
			txHashBytes, _ := HexStringToBytesReverse(input.ReferTxID)
			referTxID, _ := Uint256FromBytes(txHashBytes)
			sync.DeleteUTXO(tx.NewOutPoint(*referTxID, input.ReferTxOutputIndex))
		}
	}
}
