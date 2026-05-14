package models

import "gorm.io/gorm"

// TokenBalance 代币余额表（支持 ERC-20）
type TokenBalance struct {
	gorm.Model
	Address   string `gorm:"index"` // 持有者地址
	TokenAddr string `gorm:"index"` // 代币合约地址
	Symbol    string // 代币符号 (USDT, DAI)
	Balance   string // 余额 (Big Integer String)
	Decimals  int    // 精度
	ChainID   int64  // 链ID (防止多链地址冲突)
}

// LiquidityPool 流动性池状态
type LiquidityPool struct {
	gorm.Model
	PoolAddress string `gorm:"index"`
	Token0      string // 交易对代币0
	Token1      string // 交易对代币1
	Reserve0    string // 代币0储备量
	Reserve1    string // 代币1储备量
	BlockNumber uint64
}
