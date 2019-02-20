package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/elastos/Elastos.ELA.Client/log"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/vm"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/core/types"
)

const (
	DESTROY_ADDRESS = "0000000000000000000000000000000000"
)

var SystemAssetId = getSystemAssetId()

type Transfer struct {
	Address string
	Amount  *common.Fixed64
}

type CrossChainOutput struct {
	Address           string
	Amount            *common.Fixed64
	CrossChainAddress string
}

var wallet Wallet // Single instance of wallet

type Wallet interface {
	DataStore

	Open(name string, password []byte) error
	ChangePassword(oldPassword, newPassword []byte) error

	AddStandardAccount(publicKey *crypto.PublicKey) (*common.Uint168, error)
	AddMultiSignAccount(M int, publicKey ...*crypto.PublicKey) (*common.Uint168, error)

	CreateTransaction(fromAddress, toAddress string, amount, fee *common.Fixed64) (*types.Transaction, error)
	CreateLockedTransaction(fromAddress, toAddress string, amount, fee *common.Fixed64, lockedUntil uint32) (*types.Transaction, error)
	CreateMultiOutputTransaction(fromAddress string, fee *common.Fixed64, output ...*Transfer) (*types.Transaction, error)
	CreateLockedMultiOutputTransaction(fromAddress string, fee *common.Fixed64, lockedUntil uint32, output ...*Transfer) (*types.Transaction, error)
	CreateCrossChainTransaction(fromAddress, toAddress, crossChainAddress string, amount, fee *common.Fixed64) (*types.Transaction, error)

	Sign(name string, password []byte, transaction *types.Transaction) (*types.Transaction, error)

	Reset() error
}

type WalletImpl struct {
	DataStore
	Keystore
}

func Create(name string, password []byte) (*WalletImpl, error) {
	keyStore, err := CreateKeystore(name, password)
	if err != nil {
		log.Error("Wallet create key store failed:", err)
		return nil, err
	}

	dataStore, err := OpenDataStore()
	if err != nil {
		log.Error("Wallet create data store failed:", err)
		return nil, err
	}

	dataStore.AddAddress(keyStore.GetProgramHash(), keyStore.GetRedeemScript(), TypeMaster)

	return &WalletImpl{
		DataStore: dataStore,
		Keystore:  keyStore,
	}, nil
}

func GetWallet() (*WalletImpl, error) {
	dataStore, err := OpenDataStore()
	if err != nil {
		return nil, err
	}

	return &WalletImpl{
		DataStore: dataStore,
	}, nil
}

func (wallet *WalletImpl) Open(name string, password []byte) error {
	keyStore, err := OpenKeystore(name, password)
	if err != nil {
		return err
	}
	wallet.Keystore = keyStore
	return nil
}

func (wallet *WalletImpl) AddStandardAccount(publicKey *crypto.PublicKey) (*common.Uint168, error) {
	contract, err := contract.CreateStandardContract(publicKey)
	if err != nil {
		return nil, errors.New("[Wallet], CreateStandardContract failed")
	}
	programHash := contract.ToProgramHash()
	err = wallet.AddAddress(programHash, contract.Code, TypeStand)
	if err != nil {
		return nil, err
	}

	return programHash, nil
}

func (wallet *WalletImpl) AddMultiSignAccount(M int, publicKeys ...*crypto.PublicKey) (*common.Uint168, error) {
	contract, err := contract.CreateMultiSigContract(M, publicKeys)
	programHash := contract.ToProgramHash()
	err = wallet.AddAddress(programHash, contract.Code, TypeMulti)
	if err != nil {
		return nil, err
	}

	return programHash, nil
}

func (wallet *WalletImpl) CreateTransaction(fromAddress, toAddress string, amount, fee *common.Fixed64) (*types.Transaction, error) {
	return wallet.CreateLockedTransaction(fromAddress, toAddress, amount, fee, uint32(0))
}

func (wallet *WalletImpl) CreateLockedTransaction(fromAddress, toAddress string, amount, fee *common.Fixed64, lockedUntil uint32) (*types.Transaction, error) {
	return wallet.CreateLockedMultiOutputTransaction(fromAddress, fee, lockedUntil, &Transfer{toAddress, amount})
}

func (wallet *WalletImpl) CreateMultiOutputTransaction(fromAddress string, fee *common.Fixed64, outputs ...*Transfer) (*types.Transaction, error) {
	return wallet.CreateLockedMultiOutputTransaction(fromAddress, fee, uint32(0), outputs...)
}

func (wallet *WalletImpl) CreateLockedMultiOutputTransaction(fromAddress string, fee *common.Fixed64, lockedUntil uint32, outputs ...*Transfer) (*types.Transaction, error) {
	return wallet.createTransaction(fromAddress, fee, lockedUntil, outputs...)
}

