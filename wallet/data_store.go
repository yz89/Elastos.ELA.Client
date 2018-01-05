package wallet

import (
	"os"
	"math"
	"sync"
	"bytes"
	"database/sql"

	. "ELAClient/common"
	"ELAClient/common/log"
	tx "ELAClient/core/transaction"

	_ "github.com/mattn/go-sqlite3"
)

/*
钱包的数据仓库，存储UTXO，合约脚本等，使用SQLite
*/
const (
	DriverName      = "sqlite3"
	DBName          = "./wallet.db"
	QueryHeightCode = 0
	ResetHeightCode = math.MaxUint32
)

const (
	CreateInfoTable = `CREATE TABLE IF NOT EXISTS Info (
				Name VARCHAR(20) NOT NULL PRIMARY KEY,
				Value BLOB
			);`
	CreateAddressesTable = `CREATE TABLE IF NOT EXISTS Addresses (
				Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				ProgramHash BLOB UNIQUE,
				RedeemScript BLOB,
				AddressType INTEGER
			);`
	CreateUTXOsTable = `CREATE TABLE IF NOT EXISTS UTXOs (
				Id INTEGER NOT NULL PRIMARY KEY,
				UTXOInput BLOB UNIQUE,
				Amount VARCHAR,
				AddressId INTEGER,
				FOREIGN KEY(AddressId) REFERENCES Addresses(Id)
			);`
)

const (
	AddressTypeStandard  = 1
	AddressTypeMultiSign = 2
)

type Address struct {
	Type         int
	Address      string
	ProgramHash  *Uint160
	RedeemScript []byte
}

type AddressUTXO struct {
	Input  *tx.UTXOTxInput
	Amount *Fixed64
}

type DataStore interface {
	sync.Locker
	DataSync

	CurrentHeight(height uint32) uint32

	AddAddress(programHash *Uint160, redeemScript []byte, addressType int) error
	DeleteAddress(programHash *Uint160) error
	GetAddressByUTXO(input *tx.UTXOTxInput) (*Address, error)
	GetAddresses() ([]*Address, error)

	AddAddressUTXO(programHash *Uint160, utxo *AddressUTXO) error
	DeleteUTXO(input *tx.UTXOTxInput) error
	GetAddressUTXOs(programHash *Uint160) ([]*AddressUTXO, error)

	ResetDataStore() error
}

type DataStoreImpl struct {
	sync.Mutex
	DataSync

	*sql.DB
}

func OpenDataStore() (DataStore, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	dataStore := &DataStoreImpl{DB: db}

	dataStore.DataSync = GetDataSync(dataStore)

	// Handle system interrupt signals
	dataStore.catchSystemSignals()

	return dataStore, nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open(DriverName, DBName)
	if err != nil {
		log.Error("Open data db error:", err)
		return nil, err
	}
	// Create info table
	_, err = db.Exec(CreateInfoTable)
	if err != nil {
		return nil, err
	}
	// Create addresses table
	_, err = db.Exec(CreateAddressesTable)
	if err != nil {
		return nil, err
	}
	// Create UTXOs table
	_, err = db.Exec(CreateUTXOsTable)
	if err != nil {
		return nil, err
	}
	stmt, err := db.Prepare("INSERT INTO Info(Name, Value) values(?,?)")
	if err != nil {
		return nil, err
	}
	stmt.Exec("Height", uint32(0))

	return db, nil
}

func (store *DataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
		store.Close()
	})
}

func (store *DataStoreImpl) ResetDataStore() error {

	addresses, err := store.GetAddresses()
	if err != nil {
		return err
	}

	store.DB.Close()
	os.Remove(DBName)

	store.DB, err = initDB()
	if err != nil {
		return err
	}

	for _, address := range addresses {
		err = store.AddAddress(address.ProgramHash, address.RedeemScript, address.Type)
		if err != nil {
			return err
		}
	}

	return nil
}

func (store *DataStoreImpl) CurrentHeight(height uint32) uint32 {
	store.Lock()
	defer store.Unlock()

	row := store.QueryRow("SELECT Value FROM Info WHERE Name=?", "Height")
	var storedHeight uint32
	row.Scan(&storedHeight)

	if height > storedHeight {
		// Received reset height code
		if height == ResetHeightCode {
			height = 0
		}
		// Insert current height
		stmt, err := store.Prepare("UPDATE Info SET Value=? WHERE Name=?")
		if err != nil {
			return uint32(0)
		}
		_, err = stmt.Exec(height, "Height")
		if err != nil {
			return uint32(0)
		}
		return height
	}
	return storedHeight
}

