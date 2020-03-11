package config

import (
	"time"

	"github.com/jinzhu/gorm"
)

// TimeLayout 转换字符
const TimeLayout = "2006-01-02 15:04:05"
// TimeLayout 转换字符

var (
	dateTime = time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
)
// BeforeCreate 插入前数据处理
func (p *Config) BeforeCreate(scope *gorm.Scope) (err error) {
	err = scope.SetColumn("CreatedAt", dateTime)
	if err != nil {
		return err
	}
	err = scope.SetColumn("UpdatedAt", dateTime)
	if err != nil {
		return err
	}
	return nil
}

// BeforeUpdate 更新前数据处理
func (p *Config) BeforeUpdate(scope *gorm.Scope) (err error) {
	err = scope.SetColumn("UpdatedAt", dateTime)
	if err != nil {
		return err
	}
	return nil
}