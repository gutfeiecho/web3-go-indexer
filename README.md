```
type NFT struct {
	ID       uint   `gorm:"primaryKey"`
	TokenID  string `gorm:"index"`
	Owner    string `gorm:"index"` // 小写地址
	Contract string `gorm:"not null"`
	TxHash   string `gorm:"not null"`
	BlockNum uint64 `gorm:"not null"`
}
type TokenBalance struct {
	gorm.Model
	Address   string `gorm:"index"` // 持有者地址
	TokenAddr string `gorm:"index"` // 代币合约地址
	Symbol    string // 代币符号 (USDT, DAI)
	Balance   string // 余额 (Big Integer String)
	Decimals  int    // 精度
	ChainID   int64  // 链ID (防止多链地址冲突)
}
```
| 特性 | 包含 `gorm.Model` (如 TokenBalance) | 不包含 `gorm.Model` (如 NFT) |
| :--- | :--- | :--- |
| 主键 | 默认是自增 `ID` | 可自定义（如 `TokenID`, `Address`） |
| 时间记录 | 自动记录 DB 操作时间 | 需手动定义（如记录 `BlockNum`） |
| 删除行为 | 默认软删除（逻辑删除） | 默认硬删除（物理删除） |
| 适用场景 | 传统业务表、需要审计日志 | 链上数据映射、高性能归档、自定义主键 |

