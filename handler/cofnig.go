package handler

import (
	"context"
	"fmt"

	pb "github.com/lecex/pay/proto/config"
	"github.com/lecex/pay/service/repository"
)

// Config 配置
type Config struct {
	Repo repository.Config
}

// List 获取所有配置
func (srv *Config) List(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	configs, err := srv.Repo.List(req.ListQuery)
	total, err := srv.Repo.Total(req.ListQuery)
	if err != nil {
		return err
	}
	res.Total = total
	res.Configs = configs
	return err
}

// Get 获取配置
func (srv *Config) Get(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	config, err := srv.Repo.Get(req.Config)
	if err != nil {
		return err
	}
	res.Config = config
	return err
}

// Create 创建配置
func (srv *Config) Create(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	config, err := srv.Repo.Create(req.Config)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建配置失败:%s", err)
	}
	res.Config = config
	res.Valid = true
	return err
}

// Update 更新配置
func (srv *Config) Update(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	valid, err := srv.Repo.Update(req.Config)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("更新配置失败:%s", err)
	}
	res.Valid = valid
	return err
}

// Delete 删除配置配置
func (srv *Config) Delete(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	valid, err := srv.Repo.Delete(req.Config)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("删除配置失败:%s", err)
	}
	res.Valid = valid
	return err
}
