package order

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// TimeLayout 转换字符
const TimeLayout = "2006-01-02 15:04:05"

// TimeLayout 转换字符
func dateTime() string {
	return time.Now().In(time.FixedZone("CST", 8*3600)).Format(TimeLayout)
}

// BeforeCreate 插入前数据处理
func (p *Order) BeforeCreate(scope *gorm.Scope) (err error) {
	uuid := uuid.NewV4()
	err = scope.SetColumn("Id", uuid.String())
	if err != nil {
		return err
	}
	err = scope.SetColumn("CreatedAt", dateTime())
	if err != nil {
		return err
	}
	err = scope.SetColumn("UpdatedAt", dateTime())
	if err != nil {
		return err
	}
	return nil
}

// BeforeUpdate 更新前数据处理
func (p *Order) BeforeUpdate(scope *gorm.Scope) (err error) {
	err = scope.SetColumn("UpdatedAt", dateTime())
	if err != nil {
		return err
	}
	return nil
}
