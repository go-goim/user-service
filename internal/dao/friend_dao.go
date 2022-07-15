package dao

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	friendpb "github.com/go-goim/api/user/friend/v1"
	"github.com/go-goim/core/pkg/cache"
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/log"
	"github.com/go-goim/core/pkg/types"

	"github.com/go-goim/user-service/internal/data"
)

type FriendDao struct{}

var (
	friendDao     *FriendDao
	friendDaoOnce sync.Once
)

func GetUserRelationDao() *FriendDao {
	friendDaoOnce.Do(func() {
		friendDao = &FriendDao{}
	})
	return friendDao
}

func (d *FriendDao) GetFriend(ctx context.Context, uid, friendUID *types.ID) (*data.Friend, error) {
	userRelation := &data.Friend{}
	err := db.GetDBFromCtx(ctx).Where("uid = ? AND friend_uid = ?", uid, friendUID).First(userRelation).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return userRelation, nil
}

func (d *FriendDao) GetFriendByStatus(ctx context.Context, uid, friendUID *types.ID, status int) (*data.Friend, error) {
	ur := new(data.Friend)
	err := db.GetDBFromCtx(ctx).Where("uid = ? AND friend_uid = ? AND status = ?", uid, friendUID, status).
		First(&ur).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return ur, nil
}

func (d *FriendDao) CheckIsFriend(ctx context.Context, uid, friendUID *types.ID) (bool, error) {
	// check from cache
	ok, err := d.GetFriendStatusFromCache(ctx, uid, friendUID)
	if err != nil {
		return false, err
	}

	if ok {
		return ok, nil
	}

	// check from db

	ur, err := d.GetFriendByStatus(ctx, uid, friendUID, int(friendpb.FriendStatus_FRIEND))
	if err != nil {
		return false, err
	}

	if ur == nil {
		return false, nil
	}

	ur2, err := d.GetFriendByStatus(ctx, friendUID, uid, int(friendpb.FriendStatus_FRIEND))
	if err != nil {
		return false, err
	}

	if ur2 == nil {
		return false, nil
	}

	// set cache
	err = d.SetFriendStatusToCache(ctx, uid, friendUID)
	if err != nil {
		log.Error("set friend status to cache error", "err", err)
	}

	return true, nil
}

func friendStatusKey(uid, friendUID *types.ID) string {
	if uid.Int64() > friendUID.Int64() {
		return fmt.Sprintf("friend_status:%d:%d", friendUID.Int64(), uid.Int64())
	}
	return fmt.Sprintf("friend_status:%d:%d", uid.Int64(), friendUID.Int64())
}

// GetFriendStatusFromCache get friend status from cache.
// cache key: sort(uid, friend_uid), so that there is no duplicated key, only one record between two users.
// cache value: 1 as constant.
func (d *FriendDao) GetFriendStatusFromCache(ctx context.Context, uid, friendUID *types.ID) (bool, error) {
	_, err := cache.Get(ctx, friendStatusKey(uid, friendUID))
	if err != nil {
		if err == cache.ErrCacheMiss {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// SetFriendStatusToCache set friend status to cache.
func (d *FriendDao) SetFriendStatusToCache(ctx context.Context, uid, friendUID *types.ID) error {
	return cache.Set(ctx, friendStatusKey(uid, friendUID), []byte("1"), 0) // 0 means no expire time.
}

func (d *FriendDao) DeleteFriendStatusFromCache(ctx context.Context, uid, friendUID *types.ID) error {
	return cache.Delete(ctx, friendStatusKey(uid, friendUID))
}

func (d *FriendDao) GetFriends(ctx context.Context, uid *types.ID) ([]*data.Friend, error) {
	userRelationList := make([]*data.Friend, 0)
	err := db.GetDBFromCtx(ctx).Where("uid = ?", uid).Order("id").Find(&userRelationList).Error
	if err != nil {
		return nil, err
	}

	return userRelationList, nil
}

func (d *FriendDao) CreateFriend(ctx context.Context, friend *data.Friend) error {
	friend.CreatedAt = time.Now().Unix()
	friend.UpdatedAt = time.Now().Unix()

	return db.GetDBFromCtx(ctx).Create(friend).Error
}

func (d *FriendDao) UpdateFriendStatus(ctx context.Context, userRelation *data.Friend) error {
	tx := db.GetDBFromCtx(ctx).Model(userRelation).Updates(map[string]interface{}{
		"updated_at": time.Now().Unix(),
		"status":     userRelation.Status,
	})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *FriendDao) CountFriends(ctx context.Context, uid *types.ID) (int64, error) {
	var count int64
	err := db.GetDBFromCtx(ctx).Model(&data.Friend{}).Where("uid = ?", uid).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
