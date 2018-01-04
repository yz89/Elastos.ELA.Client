package wallet

import (
	"bytes"
	"errors"
	"strconv"
	"math/rand"

	"ELAClient/crypto"
	. "ELAClient/common"
	"ELAClient/core/asset"
	"ELAClient/common/log"
	tx "ELAClient/core/transaction"
	pg "ELAClient/core/contract/program"
	"ELAClient/core/transaction/payload"
)

var SystemAssetId = getSystemAssetId()

type Output struct {
	Address string
	Amount  *Fixed64
}

var wallet Wallet // Single instance of wallet

type Wallet interface {
	DataStore

	VerifyPassword(password []byte) error
	ChangePassword(oldPassword, newPassword []byte) error

	AddAccount(publicKey ...*crypto.PubKey) (*Uint160, error)

	CreateTransaction(fromAddress, toAddress string, amount, fee *Fixed64) (*tx.Transaction, error)
	CreateLockedTransaction(fromAddress, toAddress string, amount, fee *Fixed64, lockedUntil uint32) (*tx.Transaction, error)
	CreateMultiOutputTransaction(fromAddress string, fee *Fixed64, output ...*Output) (*tx.Transaction, error)
	CreateLockedMultiOutputTransaction(fromAddress string, fee *Fixed64, lockedUntil uint32, output ...*Output) (*tx.Transaction, error)

	Sign(password []byte, transaction *tx.Transaction) (*tx.Transaction, error)

	Reset() error
}

type WalletImpl struct {
	DataStore
	KeyStore
}

func Create(password []byte) (Wallet, error) {
	keyStore, err := CreateKeyStore(password)
	if err != nil {
		log.Error("Wallet create key store failed:", err)
		return nil, err
	}

	dataStore, err := OpenDataStore(true)
	if err != nil {
		log.Error("Wallet create data store failed:", err)
		return nil, err
	}

	dataStore.AddAddress(keyStore.GetProgramHash(), keyStore.GetRedeemScript(), AddressTypeStandard)

	wallet = &WalletImpl{
		DataStore: dataStore,
		KeyStore:  keyStore,
	}
	return wallet, nil
}

func Open() (Wallet, error) {
	if wallet == nil {
		dataStore, err := OpenDataStore(false)
		if err != nil {
			return nil, err
		}

		wallet = &WalletImpl{
			DataStore: dataStore,
		}
	}
	return wallet, nil
}

func (wallet *WalletImpl) VerifyPassword(password []byte) error {
	keyStore, err := OpenKeyStore(password)
	if err != nil {
		return err
	}
	wallet.KeyStore = keyStore
	return nil
}

func (wallet *WalletImpl) AddAccount(publicKeys ...*crypto.PubKey) (*Uint160, error) {
	var err error
	var signType int
	var addressType int
	var redeemScript []byte

	if len(publicKeys) == 1 { // Standard address
		signType = SignTypeSingle
		addressType = AddressTypeStandard
		redeemScript, err = CreateSignatureRedeemScript(publicKeys[0])
	} else { // Multi sign address
		signType = SignTypeMulti
		addressType = AddressTypeMultiSign
		redeemScript, err = CreateMultiSignRedeemScript(publicKeys)
	}
	if err != nil {
		return nil, errors.New("[Wallet], CreateRedeemScript failed")
	}

	scriptHash, err := ToScriptHash(redeemScript, signType)
	if err != nil {
		return nil, errors.New("[Wallet], CreateAddress failed")
	}

	err = wallet.AddAddress(scriptHash, redeemScript, addressType)
	if err != nil {
		return nil, err
	}

	return scriptHash, nil
}

func (wallet *WalletImpl) CreateTransaction(fromAddress, toAddress string, amount, fee *Fixed64) (*tx.Transaction, error) {
	return wallet.CreateLockedTransaction(fromAddress, toAddress, amount, fee, uint32(0))
}

func (wallet *WalletImpl) CreateLockedTransaction(fromAddress, toAddress string, amount, fee *Fixed64, lockedUntil uint32) (*tx.Transaction, error) {
	return wallet.CreateLockedMultiOutputTransaction(fromAddress, fee, lockedUntil, &Output{toAddress, amount})
}

func (wallet *WalletImpl) CreateMultiOutputTransaction(fromAddress string, fee *Fixed64, outputs ...*Output) (*tx.Transaction, error) {
	return wallet.CreateLockedMultiOutputTransaction(fromAddress, fee, uint32(0), outputs...)
}

func (wallet *WalletImpl) CreateLockedMultiOutputTransaction(fromAddress string, fee *Fixed64, lockedUntil uint32, outputs ...*Output) (*tx.Transaction, error) {
	return wallet.createTransaction(fromAddress, fee, lockedUntil, outputs...)
}

