package wallet

import (
	"bytes"
	"database/sql"
	"math"
	"os"
	"sync"

	"github.com/elastos/Elastos.ELA.Client/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	_ "github.com/mattn/go-sqlite3"
)

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
				ProgramHash BLOB UNIQUE NOT NULL,
				RedeemScript BLOB UNIQUE NOT NULL,
				Type INTEGER NOT NULL
			);`
	CreateUTXOsTable = `CREATE TABLE IF NOT EXISTS UTXOs (
				OutPoint BLOB NOT NULL PRIMARY KEY,
				Amount BLOB NOT NULL,
				LockTime INTEGER NOT NULL,
				AddressId INTEGER NOT NULL,
				FOREIGN KEY(AddressId) REFERENCES Addresses(Id)
			);`
)

type UTXO struct {
	Op       *types.OutPoint
	Amount   *common.Fixed64
	LockTime uint32
}

type DataStore interface {
	sync.Locker
	DataSync

	CurrentHeight(height uint32) uint32

	AddAddress(programHash *common.Uint168, redeemScript []byte, addrType int) error
	DeleteAddress(programHash *common.Uint168) error
	GetAddressInfo(programHash *common.Uint168) (*Address, error)
	GetAddresses() ([]*Address, error)

	AddAddressUTXO(programHash *common.Uint168, utxo *UTXO) error
	DeleteUTXO(input *types.OutPoint) error
	GetAddressUTXOs(programHash *common.Uint168) ([]*UTXO, error)

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
	sql := `INSERT INTO Info(Name, Value) SELECT ?,? WHERE NOT EXISTS(SELECT 1 FROM Info WHERE Name=?)`
	_, err = db.Exec(sql, "Height", uint32(0), "Height")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (store *DataStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
		store.Close()
		os.Exit(-1)
	})
}

func (store *DataStoreImpl) ResetDataStore() error {

	_, err := store.Exec(`DROP TABLE IF EXISTS Info;
								DROP TABLE IF EXISTS UTXOs;`)
	if err != nil {
		return err
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
		_, err := store.Exec("UPDATE Info SET Value=? WHERE Name=?", height, "Height")
		if err != nil {
			return uint32(0)
		}
		return height
	}
	return storedHeight
}

func (store *DataStoreImpl) AddAddress(programHash *common.Uint168, redeemScript []byte, addrType int) error {
	store.Lock()
	defer store.Unlock()

	sql := "INSERT INTO Addresses(ProgramHash, RedeemScript, Type) values(?,?,?)"
	_, err := store.Exec(sql, programHash.Bytes(), redeemScript, addrType)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteAddress(programHash *common.Uint168) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.Bytes())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}

	// Delete UTXOs of this address
	_, err = store.Exec("DELETE FROM UTXOs WHERE AddressId=?", addressId)
	if err != nil {
		return err
	}

	// Delete address from address table
	_, err = store.Exec("DELETE FROM Addresses WHERE Id=?", addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressInfo(programHash *common.Uint168) (*Address, error) {
	store.Lock()
	defer store.Unlock()

	// Query address info by it's ProgramHash
	row := store.QueryRow(`SELECT RedeemScript, Type FROM Addresses WHERE ProgramHash=?`, programHash.Bytes())
	var redeemScript []byte
	var addrType int
	err := row.Scan(&redeemScript, &addrType)
	if err != nil {
		return nil, err
	}
	address, err := programHash.ToAddress()
	if err != nil {
		return nil, err
	}
	return &Address{address, programHash, redeemScript, addrType}, nil
}

func (store *DataStoreImpl) GetAddresses() ([]*Address, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query("SELECT ProgramHash, RedeemScript, Type FROM Addresses")
	if err != nil {
		log.Error("Get address query error:", err)
		return nil, err
	}
	defer rows.Close()

	var addresses []*Address
	for rows.Next() {
		var programHashBytes []byte
		var redeemScript []byte
		var addrType int
		err = rows.Scan(&programHashBytes, &redeemScript, &addrType)
		if err != nil {
			log.Error("Get address scan row:", err)
			return nil, err
		}
		programHash, err := common.Uint168FromBytes(programHashBytes)
		if err != nil {
			return nil, err
		}
		address, err := programHash.ToAddress()
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, &Address{address, programHash, redeemScript, addrType})
	}
	return addresses, nil
}

func (store *DataStoreImpl) AddAddressUTXO(programHash *common.Uint168, utxo *UTXO) error {
	store.Lock()
	defer store.Unlock()

	// Find addressId by ProgramHash
	row := store.QueryRow("SELECT Id FROM Addresses WHERE ProgramHash=?", programHash.Bytes())
	var addressId int
	err := row.Scan(&addressId)
	if err != nil {
		return err
	}
	// Serialize input
	buf := new(bytes.Buffer)
	utxo.Op.Serialize(buf)
	opBytes := buf.Bytes()
	// Serialize amount
	buf = new(bytes.Buffer)
	utxo.Amount.Serialize(buf)
	amountBytes := buf.Bytes()
	// Do insert
	sql := "INSERT INTO UTXOs(OutPoint, Amount, LockTime, AddressId) values(?,?,?,?)"
	_, err = store.Exec(sql, opBytes, amountBytes, utxo.LockTime, addressId)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) DeleteUTXO(op *types.OutPoint) error {
	store.Lock()
	defer store.Unlock()

	// Serialize input
	buf := new(bytes.Buffer)
	op.Serialize(buf)
	opBytes := buf.Bytes()
	// Do delete
	_, err := store.Exec("DELETE FROM UTXOs WHERE OutPoint=?", opBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *DataStoreImpl) GetAddressUTXOs(programHash *common.Uint168) ([]*UTXO, error) {
	store.Lock()
	defer store.Unlock()

	rows, err := store.Query(`SELECT UTXOs.OutPoint, UTXOs.Amount, UTXOs.LockTime FROM UTXOs INNER JOIN Addresses
 								ON UTXOs.AddressId=Addresses.Id WHERE Addresses.ProgramHash=?`, programHash.Bytes())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*UTXO
	for rows.Next() {
		var opBytes []byte
		var amountBytes []byte
		var lockTime uint32
		err = rows.Scan(&opBytes, &amountBytes, &lockTime)
		if err != nil {
			return nil, err
		}

		var op types.OutPoint
		reader := bytes.NewReader(opBytes)
		op.Deserialize(reader)

		var amount common.Fixed64
		reader = bytes.NewReader(amountBytes)
		amount.Deserialize(reader)

		inputs = append(inputs, &UTXO{&op, &amount, lockTime})
	}
	return inputs, nil
}
