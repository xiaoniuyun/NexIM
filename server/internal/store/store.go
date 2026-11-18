package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexim/server/internal/config"
	"nexim/server/internal/model"

	"github.com/go-redis/redis/v8"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	DB     *gorm.DB
	Scylla *gocql.Session
	Msg    MessageRepo // 消息库：mysql | scylla
	RDB    *redis.Client
	Ctx    context.Context
	Log    *zap.Logger
}

func New(cfg *config.Config, log *zap.Logger) (*Store, error) {
	db, err := gorm.Open(mysql.Open(cfg.DB.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, fmt.Errorf("mysql connect: %w", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	s := &Store{DB: db, RDB: rdb, Ctx: ctx, Log: log}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	if err := s.seed(); err != nil {
		return nil, err
	}

	// 消息库：standard 形态落 ScyllaDB，lite 形态落 MySQL
	if strings.EqualFold(cfg.DB.MessageDriver, "scylla") {
		sess, err := OpenScylla(cfg.DB)
		if err != nil {
			return nil, err
		}
		s.Scylla = sess
		s.Msg = NewScyllaMessageRepo(sess)
		log.Info("message store ready",
			zap.String("driver", "scylla"),
			zap.Strings("hosts", cfg.DB.Scylla.Hosts),
			zap.String("keyspace", cfg.DB.Scylla.Keyspace))
	} else {
		s.Msg = NewMySQLMessageRepo(db)
		log.Info("message store ready", zap.String("driver", "mysql"))
	}
	return s, nil
}

func (s *Store) migrate() error {
	if err := s.DB.AutoMigrate(
		&model.User{}, &model.Device{}, &model.Conversation{}, &model.Message{},
		&model.Friend{}, &model.FriendApply{}, &model.Dept{}, &model.AuditLog{},
		&model.Group{}, &model.GroupMember{}, &model.FileRecord{}, &model.Notice{},
		&model.Role{}, &model.UserRole{}, &model.Call{},
	); err != nil {
		return err
	}
	return s.fixIndexes()
}

// fixIndexes 修正历史版本遗留的错误唯一索引：
//   - im_message_0000.uk_conv_seq 曾为 seq 单列唯一，导致新会话首条消息必然主键冲突
//   - im_conversation.conv_id 曾为单列唯一，导致会话对端的那一行无法写入
//
// AutoMigrate 不会修改已存在的同名索引，这里显式按列数判断并重建。
func (s *Store) fixIndexes() error {
	// 消息表：uk_conv_seq 必须是 (conv_id, seq)
	if cols := s.indexColumns("im_message_0000", "uk_conv_seq"); len(cols) == 1 {
		s.Log.Warn("rebuild index uk_conv_seq", zap.Strings("old", cols))
		if err := s.DB.Exec("ALTER TABLE `im_message_0000` DROP INDEX `uk_conv_seq`").Error; err != nil {
			return err
		}
		if err := s.DB.Exec("ALTER TABLE `im_message_0000` ADD UNIQUE KEY `uk_conv_seq` (`conv_id`,`seq`)").Error; err != nil {
			return err
		}
	}

	// 会话表：去掉 conv_id 单列唯一，改为 (conv_id, owner_uid) 联合唯一
	if cols := s.indexColumns("im_conversation", "idx_im_conversation_conv_id"); len(cols) > 0 {
		s.Log.Warn("drop legacy unique index idx_im_conversation_conv_id", zap.Strings("columns", cols))
		if err := s.DB.Exec("ALTER TABLE `im_conversation` DROP INDEX `idx_im_conversation_conv_id`").Error; err != nil {
			return err
		}
	}
	if cols := s.indexColumns("im_conversation", "uk_conv_owner"); len(cols) != 2 {
		s.DB.Exec("ALTER TABLE `im_conversation` ADD UNIQUE KEY `uk_conv_owner` (`conv_id`,`owner_uid`)")
	}
	return nil
}

func (s *Store) indexColumns(table, index string) []string {
	var cols []string
	s.DB.Raw(`SELECT COLUMN_NAME FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?
		ORDER BY SEQ_IN_INDEX`, table, index).Scan(&cols)
	return cols
}

func (s *Store) seed() error {
	var count int64
	s.DB.Model(&model.User{}).Count(&count)
	if count > 0 {
		return nil
	}
	now := model.Now()
	depts := []model.Dept{
		{ParentID: 0, Name: "总部", Path: "/1/"},
		{ParentID: 1, Name: "研发中心", Path: "/1/3/"},
		{ParentID: 1, Name: "产品部", Path: "/1/5/"},
		{ParentID: 1, Name: "市场部", Path: "/1/8/"},
	}
	s.DB.Create(&depts)

	users := []model.User{
		{UID: "u_1001", Username: "linxiao", Nickname: "林晓", Avatar: "#4f6ef7", DeptID: 3, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1002", Username: "zhangwei", Nickname: "张伟", Avatar: "#3ba7d8", DeptID: 3, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1003", Username: "zhaolin", Nickname: "赵琳", Avatar: "#f2994a", DeptID: 3, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1004", Username: "ligong", Nickname: "李工", Avatar: "#3ba58c", DeptID: 3, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1005", Username: "chenjing", Nickname: "陈静", Avatar: "#a05ce6", DeptID: 5, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1006", Username: "heyu", Nickname: "何雨", Avatar: "#e9834a", DeptID: 8, Status: 1, CreatedAt: now, UpdatedAt: now},
	}
	for i := range users {
		h, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		users[i].PasswordHash = string(h)
	}
	s.DB.Create(&users)

	// 好友关系：林晓与其他人互为好友
	for _, uid := range []string{"u_1002", "u_1003", "u_1004", "u_1005", "u_1006"} {
		s.DB.Create(&model.Friend{UID: "u_1001", FriendUID: uid, Status: 1, CreatedAt: now})
		s.DB.Create(&model.Friend{UID: uid, FriendUID: "u_1001", Status: 1, CreatedAt: now})
	}

	// 角色
	roles := []model.Role{
		{Code: "SUPER_ADMIN", Name: "超级管理员", Permissions: `["*"]`, DataScope: "all", Version: 1},
		{Code: "USER", Name: "普通用户", Permissions: `["conv:list","message:list","message:send"]`, DataScope: "self", Version: 1},
		{Code: "DEPT_ADMIN", Name: "部门管理员", Permissions: `["conv:list","user:list","dept:list"]`, DataScope: "dept", Version: 1},
	}
	s.DB.Create(&roles)
	var adminRole model.Role
	s.DB.Where("code = ?", "SUPER_ADMIN").First(&adminRole)
	var linxiao model.User
	s.DB.Where("username = ?", "linxiao").First(&linxiao)
	s.DB.Create(&model.UserRole{UID: linxiao.UID, RoleID: adminRole.ID})

	// 群聊
	group := model.Group{GroupID: "g_2001", Name: "产品评审群", OwnerUID: "u_1001", Avatar: "#4f6ef7", Notice: "每周三 15:00 产品评审", MemberCount: 5, Status: 1, CreatedAt: now}
	s.DB.Create(&group)
	for _, uid := range []string{"u_1001", "u_1002", "u_1003", "u_1004", "u_1005"} {
		role := "member"
		if uid == "u_1001" {
			role = "owner"
		} else if uid == "u_1002" {
			role = "admin"
		}
		s.DB.Create(&model.GroupMember{GroupID: group.GroupID, UID: uid, Role: role, JoinedAt: now})
	}

	// 公告
	s.DB.Create(&model.Notice{
		Title:   "V1.0 正式版发布公告",
		Content: "V1.0 正式版将于本周五发布，新增三平台桌面客户端、Android / iOS 原生客户端、完整 RBAC 权限与消息审计能力。发布期间消息服务不中断。",
		Author:  "系统管理员", Status: 1, CreatedAt: now, UpdatedAt: now,
	})
	s.DB.Create(&model.Notice{
		Title:   "系统维护通知",
		Content: "本周日 02:00-04:00 进行系统维护，期间消息收发可能延迟，请提前做好安排。",
		Author:  "系统管理员", Status: 1, CreatedAt: now, UpdatedAt: now,
	})

	s.Log.Info("seed data created", zap.Int("users", len(users)))
	return nil
}

// Now 返回毫秒时间戳
func Now() int64 { return time.Now().UnixMilli() }
