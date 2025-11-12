package models

type NFT struct {
	ID       uint   `gorm:"primaryKey"`
	TokenID  string `gorm:"index"`
	Owner    string `gorm:"index"` // 小写地址
	Contract string `gorm:"not null"`
	TxHash   string `gorm:"not null"`
	BlockNum uint64 `gorm:"not null"`
}
