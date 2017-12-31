package wallet

import (
	"os"
	"fmt"
	"errors"
	"strings"

	"ELAClient/crypto"
	"ELAClient/wallet"
	. "ELAClient/common"
	. "ELAClient/cli/common"
	"ELAClient/common/password"

	"github.com/urfave/cli"
	"ELAClient/common/log"
)

const (
	MinMultiSignKeys = 3
)

func printLine() {
	fmt.Println("============================================================")
}

func createWallet(password string) error {
	_, err := wallet.Create(getPassword([]byte(password), true))
	if err != nil {
		return err
	}
	return showAccountInfo(password)
}

func addMultiSignAccount(wallet wallet.Wallet, pubKeysStr string) error {
	publicKeys := strings.Split(pubKeysStr, ":")
	if len(publicKeys) < MinMultiSignKeys {
		return errors.New("public keys is not enough")
	}
	var keys []*crypto.PubKey
	for _, v := range publicKeys {
		keyBytes, err := HexStringToBytes(v)
		if err != nil {
			return err
		}
		rawKey, err := crypto.DecodePoint(keyBytes)
		if err != nil {
			return err
		}
		keys = append(keys, rawKey)
	}
	programHash, err := wallet.AddMultiSignAddress(keys)
	if err != nil {
		return err
	}
	address, err := programHash.ToAddress()
	if err != nil {
		return err
	}
	fmt.Println(address)
	return nil
}

func changePassword(wallet wallet.Wallet) error {
	// Verify old password
	oldPassword, _ := password.GetPassword()
	wallet.VerifyPassword(oldPassword)

	fmt.Println("# input new password #")
	newPassword, _ := password.GetConfirmedPassword()
	if err := wallet.ChangePassword(newPassword); err != nil {
		return errors.New("failed to change password")
	}
	fmt.Println("password changed successful")

	return nil
}

func showAccountInfo(password string) error {
	log.Info("Enter show account info")
	keyStore, err := wallet.OpenKeyStore(getPassword([]byte(password), false))
	if err != nil {
		return err
	}
	programHash := keyStore.GetProgramHash()
	address, _ := programHash.ToAddress()
	publicKey, _ := keyStore.GetPublicKey().EncodePoint(true)
	printLine()
	fmt.Println("Address:     ", address)
	fmt.Println("Public Key:  ", BytesToHexString(publicKey))
	fmt.Println("ProgramHash: ", BytesToHexString(programHash.ToArrayReverse()))
	printLine()
	return nil
}

func listBalanceInfo(wallet wallet.Wallet) error {
	wallet.SyncChainData()
	addresses, err := wallet.GetAddresses()
	if err != nil {
		log.Error("Get addresses error:", err)
		return errors.New("get wallet addresses failed")
	}
	printLine()
	for _, address := range addresses {
		balance := Fixed64(0)
		programHash := address.ProgramHash
		UTXOs, err := wallet.GetAddressUTXOs(programHash)
		if err != nil {
			return errors.New("get " + address.Address + " UTXOs failed")
		}
		for _, utxo := range UTXOs {
			balance += *utxo.Amount
		}
		fmt.Println("Address:     ", address.Address)
		fmt.Println("ProgramHash: ", BytesToHexString(address.ProgramHash.ToArrayReverse()))
		fmt.Println("Balance:     ", balance.String())
		printLine()
	}
	return nil
}

