package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"web3-go-indexer/models"

	"web3-go-indexer/db"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	ALCHEMY_URL  = "wss://eth-sepolia.g.alchemy.com/v2/O5RGplAxuF3uQR5kMLAZF"
	NFT_CONTRACT = "0x5710b77f5461bebf594cc6886A11db239D8cBE23" // NFT 合约地址
)

var transferEventSignature = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func main() {
	// 初始化数据库
	db.InitDB()

	// 连接以太坊节点
	client, err := ethclient.Dial(ALCHEMY_URL)
	if err != nil {
		log.Fatal("Failed to connect to Ethereum node:", err)
	}
	defer client.Close()

	contractAddr := common.HexToAddress(NFT_CONTRACT)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatal("Failed to subscribe to logs:", err)
	}
	defer sub.Unsubscribe()

	log.Println("✅ Listening for Transfer events on contract:", NFT_CONTRACT)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal("Subscription error:", err)
		case vLog := <-logs:
			// 只处理 Transfer 事件
			if vLog.Topics[0] == transferEventSignature {
				handleTransferEvent(vLog)
			}
		}
	}
}

func handleTransferEvent(vLog types.Log) {
	// 解析 Transfer(from, to, tokenId)
	if len(vLog.Topics) != 4 {
		log.Println("⚠️ Invalid Transfer event topics length")
		return
	}

	from := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	to := common.HexToAddress(vLog.Topics[2].Hex()).Hex()
	tokenID := new(big.Int).SetBytes(common.LeftPadBytes(vLog.Topics[3].Bytes(), 32))

	// 打印日志
	fmt.Printf("🆕 Transfer Event:\n  From: %s\n  To: %s\n  TokenID: %s\n  Tx: %s\n  Block: %d\n",
		from, to, tokenID.String(), vLog.TxHash.Hex(), vLog.BlockNumber)

	// 保存到数据库（只记录接收方为 owner）
	nft := models.NFT{
		TokenID:  tokenID.String(),
		Owner:    to, // 小写地址，Hex() 返回小写
		Contract: vLog.Address.Hex(),
		TxHash:   vLog.TxHash.Hex(),
		BlockNum: vLog.BlockNumber,
	}

	// 防止重复插入（可根据 TokenID + Contract 去重）
	result := db.DB.Where("token_id = ? AND contract = ?", nft.TokenID, nft.Contract).FirstOrCreate(&nft)
	if result.Error != nil {
		log.Printf("❌ Failed to save NFT: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("✅ Saved new NFT ownership: TokenID=%s, Owner=%s", nft.TokenID, nft.Owner)
	} else {
		log.Printf("ℹ️ NFT already exists: TokenID=%s", nft.TokenID)
	}
}