func (wallet *WalletImpl) CreateCrossChainTransaction(fromAddress, toAddress, crossChainAddress string, amount, fee *common.Fixed64) (*types.Transaction, error) {
	return wallet.createCrossChainTransaction(fromAddress, fee, uint32(0), &CrossChainOutput{toAddress, amount, crossChainAddress})
}

func (wallet *WalletImpl) createTransaction(fromAddress string, fee *common.Fixed64, lockedUntil uint32, outputs ...*Transfer) (*types.Transaction, error) {
	// Check if output is valid
	if len(outputs) == 0 {
		return nil, errors.New("[Wallet], Invalid transaction target")
	}
	// Sync chain block data before create transaction
	wallet.SyncChainData()

	// Check if from address is valid
	spender, err := common.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("[Wallet], Invalid spender address: ", fromAddress, ", error: ", err))
	}
	// Create transaction outputs
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []*types.Output            // The outputs in transaction
	totalOutputAmount += *fee          // Add transaction fee

	for _, output := range outputs {
		receiver, err := common.Uint168FromAddress(output.Address)
		if err != nil {
			return nil, errors.New(fmt.Sprint("[Wallet], Invalid receiver address: ", output.Address, ", error: ", err))
		}
		txOutput := &types.Output{
			AssetID:     SystemAssetId,
			ProgramHash: *receiver,
			Value:       *output.Amount,
			OutputLock:  lockedUntil,
		}
		totalOutputAmount += *output.Amount
		txOutputs = append(txOutputs, txOutput)
	}
	// Get spender's UTXOs
	UTXOs, err := wallet.GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := wallet.removeLockedUTXOs(UTXOs) // Remove locked UTXOs
	availableUTXOs = SortUTXOs(availableUTXOs)        // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*types.Input // The inputs in transaction
	for _, utxo := range availableUTXOs {
		input := &types.Input{
			Previous: types.OutPoint{
				TxID:  utxo.Op.TxID,
				Index: utxo.Op.Index,
			},
			Sequence: utxo.LockTime,
		}
		txInputs = append(txInputs, input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &types.Output{
				AssetID:     SystemAssetId,
				Value:       *utxo.Amount - totalOutputAmount,
				OutputLock:  uint32(0),
				ProgramHash: *spender,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}
	if totalOutputAmount > 0 {
		return nil, errors.New("[Wallet], Available token is not enough")
	}

	account, err := wallet.GetAddressInfo(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spenders account info failed")
	}

	return wallet.newTransaction(account.RedeemScript, txInputs, txOutputs, types.TransferAsset), nil
}

func (wallet *WalletImpl) createCrossChainTransaction(fromAddress string, fee *common.Fixed64, lockedUntil uint32, outputs ...*CrossChainOutput) (*types.Transaction, error) {
	// Check if output is valid
	if len(outputs) == 0 {
		return nil, errors.New("[Wallet], Invalid transaction target")
	}
	// Sync chain block data before create transaction
	wallet.SyncChainData()

	// Check if from address is valid
	spender, err := common.Uint168FromAddress(fromAddress)
	if err != nil {
		return nil, errors.New(fmt.Sprint("[Wallet], Invalid spender address: ", fromAddress, ", error: ", err))
	}
	// Create transaction outputs
	var totalOutputAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []*types.Output            // The outputs in transaction
	totalOutputAmount += *fee          // Add transaction fee
	perAccountFee := *fee / common.Fixed64(len(outputs))

	txPayload := &payload.TransferCrossChainAsset{}
	for index, output := range outputs {
		var receiver *common.Uint168
		if output.Address == DESTROY_ADDRESS {
			receiver = &common.Uint168{}
		} else {
			receiver, err = common.Uint168FromAddress(output.Address)
			if err != nil {
				return nil, errors.New(fmt.Sprint("[Wallet], Invalid receiver address: ", output.Address, ", error: ", err))
			}
		}
		txOutput := &types.Output{
			AssetID:     SystemAssetId,
			ProgramHash: *receiver,
			Value:       *output.Amount,
			OutputLock:  lockedUntil,
		}
		totalOutputAmount += *output.Amount
		txOutputs = append(txOutputs, txOutput)

		txPayload.CrossChainAddresses = append(txPayload.CrossChainAddresses, output.CrossChainAddress)
		txPayload.OutputIndexes = append(txPayload.OutputIndexes, uint64(index))
		txPayload.CrossChainAmounts = append(txPayload.CrossChainAmounts, *output.Amount-perAccountFee)
	}
	// Get spender's UTXOs
	UTXOs, err := wallet.GetAddressUTXOs(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spender's UTXOs failed")
	}
	availableUTXOs := wallet.removeLockedUTXOs(UTXOs) // Remove locked UTXOs
	availableUTXOs = SortUTXOs(availableUTXOs)        // Sort available UTXOs by value ASC

	// Create transaction inputs
	var txInputs []*types.Input // The inputs in transaction
	for _, utxo := range availableUTXOs {
		input := &types.Input{
			Previous: types.OutPoint{
				TxID:  utxo.Op.TxID,
				Index: utxo.Op.Index,
			},
			Sequence: utxo.LockTime,
		}
		txInputs = append(txInputs, input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &types.Output{
				AssetID:     SystemAssetId,
				Value:       *utxo.Amount - totalOutputAmount,
				OutputLock:  uint32(0),
				ProgramHash: *spender,
			}
			txOutputs = append(txOutputs, change)
			totalOutputAmount = 0
			break
		}
	}
	if totalOutputAmount > 0 {
		return nil, errors.New("[Wallet], Available token is not enough")
	}

	account, err := wallet.GetAddressInfo(spender)
	if err != nil {
		return nil, errors.New("[Wallet], Get spenders account info failed")
	}

	txn := wallet.newTransaction(account.RedeemScript, txInputs, txOutputs, types.TransferCrossChainAsset)
	txn.Payload = txPayload

	return txn, nil
}

func (wallet *WalletImpl) Sign(name string, password []byte, txn *types.Transaction) (*types.Transaction, error) {
	// Verify password
	err := wallet.Open(name, password)
	if err != nil {
		return nil, err
	}
	// Get sign type
	signType, err := crypto.GetScriptType(txn.Programs[0].Code)
	if err != nil {
		return nil, err
	}
	// Look up transaction type
	if signType == vm.CHECKSIG {
		// Sign single transaction
		txn, err = wallet.signStandardTransaction(txn)
		if err != nil {
			return nil, err
		}

	} else if signType == vm.CHECKMULTISIG {
		// Sign multi sign transaction
		txn, err = wallet.signMultiSignTransaction(txn)
		if err != nil {
			return nil, err
		}
	}

	return txn, nil
}

func (wallet *WalletImpl) signStandardTransaction(txn *types.Transaction) (*types.Transaction, error) {
	code := txn.Programs[0].Code
	// Get signer
	programHash:= common.ToProgramHash(byte(contract.PrefixStandard),code)

	// Check if current user is a valid signer
	if *programHash != *wallet.Keystore.GetProgramHash() {
		return nil, errors.New("[Wallet], Invalid signer")
	}
	// Sign transaction
	signedTx, err := wallet.Keystore.Sign(txn)
	if err != nil {
		return nil, err
	}
	// Add verify program for transaction
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signedTx)))
	buf.Write(signedTx)
	// Add signature
	txn.Programs[0].Parameter = buf.Bytes()

	return txn, nil
}