func getPassword(passwd []byte, confirmed bool) []byte {
	var tmp []byte
	var err error
	if len(passwd) > 0 {
		tmp = []byte(passwd)
	} else {
		if confirmed {
			tmp, err = password.GetConfirmedPassword()
		} else {
			tmp, err = password.GetPassword()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return tmp
}

func walletAction(context *cli.Context) {
	if context.NumFlags() == 0 {
		cli.ShowSubcommandHelp(context)
		os.Exit(0)
	}
	pass := context.String("password")

	// create wallet
	if context.Bool("create") {
		if err := createWallet(pass); err != nil {
			fmt.Println("error: create wallet failed, msg:", err)
			os.Exit(1)
		}
		return
	}

	wallet, err := wallet.Open()
	if err != nil {
		fmt.Println("error: open wallet failed, msg:", err)
		os.Exit(2)
	}

	// show account info
	if context.Bool("account") {
		if err := showAccountInfo(pass); err != nil {
			fmt.Println("error: show account info failed, msg:", err)
			os.Exit(3)
		}
		return
	}

	// change password
	if context.Bool("changepassword") {
		if err := changePassword(wallet); err != nil {
			fmt.Println("error: change password failed, msg:", err)
			os.Exit(4)
		}
		return
	}

	// add multisig account
	if pubKeysStr := context.String("addmultisignaccount"); pubKeysStr != "" {
		if err := addMultiSignAccount(wallet, pubKeysStr); err != nil {
			fmt.Println("error: add multi sign account failed, msg:", err)
			os.Exit(5)
		}
		return
	}

	// show addresses balance in this wallet
	if context.Bool("balance") {
		if err := listBalanceInfo(wallet); err != nil {
			fmt.Println("error: list balance info failed, msg:", err)
			os.Exit(6)
		}
		return
	}

	// create transaction
	if context.Bool("createtransaction") {
		if err := createTransaction(context, wallet); err != nil {
			fmt.Println("error:", err)
			os.Exit(7)
		}
		return
	}

	// sign transaction
	if param := context.String("signtransaction"); param != "" {
		if err := signTransaction(context, param, wallet); err != nil {
			fmt.Println("error:", err)
			os.Exit(7)
		}
		return
	}

	// send transaction
	if param := context.String("sendtransaction"); param != "" {
		if err := sendTransaction(param); err != nil {
			fmt.Println("error:", err)
			os.Exit(7)
		}
		return
	}

	// reset wallet
	if context.Bool("reset") {
		if err := wallet.Reset(); err != nil {
			fmt.Println("error: reset wallet data store failed, msg:", err)
			os.Exit(8)
		}
		fmt.Println("wallet data store was reset successfully")
		return
	}
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "wallet",
		Usage:       "user wallet operation",
		Description: "With ela-cli wallet, you could control your asset.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "password, p",
				Usage: "keystore password",
			},
			cli.BoolFlag{
				Name:  "create, c",
				Usage: "create wallet",
			},
			cli.BoolFlag{
				Name:  "account",
				Usage: "show account info",
			},
			cli.BoolFlag{
				Name:  "changepassword",
				Usage: "change wallet password",
			},
			cli.StringFlag{
				Name:  "addmultisignaccount",
				Usage: "add new multi-sign account address",
			},
			cli.BoolFlag{
				Name:  "balance, b",
				Usage: "list balances",
			},
			cli.BoolFlag{
				Name:  "createtransaction, ct",
				Usage: "create a transaction use [--from] --to --amount --fee [--lock], or [--from] --multioutput --fee [--lock]",
			},
			cli.StringFlag{
				Name:  "signtransaction, sign",
				Usage: "sign transaction with the transaction file path or it's content",
			},
			cli.StringFlag{
				Name:  "sendtransaction, send",
				Usage: "send transaction with the transaction file path or it's content",
			},
			cli.StringFlag{
				Name:  "from",
				Usage: "the spend address of the transaction, if not specified use default address",
			},
			cli.StringFlag{
				Name:  "to",
				Usage: "the receive address of the transaction",
			},
			cli.StringFlag{
				Name:  "amount, a",
				Usage: "the transfer amount of the transaction",
			},
			cli.StringFlag{
				Name:  "fee, f",
				Usage: "the transfer fee of the transaction",
			},
			cli.StringFlag{
				Name:  "lock, l",
				Usage: "the lock time to specify when the received asset can be spent",
			},
			cli.StringFlag{
				Name:  "multioutput, m",
				Usage: "the file path to specify a CSV format file with [address,amount] as content",
			},
			cli.BoolFlag{
				Name:  "reset",
				Usage: "reset wallet data store",
			},
		},
		Action: walletAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "wallet")
			return cli.NewExitError("", 1)
		},
	}
}
