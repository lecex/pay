package repository

import (
	"fmt"
	// 公共引入
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/util/log"

	"github.com/lecex/core/uitl"
	pb "github.com/lecex/pay/proto/order"
)

//Order 仓库接口
type Order interface {
	Amount(req *pb.ListQuery) (int64, error)
	List(req *pb.ListQuery) ([]*pb.Order, error)
	Total(req *pb.ListQuery) (int64, error)
	Create(order *pb.Order) error
	Delete(order *pb.Order) (bool, error)
	Update(order *pb.Order) error
	Get(order *pb.Order) error
	StoreIdAndOrderNoGet(order *pb.Order) error
	Exist(config *pb.Order) bool
}

// OrderRepository 订单仓库
type OrderRepository struct {
	DB *gorm.DB
}

// Amount 获取所有订单查询总量
func (repo *OrderRepository) Amount(req *pb.ListQuery) (total int64, err error) {
	type AmountStruct struct {
		Amount int64 `json:"amount"`
	}
	result := AmountStruct{}
	err = repo.DB.Table("orders").Select("SUM(total_amount) AS amount").Where(req.Where).Scan(&result).Error
	if err != nil {
		log.Log(err)
		return result.Amount, err
	}
	return result.Amount, nil
}

// List 获取所有订单信息
func (repo *OrderRepository) List(req *pb.ListQuery) (orders []*pb.Order, err error) {
	db := repo.DB
	limit, offset := uitl.Page(req.Limit, req.Page) // 分页
	sort := uitl.Sort(req.Sort)                     // 排序 默认 created_at desc
	// 查询条件
	if err := db.Where(req.Where).Order(sort).Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		log.Log(err)
		return nil, err
	}
	return orders, nil
}

// Total 获取所有订单查询总量
func (repo *OrderRepository) Total(req *pb.ListQuery) (total int64, err error) {
	orders := []pb.Order{}
	db := repo.DB
	if err := db.Where(req.Where).Find(&orders).Count(&total).Error; err != nil {
		log.Log(err)
		return total, err
	}
	return total, nil
}

// Exist 检测主键是否存在
func (repo *OrderRepository) Exist(order *pb.Order) bool {
	var count int
	repo.DB.Model(&order).Count(&count)
	return count > 0
}

// Get 获取订单信息
func (repo *OrderRepository) Get(order *pb.Order) error {
	if err := repo.DB.Where(&order).Find(&order).Error; err != nil {
		return err
	}
	return nil
}

// StoreIdAndOrderNoGet 根据 商家iD 订单ID获取
func (repo *OrderRepository) StoreIdAndOrderNoGet(order *pb.Order) error {
	if err := repo.DB.Where("store_id = ?", order.StoreId).Where("order_no = ?", order.OrderNo).Find(&order).Error; err != nil {
		return err
	}
	return nil
}

// Create 创建订单
func (repo *OrderRepository) Create(order *pb.Order) error {
	err := repo.DB.Create(order).Error
	if err != nil {
		// 写入数据库未知失败记录
		log.Log(err)
		return fmt.Errorf("注册订单失败")
	}
	return nil
}

// Update 更新订单
func (repo *OrderRepository) Update(order *pb.Order) error {
	if order.Id == "" {
		return fmt.Errorf("请传入更新id")
	}
	order.CreatedAt = ""
	return repo.DB.Model(order).Save(order).Error
}

// Delete 删除订单
func (repo *OrderRepository) Delete(order *pb.Order) (bool, error) {
	id := &pb.Order{
		Id: order.Id,
	}
	err := repo.DB.Delete(id).Error
	if err != nil {
		log.Log(err)
		return false, err
	}
	return true, nil
}
