package migrations

import (
	db "github.com/lecex/pay/providers/database"

	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
)

func init() {
	config()
	order()
	alipay()
	wechat()
}

// config 用户数据迁移
func config() {
	config := &configPB.Config{}
	if !db.DB.HasTable(&config) {
		db.DB.Exec(`
			CREATE TABLE configs (
			id varchar(36) NOT NULL COMMENT '商家ID(user_id)',
			store_name varchar(64) DEFAULT NULL COMMENT '商户名',
			alipay_id int(11) DEFAULT 0 COMMENT '支付宝配置ID',
			wechat_id int(11) DEFAULT 0 COMMENT '微信配置ID',
			stauts int(11) DEFAULT 1 COMMENT '商品状态(禁用0、启用1)',
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY store_name (store_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8;
		`)
	}
}

// alipay 商品分类数据迁移
func alipay() {
	alipay := &configPB.Alipay{}
	if !db.DB.HasTable(&alipay) {
		db.DB.Exec(`
			CREATE TABLE alipays (
			id int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
			app_id varchar(255) DEFAULT NULL COMMENT '应用ID',
			private_key text DEFAULT NULL COMMENT '私钥',
			ali_pay_public_key text DEFAULT NULL COMMENT '支付宝公钥',
			sign_type varchar(255) DEFAULT NULL COMMENT '签名方式',
			sandbox int(11) DEFAULT 0 COMMENT '沙盒模式(禁用0、启用1)',
			PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8;
		`)
	}
}

// wechat 商品分类数据迁移
func wechat() {
	wechat := &configPB.Wechat{}
	if !db.DB.HasTable(&wechat) {
		db.DB.Exec(`
			CREATE TABLE wechats (
			id int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
			app_id varchar(255) DEFAULT NULL COMMENT '应用ID',
			mch_id varchar(255) DEFAULT NULL COMMENT '商家ID',
			api_key varchar(255) DEFAULT NULL COMMENT 'API秘钥',
			sub_app_id varchar(255) DEFAULT NULL COMMENT '子应用ID',
			sub_mch_id varchar(255) DEFAULT NULL COMMENT '子商家ID',
			sandbox int(11) DEFAULT 0 COMMENT '沙盒模式(禁用0、启用1)',
			PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8;
		`)
	}
}

// order 用户数据迁移
func order() {
	order := &orderPB.Order{}
	if !db.DB.HasTable(&order) {
		db.DB.Exec(`
			CREATE TABLE orders (
			id varchar(36) NOT NULL COMMENT '订单ID',
			store_id varchar(128) DEFAULT NULL COMMENT '商家ID',
			method varchar(36) DEFAULT NULL COMMENT '付款方式 [支付宝、微信、银联等]',
			auth_code varchar(36) DEFAULT NULL COMMENT '付款码',
			title varchar(128) DEFAULT NULL COMMENT '订单标题',
			total_amount int(16) DEFAULT NULL COMMENT '订单总金额',
			order_no varchar(36) DEFAULT NULL COMMENT '商家订单编号',
			operator_id varchar(16) DEFAULT NULL COMMENT '商户操作员编号',
			terminal_id varchar(16) DEFAULT NULL COMMENT '商户机具终端编号',
			stauts int(11) DEFAULT 0 DEFAULT NULL COMMENT '订单状态 [-1 订单关闭,0 待付款,1 付款成功]',
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY store_id_AND_order_no (store_id,order_no)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8;
		`)
	}
}
