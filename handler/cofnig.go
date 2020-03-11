package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/micro/go-micro/v2/metadata"

	pb "github.com/lecex/pay/proto/config"
	"github.com/lecex/pay/service/repository"
)

// Config 配置
type Config struct {
	Repo repository.Config
}

// SelfUpdate 用户通过 token 自己更新支付数据
func (srv *Config) SelfUpdate(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	// meta["Userid"] 通过 meta 获取用户 id --- So this function needs token to use
	meta, _ := metadata.FromContext(ctx)
	if userID, ok := meta["Userid"]; ok {
		req.Config.Id = userID
		if srv.Repo.NewRecord(req.Config) {
			config, err := srv.Repo.Create(req.Config)
			if err != nil {
				return err
			}
			res.Config = config
			res.Valid = true
		} else {
			res.Valid, err = srv.Repo.Update(req.Config)
			if err != nil {
				return err
			}
			res.Config = req.Config
		}
		if err != nil {
			return err
		}
		return err
	} else {
		return errors.New("更新用户失败,未找到用户ID")
	}
	return err
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
	err = srv.Repo.Get(req.Config)
	if err != nil {
		return err
	}
	res.Config = req.Config
	return err
}

// Create 创建配置
func (srv *Config) Create(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	config, err := srv.Repo.Create(req.Config)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建配置失败")
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
		return fmt.Errorf("更新配置失败")
	}
	res.Valid = valid
	return err
}

// Delete 删除配置配置
func (srv *Config) Delete(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	valid, err := srv.Repo.Delete(req.Config)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("删除配置失败")
	}
	res.Valid = valid
	return err
}
