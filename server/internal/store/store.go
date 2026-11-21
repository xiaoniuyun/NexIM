package store

import (
	"context"
	"fmt"
	"strconv"
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
	s.seedSensitive()

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
		&model.Role{}, &model.UserRole{}, &model.Call{}, &model.Favorite{},
		&model.SensitiveHit{}, &model.SystemSetting{}, &model.LoginLog{}, &model.Blacklist{},
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
		return s.finalizeSeed()
	}
	now := model.Now()
	depts := []model.Dept{
		{ParentID: 0, Name: "总部"},
		{ParentID: 1, Name: "研发中心"},
		{ParentID: 1, Name: "产品部"},
		{ParentID: 1, Name: "市场部"},
	}
	s.DB.Create(&depts)
	// 物化路径由真实自增 ID 生成，避免历史种子里路径与 ID 不一致
	for i := range depts {
		base := "/"
		for _, p := range depts {
			if p.ID == depts[i].ParentID {
				base = p.Path
				break
			}
		}
		depts[i].Path = base + strconv.FormatInt(depts[i].ID, 10) + "/"
		s.DB.Model(&model.Dept{}).Where("id = ?", depts[i].ID).Update("path", depts[i].Path)
	}

	users := []model.User{
		{UID: "u_1001", Username: "linxiao", Nickname: "林晓", Avatar: "#4f6ef7", DeptID: depts[1].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1002", Username: "zhangwei", Nickname: "张伟", Avatar: "#3ba7d8", DeptID: depts[1].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1003", Username: "zhaolin", Nickname: "赵琳", Avatar: "#f2994a", DeptID: depts[1].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1004", Username: "ligong", Nickname: "李工", Avatar: "#3ba58c", DeptID: depts[1].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1005", Username: "chenjing", Nickname: "陈静", Avatar: "#a05ce6", DeptID: depts[2].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
		{UID: "u_1006", Username: "heyu", Nickname: "何雨", Avatar: "#e9834a", DeptID: depts[3].ID, Status: 1, CreatedAt: now, UpdatedAt: now},
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

	// 角色（内置角色权限码与权限目录一致；新安装或旧库升级均由 ensureRoles 兜底）
	s.ensureRoles()
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
	return s.finalizeSeed()
}

// finalizeSeed 启动时幂等修复历史演示数据：部门引用 → 内置角色 → 物化路径
func (s *Store) finalizeSeed() error {
	if err := s.fixUserDeptRefs(); err != nil {
		return err
	}
	if err := s.ensureRoles(); err != nil {
		return err
	}
	if err := s.ensureDefaultRoles(); err != nil {
		return err
	}
	return s.fixDeptPaths()
}

// ensureDefaultRoles 内部成员兜底为普通用户（USER），外部联系人兜底为外部联系人（EXTERNAL）
func (s *Store) ensureDefaultRoles() error {
	var userRole, extRole model.Role
	hasUser := s.DB.Where("code = ?", "USER").First(&userRole).Error == nil
	hasExt := s.DB.Where("code = ?", "EXTERNAL").First(&extRole).Error == nil
	// 仅持有普通用户角色的外部联系人，改为外部联系人角色
	if hasUser && hasExt {
		if err := s.DB.Exec(`UPDATE sys_user_role ur
			JOIN sys_user u ON u.uid = ur.uid
			JOIN (SELECT uid FROM sys_user_role GROUP BY uid HAVING COUNT(1) = 1) single_role ON single_role.uid = ur.uid
			SET ur.role_id = ?
			WHERE ur.role_id = ? AND (u.user_type = 1 OR u.dept_id = 0)`, extRole.ID, userRole.ID).Error; err != nil {
			return err
		}
	}
	if hasExt {
		if err := s.DB.Exec(`INSERT INTO sys_user_role (uid, role_id)
			SELECT u.uid, ? FROM sys_user u
			LEFT JOIN sys_user_role ur ON ur.uid = u.uid
			WHERE u.status = 1 AND (u.user_type = 1 OR u.dept_id = 0) AND ur.role_id IS NULL`, extRole.ID).Error; err != nil {
			return err
		}
	}
	if hasUser {
		return s.DB.Exec(`INSERT INTO sys_user_role (uid, role_id)
			SELECT u.uid, ? FROM sys_user u
			LEFT JOIN sys_user_role ur ON ur.uid = u.uid
			WHERE u.status = 1 AND u.user_type = 0 AND u.dept_id > 0 AND ur.role_id IS NULL`, userRole.ID).Error
	}
	return nil
}

// fixUserDeptRefs 修复历史种子中「按路径编号」而非真实部门 ID 的用户引用
func (s *Store) fixUserDeptRefs() error {
	var depts []model.Dept
	if err := s.DB.Find(&depts).Error; err != nil {
		return err
	}
	alias := map[int64]int64{}
	for _, d := range depts {
		parts := strings.Split(strings.Trim(d.Path, "/"), "/")
		if len(parts) == 0 {
			continue
		}
		oldID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err == nil && oldID != d.ID {
			alias[oldID] = d.ID
		}
	}
	if len(alias) == 0 {
		return nil
	}
	var users []model.User
	if err := s.DB.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		if real, ok := alias[u.DeptID]; ok {
			if err := s.DB.Model(&model.User{}).Where("id = ?", u.ID).Update("dept_id", real).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// fixDeptPaths 依据 parent_id 重算全部物化路径，保证「本部门及下级」查询正确
func (s *Store) fixDeptPaths() error {
	var depts []model.Dept
	if err := s.DB.Order("id").Find(&depts).Error; err != nil {
		return err
	}
	byID := map[int64]model.Dept{}
	for _, d := range depts {
		byID[d.ID] = d
	}
	var pathOf func(int64, int) string
	pathOf = func(id int64, depth int) string {
		d, ok := byID[id]
		if !ok || depth > 16 {
			return ""
		}
		base := "/"
		if d.ParentID != 0 {
			base = pathOf(d.ParentID, depth+1)
		}
		if base == "" {
			return ""
		}
		return base + strconv.FormatInt(id, 10) + "/"
	}
	for _, d := range depts {
		p := pathOf(d.ID, 0)
		if p != "" && p != d.Path {
			if err := s.DB.Model(&model.Dept{}).Where("id = ?", d.ID).Update("path", p).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// builtinRoles 内置角色定义：权限码全部来自 service.PermCatalog
func builtinRoles() []model.Role {
	return []model.Role{
		{Code: "SUPER_ADMIN", Name: "超级管理员", Permissions: `["*"]`, DataScope: "all", Version: 1},
		{Code: "OPS_ADMIN", Name: "运营管理员", Permissions: `["dashboard:view","menu:users","menu:depts","menu:roles","user:list","user:create","user:edit","user:export","dept:list","dept:create","dept:edit","rbac:role:list","rbac:role:grant"]`, DataScope: "all", Version: 1},
		{Code: "AUDITOR", Name: "审计员", Permissions: `["dashboard:view","menu:audit","menu:logs","audit:message:list","audit:message:export","log:list","log:export","rtc:record:list"]`, DataScope: "all", Version: 1},
		{Code: "DEPT_ADMIN", Name: "部门管理员", Permissions: `["dashboard:view","menu:users","menu:depts","user:list","dept:list"]`, DataScope: "deptAndSub", Version: 1},
		{Code: "USER", Name: "普通用户", Permissions: `[]`, DataScope: "self", Version: 1},
		{Code: "EXTERNAL", Name: "外部联系人", Permissions: `[]`, DataScope: "self", Version: 1},
	}
}

// ensureRoles 幂等补齐内置角色；旧库中的历史权限码（conv:* 等）迁移到新目录
func (s *Store) ensureRoles() error {
	for _, seed := range builtinRoles() {
		var r model.Role
		err := s.DB.Where("code = ?", seed.Code).First(&r).Error
		if err != nil {
			if createErr := s.DB.Create(&seed).Error; createErr != nil {
				return createErr
			}
			continue
		}
		if seed.Code == "SUPER_ADMIN" {
			continue
		}
		if strings.Contains(r.Permissions, "conv:") || strings.Contains(r.Permissions, "message:") || r.DataScope == "" {
			if err := s.DB.Model(&r).Updates(map[string]interface{}{
				"permissions": seed.Permissions, "data_scope": seed.DataScope,
				"version": gorm.Expr("version + 1"),
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// seedSensitive 初始化敏感词库与默认策略（仅当 Redis 中为空时写入，避免覆盖运营配置）
func (s *Store) seedSensitive() {
	if n, err := s.RDB.SCard(s.Ctx, "sensitive:words:v1").Result(); err == nil && n == 0 {
		words := []string{"违禁词A", "敏感词B", "广告词C", "赌博", "诈骗"}
		args := make([]interface{}, 0, len(words))
		for _, w := range words {
			args = append(args, w)
		}
		_ = s.RDB.SAdd(s.Ctx, "sensitive:words:v1", args...).Err()
	}
	if _, err := s.RDB.Get(s.Ctx, "sensitive:strategy").Result(); err == redis.Nil {
		_ = s.RDB.Set(s.Ctx, "sensitive:strategy", "block", 0).Err()
	}
}

// Now 返回毫秒时间戳
func Now() int64 { return time.Now().UnixMilli() }
