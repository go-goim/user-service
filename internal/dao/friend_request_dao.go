package dao

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/types"

	"github.com/go-goim/user-service/internal/data"
)

type FriendRequestDao struct {
}

var (
	friendRequestDao     *FriendRequestDao
	friendRequestDaoOnce sync.Once
)

func GetFriendRequestDao() *FriendRequestDao {
	friendRequestDaoOnce.Do(func() {
		friendRequestDao = &FriendRequestDao{}
	})
	return friendRequestDao
}

func (d *FriendRequestDao) CreateFriendRequest(ctx context.Context, fr *data.FriendRequest) error {
	fr.CreatedAt = time.Now().Unix()
	fr.UpdatedAt = time.Now().Unix()
	// get db from context
	return db.GetDBFromCtx(ctx).Create(fr).Error
}

func (d *FriendRequestDao) GetFriendRequest(ctx context.Context, uid, friendUID types.ID) (*data.FriendRequest, error) {
	var fr data.FriendRequest
	if err := db.GetDBFromCtx(ctx).Where("uid = ? AND friend_uid = ?", uid, friendUID).First(&fr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &fr, nil
}

func (d *FriendRequestDao) GetFriendRequestByID(ctx context.Context, id uint64) (*data.FriendRequest, error) {
	var fr data.FriendRequest
	if err := db.GetDBFromCtx(ctx).Where("id = ?", id).First(&fr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &fr, nil
}

func (d *FriendRequestDao) GetFriendRequests(ctx context.Context, uid types.ID, status int) ([]*data.FriendRequest, error) {
	var frs []*data.FriendRequest
	// query friend request send to me.
	if err := db.GetDBFromCtx(ctx).Where("friend_uid = ? AND status = ?", uid, status).Find(&frs).Error; err != nil {
		return nil, err
	}

	return frs, nil
}

func (d *FriendRequestDao) UpdateFriendRequest(ctx context.Context, fr *data.FriendRequest) error {
	return db.GetDBFromCtx(ctx).Model(fr).UpdateColumns(map[string]interface{}{
		"status":     fr.Status,
		"updated_at": fr.UpdatedAt,
	}).Error
}
