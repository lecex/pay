package handler

import (
	"context"
	"fmt"

	pb "github.com/lecex/pay/proto/order"
	"github.com/lecex/pay/service/repository"
)

// Order 订单
type Order struct {
	Repo repository.Order
}

// List 获取所有订单
func (srv *Order) List(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	orders, err := srv.Repo.List(req.ListQuery)
	total, err := srv.Repo.Total(req.ListQuery)
	if err != nil {
		return err
	}
	res.Total = total
	res.Orders = orders
	return err
}

// Get 获取订单
func (srv *Order) Get(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	order, err := srv.Repo.Get(req.Order)
	if err != nil {
		return err
	}
	res.Order = order
	return err
}

// Create 创建订单
func (srv *Order) Create(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	order, err := srv.Repo.Create(req.Order)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建订单失败")
	}
	res.Order = order
	res.Valid = true
	return err
}

// Update 更新订单
func (srv *Order) Update(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	valid, err := srv.Repo.Update(req.Order)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("更新订单失败")
	}
	res.Valid = valid
	return err
}

// Delete 删除订单订单
func (srv *Order) Delete(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	valid, err := srv.Repo.Delete(req.Order)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("删除订单失败")
	}
	res.Valid = valid
	return err
}