func (wallet *WalletImpl) createTransaction(fromAddress string, fee *Fixed64, lockedUntil uint32, outputs ...*Output) (*tx.Transaction, error) {
	// Check if output is valid
	if outputs == nil || len(outputs) == 0 {
		return nil, errors.New("[Wallet], Invalid transaction target")
	}
	// Sync chain block data before create transaction
	wallet.SyncChainData()

	// Check if from address is valid
	spender, err := ToProgramHash(fromAddress)
	if err != nil {
		return nil, errors.New("[Wallet], Invalid spender address")
	}
	// Create transaction outputs
	var totalOutputAmount = Fixed64(0) // The total amount will be spend
	var txOutputs []*tx.TxOutput       // The outputs in transaction
	totalOutputAmount += *fee          // Add transaction fee

	for _, output := range outputs {
		receiver, err := ToProgramHash(output.Address)
		if err != nil {
			return nil, errors.New("[Wallet], Invalid receiver address")
		}
		txOutput := &tx.TxOutput{
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
	var txInputs []*tx.UTXOTxInput // The inputs in transaction
	for _, utxo := range availableUTXOs {
		txInputs = append(txInputs, utxo.Input)
		if *utxo.Amount < totalOutputAmount {
			totalOutputAmount -= *utxo.Amount
		} else if *utxo.Amount == totalOutputAmount {
			totalOutputAmount = 0
			break
		} else if *utxo.Amount > totalOutputAmount {
			change := &tx.TxOutput{
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

	return wallet.newTransaction(txInputs, txOutputs), nil
}

func (wallet *WalletImpl) Sign(password []byte, txn *tx.Transaction) (*tx.Transaction, error) {
	// Verify password
	err := wallet.VerifyPassword(password)
	if err != nil {
		return nil, err
	}
	// Get transaction spender's address
	address, err := wallet.GetAddressByUTXO(txn.UTXOInputs[0])
	if err != nil {
		return nil, errors.New("[Wallet], Can not find spender's address")
	}
	// Look up transaction type
	if address.Type == AddressTypeStandard {

		// Sign single transaction
		txn, err = wallet.signStandardTransaction(password, address, txn)
		if err != nil {
			return nil, err
		}

	} else if address.Type == AddressTypeMultiSign {

		// Sign multi sign transaction
		txn, err = wallet.signMultiSignTransaction(password, address, txn)
		if err != nil {
			return nil, err
		}
	}

	return txn, nil
}

func (wallet *WalletImpl) signStandardTransaction(password []byte, address *Address, txn *tx.Transaction) (*tx.Transaction, error) {
	// Check if current user is a valid signer
	if *address.ProgramHash != *wallet.KeyStore.GetProgramHash() {
		return nil, errors.New("[Wallet], Invalid signer")
	}
	// Sign transaction
	signedTx, err := wallet.KeyStore.Sign(password, txn)
	if err != nil {
		return nil, err
	}
	// Add verify program for transaction
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signedTx)))
	buf.Write(signedTx)
	var program = &pg.Program{address.RedeemScript, buf.Bytes()}
	txn.SetPrograms([]*pg.Program{program})

	return txn, nil
}

func (wallet *WalletImpl) signMultiSignTransaction(password []byte, address *Address, txn *tx.Transaction) (*tx.Transaction, error) {
	// Check if current user is a valid signer
	var isSigner bool
	programHashes := tx.ParseMultiSignTransactionCode(address.RedeemScript)
	userProgramHash := wallet.KeyStore.GetProgramHash()
	for _, programHash := range programHashes {
		if *userProgramHash == *programHash {
			isSigner = true
		}
	}
	if !isSigner {
		return nil, errors.New("[Wallet], Invalid multi sign signer")
	}
	// Sign transaction
	signedTx, err := wallet.KeyStore.Sign(password, txn)
	if err != nil {
		return nil, err
	}
	// Check if verify program was set
	if len(txn.GetPrograms()) == 0 {
		// Add verify program for transaction
		buf := new(bytes.Buffer)
		buf.WriteByte(byte(len(signedTx)))
		buf.Write(signedTx)
		// Calculate M value
		M := len(programHashes)/2 + 1
		for i := 0; i < M-1; i++ {
			buf.WriteByte(PUSH0)
		}
		var program = &pg.Program{address.RedeemScript, buf.Bytes()}
		txn.SetPrograms([]*pg.Program{program})
	} else {
		// Append signature
		txn.AppendSignature(signedTx)
	}
	return txn, nil
}

func (wallet *WalletImpl) Reset() error {
	return wallet.ResetDataStore()
}

func getSystemAssetId() Uint256 {
	systemToken := &tx.Transaction{
		TxType:         tx.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.RegisterAsset{
			Asset: &asset.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: Uint160{},
		},
		Attributes: []*tx.TxAttribute{},
		UTXOInputs: []*tx.UTXOTxInput{},
		Outputs:    []*tx.TxOutput{},
		Programs:   []*pg.Program{},
	}
	return systemToken.Hash()
}

func (wallet *WalletImpl) removeLockedUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	var availableUTXOs []*AddressUTXO
	var currentHeight = wallet.CurrentHeight(QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.Input.Sequence > currentHeight {
			continue
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	return availableUTXOs
}

func (wallet *WalletImpl) newTransaction(inputs []*tx.UTXOTxInput, outputs []*tx.TxOutput) *tx.Transaction {

	txPayload := &payload.TransferAsset{}

	txAttr := tx.NewTxAttribute(tx.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	attributes := make([]*tx.TxAttribute, 0)
	attributes = append(attributes, &txAttr)

	return &tx.Transaction{
		TxType:        tx.TransferAsset,
		Payload:       txPayload,
		Attributes:    attributes,
		UTXOInputs:    inputs,
		BalanceInputs: []*tx.BalanceTxInput{},
		Outputs:       outputs,
		Programs:      []*pg.Program{},
	}
}
