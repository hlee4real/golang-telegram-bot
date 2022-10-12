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
	From  common.Address
	To    common.Address
	Value *big.Int
	Hash  common.Hash
}

func init() {
	godotenv.Load()
}

func parseInt(data string) int64 {
	value, _ := strconv.ParseInt(data, 10, 64)
	return value
}

func trackingNativeTokens() {
	blockNumberStart := parseInt(os.Getenv("NUMBER"))
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOTAPI"))
	if err != nil {
		fmt.Println(err)
		return
	}
	client, err := ethclient.Dial(os.Getenv("RPC"))
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
			if transaction.To() != nil && len(transaction.Data()) < 1 && transaction.To().Hex() == os.Getenv("USERADDRESS") {
				// len(data) < 1 --> transfer native tokens --> check to for notify
				//find address from
				var from common.Address
				if message, err := transaction.AsMessage(types.NewEIP155Signer(chainID), nil); err == nil {
					// fmt.Println("native token from", m.From().Hex())
					from = message.From()
				}
				content := fmt.Sprintf("Native: \nfrom: %s\nto: %s\nvalue: %s\nhash: %s", from.Hex(), transaction.To().Hex(), transaction.Value().String(), transaction.Hash())
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
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOTAPI"))
	if err != nil {
		fmt.Println(err)
		return
	}
	client, err := ethclient.Dial(os.Getenv("RPC"))
	//this rpc is for BNB smart chain test net, if you want to track ETH, please change RPC
	if err != nil {
		fmt.Println(err)
		return
	}
	contractAddress := common.HexToAddress(os.Getenv("CONTRACTADDR"))
	fromBlock := big.NewInt(blockNumberStart)
	toBlock := fromBlock.Sub(fromBlock, big.NewInt(1))

	for {
		header, err := client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			fmt.Println("Get header error and sleep 20 seconds:", err)
			time.Sleep(time.Second * 20)
			continue
		}
		//make sure from = from because from is sub before for loop start
		//then make to = current block by getting it from header
		fromBlock = toBlock.Add(toBlock, big.NewInt(1))
		toBlock = header.Number
		if toBlock.Cmp(fromBlock) == -1 {
			toBlock = fromBlock
		}

		fmt.Printf("Filter logs from %d to %d\n", fromBlock.Int64(), toBlock.Int64())
		query := ethereum.FilterQuery{
			FromBlock: fromBlock,
			ToBlock:   toBlock,
			Addresses: []common.Address{
				contractAddress,
			},
		}
		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Println(err)
			continue
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
				transferEvent.Hash = vLog.TxHash
				content := fmt.Sprintf("ERC20: \nfrom: %s\nto: %s\nvalue: %s \nhash: %s", transferEvent.From.Hex(), transferEvent.To.Hex(), transferEvent.Value.String(), transferEvent.Hash.Hex())
				bot.Send(tgbotapi.NewMessage(1262995839, content))
			}
		}

		time.Sleep(time.Second * 20)
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
