package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type LogTransfer struct {
	From   common.Address
	To     common.Address
	Value  *big.Int
	Symbol string
}

func init() {
	godotenv.Load()
}

func parseInt(data string) int64 {
	value, _ := strconv.ParseInt(data, 10, 64)
	return value
}

func trackingNativeTokens() {
	userAddress := os.Getenv("USERADDRESS")
	blockNumberStart := parseInt(os.Getenv("NUMBER"))
	botApi := os.Getenv("BOTAPI")
	rpc := os.Getenv("RPC")
	bot, err := tgbotapi.NewBotAPI(botApi)
	if err != nil {
		fmt.Println(err)
		return
	}

	client, err := ethclient.Dial(rpc)
	if err != nil {
		fmt.Println(err)
		return
	}
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	number := header.Number.Int64()

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	for index := blockNumberStart; index < number; index++ {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(index))
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, transaction := range block.Transactions() {
			// if transaction.To().Hex() == userAddress {
			if transaction.To() != nil && len(transaction.Data()) < 1 && transaction.To().Hex() == userAddress {
				// len(data) < 1 --> transfer native tokens --> check to for notify
				//find address from
				var from common.Address
				if message, err := transaction.AsMessage(types.NewEIP155Signer(chainID), nil); err == nil {
					// fmt.Println("native token from", m.From().Hex())
					from = message.From()
				}
				content := fmt.Sprintf("Native: \nfrom: %s\nto: %s\nvalue: %s", from.Hex(), transaction.To().Hex(), transaction.Value().String())
				bot.Send(tgbotapi.NewMessage(1262995839, content))
			}
		}
		if index == number-1 {
			number = number + 1
			time.Sleep(3 * time.Second)
			// fmt.Println(number)
		}
	}
}

func trackingERC20Tokens() {
	blockNumberStart := parseInt(os.Getenv("NUMBER"))
	botApi := os.Getenv("BOTAPI")
	rpc := os.Getenv("RPC")
	contractAddr := os.Getenv("CONTRACTADDR")
	bot, err := tgbotapi.NewBotAPI(botApi)
	if err != nil {
		fmt.Println(err)
		return
	}

	client, err := ethclient.Dial(rpc)
	//this rpc is for BNB smart chain test net, if you want to track ETH, please change RPC
	if err != nil {
		fmt.Println(err)
		return
	}
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	number := header.Number.Int64()
	contractAddress := common.HexToAddress(contractAddr)
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(blockNumberStart),
		ToBlock:   big.NewInt(number),
		Addresses: []common.Address{
			contractAddress,
		},
	}
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		fmt.Println(err)
		return
	}
	logTransferSig := []byte("Transfer(address,address,uint256)")
	logTransferSigHash := crypto.Keccak256Hash(logTransferSig).Hex()

	// defensive checking
	for _, vLog := range logs {
		// fmt.Println("Log Block Number: ", vLog.BlockNumber)
		if len(vLog.Topics) > 0 && vLog.Topics[0].Hex() == logTransferSigHash {
			var transferEvent LogTransfer
			transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
			transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
			transferEvent.Value = new(big.Int).SetBytes(vLog.Data)
			content := fmt.Sprintf("ERC20: \nfrom: %s\nto: %s\nvalue: %s", transferEvent.From.Hex(), transferEvent.To.Hex(), transferEvent.Value.String())
			bot.Send(tgbotapi.NewMessage(1262995839, content))
		}
	}
}

func main() {
	go trackingNativeTokens()
	go trackingERC20Tokens()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
