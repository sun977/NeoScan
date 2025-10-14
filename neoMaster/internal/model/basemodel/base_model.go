package model

import "time"

// BaseModel 提供统一的基础字段：ID、CreatedAt、UpdatedAt。
// 约定与特性：
//  1. ID 为主键，且自增。
//  2. CreatedAt/UpdatedAt 由 GORM 自动维护时间戳。
//  3. 若新模型需要“联合主键”，可在该模型的其他字段上继续标注 gorm:"primaryKey"，
//     与 BaseModel.ID 一起形成联合主键（其中之一为 ID）。
//     例如：
//     type Asset struct {
//     BaseModel
//     OrgID uint `gorm:"primaryKey"`
//     Name  string
//     }
//     上述结构会以 (ID, OrgID) 作为联合主键。
type BaseModel struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"` // 数据库字段类型为 bigint unsigned ，对应 Go 语言 uint64
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
}
