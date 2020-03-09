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
	List(req *pb.ListQuery) ([]*pb.Order, error)
	Total(req *pb.ListQuery) (int64, error)
	Create(order *pb.Order) (*pb.Order, error)
	Delete(order *pb.Order) (bool, error)
	Update(order *pb.Order) (bool, error)
	Get(order *pb.Order) (*pb.Order, error)
}

// OrderRepository 订单仓库
type OrderRepository struct {
	DB *gorm.DB
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

// Get 获取订单信息
func (repo *OrderRepository) Get(order *pb.Order) (*pb.Order, error) {
	if order.Id != "" {
		if err := repo.DB.Model(&order).Where("id = ?", order.Id).Find(&order).Error; err != nil {
			return nil, err
		}
	}
	return order, nil
}

// Create 创建订单
func (repo *OrderRepository) Create(order *pb.Order) (*pb.Order, error) {
	err := repo.DB.Create(order).Error
	if err != nil {
		// 写入数据库未知失败记录
		log.Log(err)
		return order, fmt.Errorf("注册订单失败")
	}
	return order, nil
}

// Update 更新订单
func (repo *OrderRepository) Update(order *pb.Order) (bool, error) {
	if order.Id == "" {
		return false, fmt.Errorf("请传入更新id")
	}
	id := &pb.Order{
		Id: order.Id,
	}
	err := repo.DB.Model(id).Updates(order).Error
	if err != nil {
		log.Log(err)
		return false, err
	}
	return true, nil
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
