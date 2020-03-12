package config

import (
	"time"

	"github.com/jinzhu/gorm"
)

// TimeLayout 转换字符
const TimeLayout = "2006-01-02 15:04:05"
const timeTemplate = "2006-01-02T15:04:05+08:00"

// TimeLayout 转换字符

var (
	dateTime = time.Now().In(time.FixedZone("CST", 8*3600)).Format(TimeLayout)
)

// ParseInLocation 字符串转换格式
func ParseInLocation(t string) string {
	stamp, _ := time.ParseInLocation(timeTemplate, t, time.Local)
	return stamp.In(time.FixedZone("CST", 8*3600)).Format(TimeLayout)
}

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
	err = scope.SetColumn("CreatedAt", ParseInLocation(p.CreatedAt))
	if err != nil {
		return err
	}
	err = scope.SetColumn("UpdatedAt", dateTime)
	if err != nil {
		return err
	}
	return nil
}