func (store *DataStoreImpl) AddAddress(programHash *Uint160, redeemScript []byte, addressType int) error {
	store.Lock()
	defer store.Unlock()

	stmt, err := store.Prepare("INSERT INTO Addresses(ProgramHash, RedeemScript, AddressType) values(?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(programHash.ToArray(), redeemScript, addressType)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteAddress(programHash *Uint160) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.ToArray())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}

	// Delete UTXOs of this address
	stmt, err := store.Prepare(
		"DELETE FROM UTXOs WHERE AddressId=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(addressId)
	if err != nil {
		return err
	}

	// Delete address from address table
	stmt, err = store.Prepare("DELETE FROM Addresses WHERE Id=?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressByUTXO(input *tx.UTXOTxInput) (*Address, error) {
	store.Lock()
	defer store.Unlock()

	// Serialize input
	buf := new(bytes.Buffer)
	input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Query address by UTXO input
	sql := `SELECT A.ProgramHash, A.RedeemScript, A.AddressType FROM Addresses AS A
						  INNER JOIN UTXOs AS U ON A.Id=U.AddressId WHERE UTXOInput=?`
	row := store.QueryRow(sql, inputBytes)
	var programHashBytes []byte
	var redeemScript []byte
	var addressType int
	err := row.Scan(&programHashBytes, &redeemScript, &addressType)
	if err != nil {
		return nil, err
	}
	programHash, err := Uint160ParseFromBytes(programHashBytes)
	if err != nil {
		return nil, err
	}
	address, err := programHash.ToAddress()
	if err != nil {
		return nil, err
	}
	return &Address{addressType, address, programHash, redeemScript}, nil
}

func (store *DataStoreImpl) GetAddresses() ([]*Address, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query("SELECT ProgramHash, RedeemScript, AddressType FROM Addresses")
	if err != nil {
		log.Error("Get address query error:", err)
		return nil, err
	}
	defer rows.Close()

	var addresses []*Address
	for rows.Next() {
		var programHashBytes []byte
		var redeemScript []byte
		var addressType int
		err = rows.Scan(&programHashBytes, &redeemScript, &addressType)
		if err != nil {
			log.Error("Get address scan row:", err)
			return nil, err
		}
		programHash, err := Uint160ParseFromBytes(programHashBytes)
		if err != nil {
			return nil, err
		}
		address, err := programHash.ToAddress()
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, &Address{addressType, address, programHash, redeemScript})
	}
	return addresses, nil
}

func (store *DataStoreImpl) AddAddressUTXO(programHash *Uint160, utxo *AddressUTXO) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.ToArray())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}
	// Prepare sql statement
	stmt, err := store.Prepare("INSERT INTO UTXOs(UTXOInput, Amount, AddressId) values(?,?,?)")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	utxo.Input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Serialize amount
	buf = new(bytes.Buffer)
	utxo.Amount.Serialize(buf)
	amountBytes := buf.Bytes()
	// Do insert
	_, err = stmt.Exec(inputBytes, amountBytes, addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteUTXO(input *tx.UTXOTxInput) error {
	store.Lock()
	defer store.Unlock()

	// Prepare sql statement
	stmt, err := store.Prepare("DELETE FROM UTXOs WHERE UTXOInput=?")
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	input.Serialize(buf)
	inputBytes := buf.Bytes()
	// Do delete
	_, err = stmt.Exec(inputBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressUTXOs(programHash *Uint160) ([]*AddressUTXO, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query(`SELECT UTXOs.UTXOInput, UTXOs.Amount FROM UTXOs INNER JOIN Addresses
 								ON UTXOs.AddressId=Addresses.Id WHERE Addresses.ProgramHash=?`, programHash.ToArray())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*AddressUTXO
	for rows.Next() {
		var outputBytes []byte
		var amountBytes []byte
		err = rows.Scan(&outputBytes, &amountBytes)
		if err != nil {
			return nil, err
		}

		var input tx.UTXOTxInput
		reader := bytes.NewReader(outputBytes)
		input.Deserialize(reader)

		var amount Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &AddressUTXO{&input, &amount})
	}
	return inputs, nil
}
