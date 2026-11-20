package service

import (
	"fmt"
	"strings"
	"time"

	"nexim/server/internal/config"
	"nexim/server/internal/model"
	"nexim/server/internal/pkg"
	"nexim/server/internal/store"

	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	S *store.Store
	C *config.Config
}

func NewAuth(s *store.Store, c *config.Config) *AuthService {
	return &AuthService{S: s, C: c}
}

func (a *AuthService) Login(username, password, deviceID, platform string) (map[string]interface{}, error) {
	var u model.User
	if err := a.S.DB.Where("username = ? OR uid = ?", username, username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("账号或密码错误")
		}
		return nil, err
	}
	if u.Status != 1 {
		return nil, fmt.Errorf("账号已禁用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("账号或密码错误")
	}

	// 角色/权限来自 RBAC 表；admin 标记供管理后台登录判断（拥有任一菜单权限即可进入）
	e := ComputeEffect(a.S, u.UID)
	roles := append([]string{"user"}, e.RoleCodes...)
	isAdmin := HasPerm(a.S, u.UID, "*")
	for _, p := range e.Permissions {
		if p == "dashboard:view" || strings.HasPrefix(p, "menu:") {
			isAdmin = true
			break
		}
	}
	if isAdmin {
		roles = append(roles, "admin")
	}
	access, _, err := pkg.SignToken(a.C.Auth.JWTSecret, u.UID, u.Username, deviceID, roles, a.C.Auth.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, _, err := pkg.SignToken(a.C.Auth.JWTSecret, u.UID, u.Username, deviceID, roles, a.C.Auth.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	now := model.Now()
	if deviceID != "" {
		var d model.Device
		err := a.S.DB.Where("uid = ? AND device_id = ?", u.UID, deviceID).First(&d).Error
		if err == gorm.ErrRecordNotFound {
			d = model.Device{UID: u.UID, DeviceID: deviceID, Platform: platform, LastActive: now, CreatedAt: now}
			a.S.DB.Create(&d)
		} else if err == nil {
			a.S.DB.Model(&d).Updates(map[string]interface{}{"last_active": now, "platform": platform})
		}
	}
	a.S.DB.Model(&u).Update("last_login_at", now)

	// 外部联系人角色：注册页注册的账号 user_type = 1；历史未分配部门的账号同样视为外部联系人
	ext := IsExternalUser(&u)
	if ext {
		roles = append(roles, "external")
	}
	return map[string]interface{}{
		"accessToken": access, "refreshToken": refresh,
		"uid": u.UID, "username": u.Username, "nickname": u.Nickname,
		"avatar": u.Avatar, "signature": u.Signature, "gender": u.Gender,
		"phone": u.Phone, "title": u.Title, "deptId": u.DeptID,
		"userType": u.UserType, "isExternal": ext,
		"roles": roles, "permissions": e.Permissions, "dataScope": e.DataScope,
	}, nil
}

func (a *AuthService) Register(username, password, nickname string) (map[string]interface{}, error) {
	if len(username) < 3 || len(password) < 6 {
		return nil, fmt.Errorf("用户名至少3位，密码至少6位")
	}
	var cnt int64
	a.S.DB.Where("username = ?", username).Count(&cnt)
	if cnt > 0 {
		return nil, fmt.Errorf("用户名已存在")
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	now := model.Now()
	// 注册页注册的账号一律为外部联系人（不归属部门，默认无法查看组织架构）
	u := model.User{
		UID: fmt.Sprintf("u_%d", now), Username: username, Nickname: nickname,
		PasswordHash: string(h), Avatar: "#4f6ef7", UserType: 1, Status: 1, CreatedAt: now, UpdatedAt: now,
	}
	if nickname == "" {
		u.Nickname = username
	}
	if err := a.S.DB.Create(&u).Error; err != nil {
		return nil, err
	}
	return a.Login(username, password, "", "H5")
}

func (a *AuthService) Refresh(refreshToken string) (map[string]interface{}, error) {
	claims, err := pkg.ParseToken(a.C.Auth.JWTSecret, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token 无效")
	}
	// 检查黑名单
	if _, e := a.S.RDB.Get(a.S.Ctx, "token:blacklist:"+claims.ID).Result(); e == nil {
		return nil, fmt.Errorf("令牌已失效")
	}
	access, _, err := pkg.SignToken(a.C.Auth.JWTSecret, claims.UID, claims.Username, claims.DeviceID, claims.Roles, a.C.Auth.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"accessToken": access, "refreshToken": refreshToken}, nil
}

func (a *AuthService) Logout(jti string, exp int64) error {
	ttl := time.Until(time.Unix(exp, 0))
	if ttl <= 0 {
		return nil
	}
	return a.S.RDB.Set(a.S.Ctx, "token:blacklist:"+jti, "1", ttl).Err()
}

// Permissions 下发用户最终权限（角色并集），前端菜单/按钮与服务端中间件共用同一份数据
func (a *AuthService) Permissions(uid string) map[string]interface{} {
	e := ComputeEffect(a.S, uid)
	return map[string]interface{}{
		"permissions": e.Permissions, "roleCodes": e.RoleCodes, "roles": e.RoleNames,
		"dataScope": e.DataScope, "version": e.Version, "menus": MenusOf(e),
	}
}

func (a *AuthService) Devices(uid string) []model.Device {
	var ds []model.Device
	a.S.DB.Where("uid = ?", uid).Order("last_active desc").Find(&ds)
	return ds
}

func (a *AuthService) BindDevice(uid, deviceID, platform, channel, token string) error {
	now := model.Now()
	var d model.Device
	err := a.S.DB.Where("uid = ? AND device_id = ?", uid, deviceID).First(&d).Error
	if err == gorm.ErrRecordNotFound {
		d = model.Device{UID: uid, DeviceID: deviceID, Platform: platform, PushChannel: channel, PushToken: token, LastActive: now, CreatedAt: now}
		return a.S.DB.Create(&d).Error
	}
	if err != nil {
		return err
	}
	return a.S.DB.Model(&d).Updates(map[string]interface{}{"push_channel": channel, "push_token": token, "last_active": now}).Error
}

func (a *AuthService) SmsCode(target, scene string) error {
	key := fmt.Sprintf("sms:limit:%s:%s", target, scene)
	ok, err := a.S.RDB.SetNX(a.S.Ctx, key, "1", 60*time.Second).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("发送过于频繁，请60秒后重试")
	}
	// 演示模式：验证码固定为 123456，实际应生成并发送
	a.S.RDB.Set(a.S.Ctx, fmt.Sprintf("sms:code:%s:%s", target, scene), "123456", a.C.Auth.VerifyCodeTTL)
	return nil
}

func (a *AuthService) CheckBlacklist(jti string) bool {
	_, err := a.S.RDB.Get(a.S.Ctx, "token:blacklist:"+jti).Result()
	return err != redis.Nil
}
