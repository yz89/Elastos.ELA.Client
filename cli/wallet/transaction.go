package wallet

import (
	"os"
	"fmt"
	"bufio"
	"bytes"
	"errors"
	"strings"
	"strconv"
	"io/ioutil"

	"ELAClient/rpc"
	. "ELAClient/common"
	"ELAClient/common/log"
	walt "ELAClient/wallet"
	tx "ELAClient/core/transaction"

	"github.com/urfave/cli"
)

func createTransaction(c *cli.Context, wallet walt.Wallet) error {

	from := strings.TrimLeft(c.String("from"), "-")
	log.Info("From address:", from)
	if from == "" {
		addresses, err := wallet.GetAddresses()
		if err != nil || len(addresses) == 0 {
			return errors.New("can not get default address")
		}
		from = addresses[0].Address
	}

	feeStr := strings.TrimLeft(c.String("fee"), "-")
	log.Info("Fee:", feeStr)
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}

	fee, err := StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	multiOutput := strings.TrimLeft(c.String("multioutput"), "-")
	log.Info("Multi output:", multiOutput)
	if multiOutput != "" {
		return createMultiOutputTransaction(c, wallet, multiOutput, from, fee)
	}

	to := strings.TrimLeft(c.String("to"), "-")
	log.Info("To address:", to)
	if to == "" {
		return errors.New("use --to to specify receiver address")
	}

	amountStr := strings.TrimLeft(c.String("amount"), "-")
	log.Info("Amount:", amountStr)
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}

	amount, err := StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	lockStr := strings.TrimLeft(c.String("lock"), "-")
	log.Info("Lock time:", lockStr)
	var txn *tx.Transaction
	if lockStr == "" {
		txn, err = wallet.CreateTransaction(from, to, amount, fee)
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	} else {
		lock, err := strconv.ParseUint(lockStr, 10, 32)
		if err != nil {
			return errors.New("invalid lock height")
		}
		txn, err = wallet.CreateLockedTransaction(from, to, amount, fee, uint32(lock))
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	}
	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	fmt.Println(BytesToHexString(buf.Bytes()))

	return nil
}

func createMultiOutputTransaction(c *cli.Context, wallet walt.Wallet, path, from string, fee *Fixed64) error {
	if _, err := os.Stat(path); err != nil {
		return errors.New("invalid multi output file path")
	}
	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return errors.New("open multi output file failed")
	}

	scanner := bufio.NewScanner(file)
	var multiOutput []*walt.Output
	for scanner.Scan() {
		columns := strings.Split(scanner.Text(), ",")
		if len(columns) < 2 {
			return errors.New(fmt.Sprint("invalid multi output line:", columns))
		}
		amountStr := strings.TrimSpace(columns[1])
		amount, err := StringToFixed64(amountStr)
		if err != nil {
			return errors.New("invalid multi output transaction amount: " + amountStr)
		}
		address := strings.TrimSpace(columns[0])
		multiOutput = append(multiOutput, &walt.Output{address, amount})
		log.Trace("Multi output address:", address, ", amount:", amountStr)
	}

	lockStr := strings.TrimLeft(c.String("lock"), "-")
	log.Info("Lock time:", lockStr)
	var txn *tx.Transaction
	if lockStr == "" {
		txn, err = wallet.CreateMultiOutputTransaction(from, fee, multiOutput...)
		if err != nil {
			return errors.New("create multi output transaction failed: " + err.Error())
		}
	} else {
		lock, err := strconv.ParseUint(lockStr, 10, 32)
		if err != nil {
			return errors.New("invalid lock height")
		}
		txn, err = wallet.CreateLockedMultiOutputTransaction(from, fee, uint32(lock), multiOutput...)
		if err != nil {
			return errors.New("create multi output transaction failed: " + err.Error())
		}
	}
	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	fmt.Println(BytesToHexString(buf.Bytes()))

	return nil
}

func signTransaction(c *cli.Context, parameter string, wallet walt.Wallet) error {
	content, err := getContent(parameter)
	if err != nil {
		return err
	}
	rawData, err := HexStringToBytes(content)
	if err != nil {
		return errors.New("decode transaction content failed")
	}

	var txn tx.Transaction
	err = txn.Deserialize(bytes.NewReader(rawData))
	if err != nil {
		return errors.New("deserialize transaction failed")
	}

	transaction, err := wallet.Sign(getPassword([]byte(c.String("password")), false), &txn)
	if err != nil {
		return err
	}

	haveSign, needSign, err := transaction.ParseTransactionSig()
	fmt.Println("[", haveSign, "/", haveSign+needSign, "] Sign transaction successful")

	buf := new(bytes.Buffer)
	transaction.Serialize(buf)
	fmt.Println(BytesToHexString(buf.Bytes()))

	return nil
}

func sendTransaction(parameter string) error {
	content, err := getContent(parameter)
	if err != nil {
		return err
	}
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", content)
	if err != nil {
		return err
	}
	fmt.Println(result.(string))
	return nil
}

func getContent(parameter string) (string, error) {
	log.Info("Content:", parameter)
	parameter = strings.TrimSpace(parameter) // trim space
	var err error
	var rawData []byte
	if strings.Contains(parameter, "/") { // if parameter is a file path
		if _, err = os.Stat(parameter); err != nil {
			return parameter, errors.New("invalid transaction file path")
		}
		file, err := os.OpenFile(parameter, os.O_RDONLY, 0666)
		if err != nil {
			return parameter, errors.New("open transaction file failed")
		}
		rawData, err = ioutil.ReadAll(file)
		if err != nil {
			return parameter, errors.New("read transaction file failed")
		}
		return string(rawData), nil
	}
	return parameter, nil
}
