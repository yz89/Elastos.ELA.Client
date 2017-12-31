package wallet

import (
	"os"
	"sync"
	"bytes"
	"errors"
	"io/ioutil"
	"crypto/rand"
	"encoding/json"
	"crypto/sha256"

	"ELAClient/crypto"
	. "ELAClient/common"
	"ELAClient/core/signature"
	"ELAClient/common/log"
)

/*
秘钥数据库，存储IV，MasterKey，PasswordHash地址公钥私钥，使用JsonFile存储
*/
const (
	KeyStoreVersion  = "1.0"
	KeystoreFilename = "keystore.dat"
)

var keyStore *KeyStoreImpl // Single instance of keystore

type KeyStore interface {
	ChangePassword(newPassword []byte) error

	GetPublicKey() *crypto.PubKey
	GetRedeemScript() []byte
	GetProgramHash() *Uint160

	Sign(data signature.Signable) ([]byte, error)
}

type KeyStoreImpl struct {
	sync.Mutex

	Version string

	IV                  string
	MasterKey           string
	PasswordHash        string
	PrivateKeyEncrypted string

	ProgramHash  string
	RedeemScript string
}

func CreateKeyStore(password []byte) (KeyStore, error) {
	if FileExisted(KeystoreFilename) {
		return nil, errors.New("CAUTION: keystore already exist!\n")
	}
	defer ClearBytes(password, len(password))

	keyStore = &KeyStoreImpl{
		Version: KeyStoreVersion,
	}

	iv := make([]byte, 16)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	keyStore.IV = BytesToHexString(iv)
	log.Info("IV created:", keyStore.IV)

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		return nil, err
	}

	passwordKey := crypto.ToAesKey(password)
	defer ClearBytes(passwordKey, 32)
	passwordHash := sha256.Sum256(passwordKey)
	defer ClearBytes(passwordHash[:], 32)
	keyStore.PasswordHash = BytesToHexString(passwordHash[:])

	masterKey, err := crypto.AesEncrypt(key[:], passwordKey, iv)
	if err != nil {
		return nil, err
	}
	keyStore.MasterKey = BytesToHexString(masterKey)
	log.Info("MasterKey created:", keyStore.MasterKey)

	privateKey, publicKey, _ := crypto.GenKeyPair()
	signatureRedeemScript, err := CreateSignatureRedeemScript(publicKey)
	if err != nil {
		return nil, err
	}
	keyStore.RedeemScript = BytesToHexString(signatureRedeemScript)
	log.Info("Redeem script created:", keyStore.RedeemScript)

	programHash, err := ToScriptHash(signatureRedeemScript, SignTypeSingle)
	if err != nil {
		return nil, err
	}
	keyStore.ProgramHash = BytesToHexString(programHash[:])
	log.Info("ProgramHash created:", keyStore.ProgramHash)

	encryptedPrivateKey, err := keyStore.encryptPrivateKey(privateKey, publicKey)
	defer ClearBytes(encryptedPrivateKey, len(encryptedPrivateKey))
	keyStore.PrivateKeyEncrypted = BytesToHexString(encryptedPrivateKey)
	log.Info("Private key encrypted:", keyStore.PrivateKeyEncrypted)

	err = keyStore.saveToFile()
	if err != nil {
		return nil, err
	}
	// Handle system interrupt signals
	keyStore.catchSystemSignals()

	return keyStore, nil
}

func OpenKeyStore(password []byte) (KeyStore, error) {
	if keyStore == nil {
		keyStore = &KeyStoreImpl{}
		err := keyStore.loadFromFile()
		if err != nil {
			return nil, err
		}
		// Handle system interrupt signals
		keyStore.catchSystemSignals()
	}
	err := keyStore.verifyPassword(password)
	if err != nil {
		return nil, err
	}

	return keyStore, nil
}

func (store *KeyStoreImpl) catchSystemSignals() {
	HandleSignal(func() {
		store.Lock()
	})
}

func (store *KeyStoreImpl) verifyPassword(password []byte) error {
	defer ClearBytes(password, len(password))

	passwordKey := crypto.ToAesKey(password)
	defer ClearBytes(passwordKey, 32)
	passwordHash := sha256.Sum256(passwordKey)
	defer ClearBytes(passwordHash[:], 32)

	origin, err := HexStringToBytes(store.PasswordHash)
	if err != nil {
		return err
	}
	if IsEqualBytes(origin, passwordHash[:]) {
		return nil
	}
	return errors.New("password wrong")
}

