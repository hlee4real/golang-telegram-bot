package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

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

func trackingNativeTokens() {
	godotenv.Load()
	botApi := os.Getenv("BOTAPI")
	rpc := os.Getenv("RPC")
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
	number = 23569514 + 1
	blockNumberStart := big.NewInt(23569514)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := blockNumberStart; i.Int64() < number; i = i.Add(i, big.NewInt(1)) {
		block, err := client.BlockByNumber(context.Background(), i)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, transaction := range block.Transactions() {
			// if transaction.To().Hex() == userAddress {
			if transaction.To() != nil && len(transaction.Data()) < 1 {
				// len(data) < 1 --> transfer native tokens --> check to for notify
				//find address from
				var from common.Address
				if m, err := transaction.AsMessage(types.NewEIP155Signer(chainID), nil); err == nil {
					// fmt.Println("native token from", m.From().Hex())
					from = m.From()
				}
				// fmt.Println("---------------------------")
				// fmt.Println("native token hex", transaction.Hash().Hex())
				// fmt.Println("native token value", transaction.Value().String())
				// fmt.Println("native token to", transaction.To().Hex())
				// fmt.Println("native token", id)
				n := fmt.Sprintf("Native: \nfrom: %s\nto: %s\nvalue: %s", from.Hex(), transaction.To().Hex(), transaction.Value().String())
				fmt.Println(n)
				msg := tgbotapi.NewMessage(1262995839, n)
				bot.Send(msg)
			}
		}
		// if i.Int64() == number-1 {
		// 	time.Sleep(5 * time.Second)
		// 	number++
		// }
	}
}

func trackingERC20Tokens() {
	godotenv.Load()
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
	number = 23569514 + 1
	blockNumberStart := big.NewInt(23569514)
	contractAddress := common.HexToAddress(contractAddr)
	query := ethereum.FilterQuery{
		FromBlock: blockNumberStart,
		ToBlock:   big.NewInt(number + 100),
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
			fmt.Printf("Log Name: Transfer\n")
			var transferEvent LogTransfer
			transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
			transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
			transferEvent.Value = new(big.Int).SetBytes(vLog.Data)
			fmt.Printf("From: %s\n", transferEvent.From.Hex())
			fmt.Printf("To: %s\n", transferEvent.To.Hex())
			fmt.Println("Tokens: ", transferEvent.Value.String())
			n := fmt.Sprintf("ERC20: \nfrom: %s\nto: %s\nvalue: %s", transferEvent.From.Hex(), transferEvent.To.Hex(), transferEvent.Value.String())
			msg := tgbotapi.NewMessage(1262995839, n)
			bot.Send(msg)
		}

		// if vLog.BlockNumber == query.ToBlock.Uint64() {
		// 	time.Sleep(5 * time.Second)
		// 	query.FromBlock = big.NewInt(int64(vLog.BlockNumber + 1))
		// }
	}
}

func main() {
	go trackingNativeTokens()
	go trackingERC20Tokens()
}
