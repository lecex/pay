package repository

import (
	"fmt"
	// 公共引入
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/util/log"

	"github.com/lecex/core/uitl"
	pb "github.com/lecex/pay/proto/config"
)

//Config 仓库接口
type Config interface {
	List(req *pb.ListQuery) ([]*pb.Config, error)
	Total(req *pb.ListQuery) (int64, error)
	Create(config *pb.Config) (*pb.Config, error)
	Delete(config *pb.Config) (bool, error)
	Update(config *pb.Config) (bool, error)
	Get(config *pb.Config) error
	Exist(config *pb.Config) bool
}

// ConfigRepository 配置仓库
type ConfigRepository struct {
	DB *gorm.DB
}

// List 获取所有配置信息
func (repo *ConfigRepository) List(req *pb.ListQuery) (configs []*pb.Config, err error) {
	db := repo.DB
	limit, offset := uitl.Page(req.Limit, req.Page) // 分页
	sort := uitl.Sort(req.Sort)                     // 排序 默认 created_at desc
	// 查询条件
	if err := db.Where(req.Where).Order(sort).Limit(limit).Offset(offset).Find(&configs).Error; err != nil {
		log.Log(err)
		return nil, err
	}
	// 查询关联
	for _, config := range configs {
		repo.Related(config)
	}
	return configs, nil
}

// Total 获取所有配置查询总量
func (repo *ConfigRepository) Total(req *pb.ListQuery) (total int64, err error) {
	configs := []pb.Config{}
	db := repo.DB
	if err := db.Where(req.Where).Find(&configs).Count(&total).Error; err != nil {
		log.Log(err)
		return total, err
	}
	return total, nil
}

// Exist 检测主键是否存在
func (repo *ConfigRepository) Exist(config *pb.Config) bool {
	var count int
	repo.DB.Find(&config).Count(&count)
	return count > 0
}

// Get 获取配置信息
func (repo *ConfigRepository) Get(config *pb.Config) error {
	if err := repo.DB.Find(&config).Error; err != nil {
		return err
	}
	// 查询关联
	repo.Related(config)
	return nil
}

// Create 创建配置
func (repo *ConfigRepository) Create(config *pb.Config) (*pb.Config, error) {
	err := repo.DB.Create(config).Error
	if err != nil {
		// 写入数据库未知失败记录
		log.Log(err)
		return config, fmt.Errorf("注册配置失败")
	}
	return config, nil
}

// Update 更新配置
func (repo *ConfigRepository) Update(config *pb.Config) (bool, error) {
	if config.Id == "" {
		return false, fmt.Errorf("请传入更新id")
	}
	id := &pb.Config{
		Id: config.Id,
	}
	err := repo.DB.Model(id).Omit("CreatedAt").Updates(config).Error
	if err != nil {
		log.Log(err)
		return false, err
	}
	return true, nil
}

// Delete 删除配置
func (repo *ConfigRepository) Delete(config *pb.Config) (bool, error) {
	id := &pb.Config{
		Id: config.Id,
	}
	err := repo.DB.Delete(id).Error
	if err != nil {
		log.Log(err)
		return false, err
	}
	return true, nil
}

// Related 关联处理
func (repo *ConfigRepository) Related(config *pb.Config) (*pb.Config, error) {
	Alipay := &pb.Alipay{}
	if err := repo.DB.Model(&config).Related(Alipay).Error; err != nil {
		if err.Error() != "record not found" {
			return config, err
		}
	}

	Wechat := &pb.Wechat{}
	if err := repo.DB.Model(&config).Related(Wechat).Error; err != nil {
		if err.Error() != "record not found" {
			return config, err
		}
	}
	config.Alipay = Alipay
	config.Wechat = Wechat
	return config, nil
}