func (store *KeyStoreImpl) ChangePassword(password []byte) error {
	defer ClearBytes(password, len(password))

	iv, masterKey, err := store.getMasterKey()
	if err != nil {
		return err
	}

	passwordKey := crypto.ToAesKey(password)
	defer ClearBytes(passwordKey, 32)
	passwordHash := sha256.Sum256(passwordKey)
	defer ClearBytes(passwordHash[:], 32)

	masterKey, err = crypto.AesEncrypt(masterKey, passwordKey, iv)
	if err != nil {
		return err
	}

	privateKey, publicKey, err := store.decryptPrivateKey()
	if err != nil {
		return err
	}
	defer ClearBytes(privateKey, len(privateKey))

	encryptedPrivateKey, err := store.encryptPrivateKey(privateKey, publicKey)
	if err != nil {
		return err
	}
	defer ClearBytes(encryptedPrivateKey, len(encryptedPrivateKey))

	store.MasterKey = BytesToHexString(masterKey)
	store.PasswordHash = BytesToHexString(passwordHash[:])
	store.PrivateKeyEncrypted = BytesToHexString(encryptedPrivateKey)

	err = store.saveToFile()
	if err != nil {
		return err
	}

	return nil
}

func (store *KeyStoreImpl) GetPublicKey() *crypto.PubKey {
	_, publicKey, err := store.decryptPrivateKey()
	if err != nil {
		return nil
	}
	return publicKey
}

func (store *KeyStoreImpl) GetRedeemScript() []byte {
	redeemScriptBytes, _ := HexStringToBytes(store.RedeemScript)
	return redeemScriptBytes
}

func (store *KeyStoreImpl) GetProgramHash() *Uint160 {
	programHash, _ := HexStringToBytes(store.ProgramHash)
	uint160, _ := Uint160ParseFromBytes(programHash)
	return uint160
}

func (store *KeyStoreImpl) Sign(data signature.Signable) ([]byte, error) {
	privateKey, _, err := store.decryptPrivateKey()
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	data.SerializeUnsigned(buf)
	signedData, err := crypto.Sign(privateKey, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return signedData, nil
}

func (store *KeyStoreImpl) getMasterKey() (iv, masterKey []byte, err error) {
	iv, err = HexStringToBytes(store.IV)
	if err != nil {
		return nil, nil, err
	}

	masterKey, err = HexStringToBytes(store.MasterKey)
	if err != nil {
		return nil, nil, err
	}

	return iv, masterKey, nil
}

func (store *KeyStoreImpl) encryptPrivateKey(privateKey []byte, publicKey *crypto.PubKey) ([]byte, error) {
	decryptedPrivateKey := make([]byte, 96)
	defer ClearBytes(decryptedPrivateKey, 96)

	temp, err := publicKey.EncodePoint(false)
	if err != nil {
		return nil, err
	}
	for i := 1; i <= 64; i++ {
		decryptedPrivateKey[i-1] = temp[i]
	}
	for i := len(privateKey) - 1; i >= 0; i-- {
		decryptedPrivateKey[96+i-len(privateKey)] = privateKey[i]
	}
	iv, masterKey, err := store.getMasterKey()
	if err != nil {
		return nil, err
	}
	encryptedPrivateKey, err := crypto.AesEncrypt(decryptedPrivateKey, masterKey, iv)
	if err != nil {
		return nil, err
	}
	return encryptedPrivateKey, nil
}

func (store *KeyStoreImpl) decryptPrivateKey() ([]byte, *crypto.PubKey, error) {
	encryptedPrivateKey, err := HexStringToBytes(store.PrivateKeyEncrypted)
	if err != nil {
		return nil, nil, err
	}
	if len(encryptedPrivateKey) != 96 {
		return nil, nil, errors.New("invalid encrypted private key")
	}
	iv, masterKey, err := store.getMasterKey()
	if err != nil {
		return nil, nil, err
	}
	keyPair, err := crypto.AesDecrypt(encryptedPrivateKey, masterKey, iv)
	if err != nil {
		return nil, nil, err
	}
	privateKey := keyPair[64:96]

	return privateKey, crypto.NewPubKey(privateKey), nil
}

func (store *KeyStoreImpl) loadFromFile() error {
	store.Lock()
	defer store.Unlock()

	var err error
	file, err := os.OpenFile(KeystoreFilename, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}

	if file != nil {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, store)
		if err != nil {
			return err
		}
	} else {
		return errors.New("keystore file not exist")
	}
	return nil
}

func (store *KeyStoreImpl) saveToFile() error {
	store.Lock()
	defer store.Unlock()

	var err error
	file, err := os.OpenFile(KeystoreFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	if file != nil {
		data, err := json.Marshal(*store)
		if err != nil {
			return err
		}
		_, err = file.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}