func (wallet *WalletImpl) signMultiSignTransaction(txn *types.Transaction) (*types.Transaction, error) {
	code := txn.Programs[0].Code
	param := txn.Programs[0].Parameter
	// Check if current user is a valid signer
	var signerIndex = -1
	programHashes, err := account.GetSigners(code)
	if err != nil {
		return nil, err
	}
	userProgramHash := wallet.Keystore.GetProgramHash()
	for i, programHash := range programHashes {
		if userProgramHash.ToCodeHash() == *programHash {
			signerIndex = i
			break
		}
	}
	if signerIndex == -1 {
		return nil, errors.New("[Wallet], Invalid multi sign signer")
	}
	// Sign transaction
	signature, err := wallet.Keystore.Sign(txn)
	if err != nil {
		return nil, err
	}
	// Append signature
	buf := new(bytes.Buffer)
	txn.SerializeUnsigned(buf)
	txn.Programs[0].Parameter, err = crypto.AppendSignature(signerIndex, signature, buf.Bytes(), code, param)
	if err != nil {
		return nil, err
	}

	return txn, nil
}

func (wallet *WalletImpl) Reset() error {
	return wallet.ResetDataStore()
}

func getSystemAssetId() common.Uint256 {
	systemToken := &types.Transaction{
		TxType:         types.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.RegisterAsset{
			Asset: payload.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		Attributes: []*types.Attribute{},
		Inputs:     []*types.Input{},
		Outputs:    []*types.Output{},
		Programs:   []*program.Program{},
	}
	return systemToken.Hash()
}

func (wallet *WalletImpl) removeLockedUTXOs(utxos []*UTXO) []*UTXO {
	var availableUTXOs []*UTXO
	var currentHeight = wallet.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.LockTime > 0 {
			if utxo.LockTime >= currentHeight {
				continue
			}
			utxo.LockTime = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func (wallet *WalletImpl) newTransaction(redeemScript []byte, inputs []*types.Input, outputs []*types.Output, txType types.TxType) *types.Transaction {
	// Create payload
	txPayload := &payload.TransferAsset{}
	// Create attributes
	txAttr := types.NewAttribute(types.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*types.Attribute, 0)
	attributes = append(attributes, &txAttr)
	// Create program
	var pg = &program.Program{redeemScript, nil}
	// Create transaction
	return &types.Transaction{
		TxType:     txType,
		Payload:    txPayload,
		Attributes: attributes,
		Inputs:     inputs,
		Outputs:    outputs,
		Programs:   []*program.Program{pg},
		LockTime:   0,
	}
}
