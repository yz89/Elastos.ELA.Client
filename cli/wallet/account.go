package wallet

import (
	"os"
	"fmt"
	"errors"
	"strings"
	"io/ioutil"

	"ELAClient/crypto"
	. "ELAClient/common"
	. "ELAClient/wallet"
)

func addAccount(wallet Wallet, content string) error {
	// Get address content from file or cli input
	publicKeys, err := getPublicKeys(content)
	if err != nil {
		return err
	}

	if len(publicKeys) > 1 && len(publicKeys) < MinMultiSignKeys {
		return errors.New(fmt.Sprint("multi sign account require at lest ", MinMultiSignKeys, " public keys"))
	}

	_, err = wallet.AddAccount(publicKeys...)
	if err != nil {
		return err
	}

	// When add a new address, reset stored height to trigger synchronize blocks.
	wallet.CurrentHeight(ResetHeightCode)

	return listBalanceInfo(wallet)
}

func getPublicKeys(content string) ([]*crypto.PublicKey, error) {
	// Content can not be empty
	if content == "" {
		return nil, errors.New("content should be the public key[s] file path or public key strings separated by comma")
	}

	// Get public key strings
	var publicKeyStrings []string
	if _, err := os.Stat(content); err == nil { // if content is a file

		file, err := os.OpenFile(content, os.O_RDONLY, 0666)
		if err != nil {
			return nil, errors.New("open public key file failed")
		}
		rawData, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.New("read public key file failed")
		}
		publicKeyStrings = strings.Split(strings.TrimSpace(string(rawData)), "\n")
	} else {
		publicKeyStrings = strings.Split(strings.TrimSpace(content), ",")
	}

	// Check if have duplicate public key
	keyMap := map[string]string{}
	for _, publicKeyString := range publicKeyStrings {
		if keyMap[publicKeyString] == "" {
			keyMap[publicKeyString] = publicKeyString
		} else {
			return nil, errors.New(fmt.Sprint("duplicate public key:", publicKeyString))
		}
	}

	// Decode public keys from public key strings
	var publicKeys []*crypto.PublicKey
	for _, v := range publicKeyStrings {
		keyBytes, err := HexStringToBytes(strings.TrimSpace(v))
		if err != nil {
			return nil, err
		}
		publicKey, err := crypto.DecodePoint(keyBytes)
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, publicKey)
	}

	return publicKeys, nil
}

func deleteAccount(wallet Wallet, address string) error {
	programHash, err := ToProgramHash(address)
	if err != nil {
		return err
	}

	err = wallet.DeleteAddress(programHash)
	if err != nil {
		return err
	}

	return listBalanceInfo(wallet)
}
