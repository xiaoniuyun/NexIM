package service

import (
	"fmt"

	"nexim/server/internal/model"
	"nexim/server/internal/store"
)

type FavoriteService struct {
	S *store.Store
}

func NewFavorite(s *store.Store) *FavoriteService { return &FavoriteService{S: s} }

// FavoriteVO 面向客户端的收藏项，附带会话名称便于跳转
type FavoriteVO struct {
	model.Favorite
	ConvName string `json:"convName"`
}

// Add 收藏消息：对内容做快照，同一消息重复收藏保持幂等
func (f *FavoriteService) Add(uid, msgID string) (*FavoriteVO, error) {
	msg, err := f.S.Msg.GetByMsgID(msgID)
	if err != nil {
		return nil, fmt.Errorf("消息不存在")
	}
	if !f.canAccess(uid, msg) {
		return nil, fmt.Errorf("无权收藏该消息")
	}
	var exist model.Favorite
	if f.S.DB.Where("uid = ? AND msg_id = ?", uid, msgID).First(&exist).Error == nil {
		return f.vo(exist), nil
	}
	fromName := msg.FromUID
	var u model.User
	if f.S.DB.Where("uid = ?", msg.FromUID).First(&u).Error == nil && u.Nickname != "" {
		fromName = u.Nickname
	}
	fav := model.Favorite{
		UID: uid, MsgID: msg.MsgID, ConvID: msg.ConvID, MsgType: msg.Type,
		Content: msg.Content, Extra: msg.Extra, FromUID: msg.FromUID,
		FromName: fromName, CreatedAt: model.Now(),
	}
	if err := f.S.DB.Create(&fav).Error; err != nil {
		return nil, err
	}
	return f.vo(fav), nil
}

func (f *FavoriteService) canAccess(uid string, msg *model.Message) bool {
	if msg.FromUID == uid || msg.ToID == uid {
		return true
	}
	var n int64
	f.S.DB.Model(&model.Conversation{}).Where("conv_id = ? AND owner_uid = ?", msg.ConvID, uid).Count(&n)
	return n > 0
}

func (f *FavoriteService) vo(fav model.Favorite) *FavoriteVO {
	vo := &FavoriteVO{Favorite: fav}
	var c model.Conversation
	if f.S.DB.Where("conv_id = ? AND owner_uid = ?", fav.ConvID, fav.UID).First(&c).Error == nil {
		vo.ConvName = c.PeerName
	}
	return vo
}

// List 返回收藏列表（按收藏时间倒序）
func (f *FavoriteService) List(uid string) []FavoriteVO {
	var rows []model.Favorite
	f.S.DB.Where("uid = ?", uid).Order("created_at desc").Find(&rows)
	out := make([]FavoriteVO, 0, len(rows))
	for _, r := range rows {
		out = append(out, *f.vo(r))
	}
	return out
}

// Remove 取消收藏
func (f *FavoriteService) Remove(uid string, id int64) error {
	res := f.S.DB.Where("id = ? AND uid = ?", id, uid).Delete(&model.Favorite{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("收藏不存在")
	}
	return nil
}
