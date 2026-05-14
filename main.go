package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"web3-go-indexer/db"
	"web3-go-indexer/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

const (
	ALCHEMY_URL  = "wss://eth-sepolia.g.alchemy.com/v2/O5RGplAxuF3uQR5kMLAZF"
	NFT_CONTRACT = "0x5710b77f5461bebf594cc6886A11db239D8cBE23" // NFT 合约地址
)

// Transfer 事件的 Keccak-256 哈希，不是随机值，所有 ERC-721 合约通用。
// 只要一个智能合约遵循了标准的ERC-20(代币)或ERC-721(NFT)协议，它的Transfer事件在区块链上产生的一个日志主题(Topic[0])就一定是这个值。
var transferEventSignature = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

/*
* 这是一个以太坊 NFT 转账事件监听器（Indexer），用于实时监听指定 NFT 合约的 Transfer 事件，并将所有权变更记录到本地数据库中。
 */
func main() {
	// 初始化数据库
	db.InitDB()

	// 启动链上事件监听（放到后台协程，不阻塞主线程）
	go startBlockchainListener()

	// 启动Gin HTTP服务器
	startGinServer()
}

// startBlockchainListener监听
func startBlockchainListener() {
	// 连接以太坊节点
	client, err := ethclient.Dial(ALCHEMY_URL)
	if err != nil {
		log.Fatal("Failed to connect to Ethereum node:", err)
	}
	// 延迟执行，确保善后
	// 被defer修饰的语句（通常是一个函数调用），不会立刻执行，而是会被压入一个“延迟调用栈”中。
	// defer的进阶特性：后进先出(LIFO)，即使发生panic也会执行
	defer client.Close()

	contractAddr := common.HexToAddress(NFT_CONTRACT)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
	}

	// 创建一个用于接收链上日志的“专用管道”（Channel）
	// types.log是go-ethereum库中定义的一种数据结构，专门用来存放从以太坊区块链上抓取到的日志信息
	logs := make(chan types.Log)
	// SubscribeFilterLogs创建了一个订阅（Subscription），他会持续占用服务器的内存和带宽来接收链上日志。
	// context.Background()返回一个空的、non-nil、永不取消的根上下文（Root Context）。
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatal("Failed to subscribe to logs:", err)
	}
	// 在监听器停止工作前，一定要取消这个订阅
	defer sub.Unsubscribe()

	log.Println("✅ Listening for Transfer events on contract:", NFT_CONTRACT)

	for { // 开启一个死循环，让监听器永远在线
		select {
		// <- 叫做“通道操作符”，它的箭头方向代表了数据的流向
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

// startGinServer提供RESTful API
func startGinServer() {
	// 使用默认中间件
	r := gin.Default()

	// 定义API路由组
	api := r.Group("/api/v1")
	{
		// 健康检查接口
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Web3 Indexer is running"})
		})
	}

	log.Println("🚀 Gin Server is running on :8080")

	// 启动服务，监听8080端口
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start Gin server:", err)
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
