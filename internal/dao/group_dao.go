package dao

import (
	"context"
	"sync"

	redisv8 "github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/user-service/internal/app"
	"github.com/go-goim/user-service/internal/data"
)

type GroupDao struct {
	rdb *redisv8.Client
}

var (
	groupDao     *GroupDao
	groupDaoOnce sync.Once
)

func GetGroupDao() *GroupDao {
	groupDaoOnce.Do(func() {
		groupDao = &GroupDao{
			rdb: app.GetApplication().Redis,
		}
	})
	return groupDao
}

func (d *GroupDao) GetGroup(ctx context.Context, id int64) (*data.Group, error) {
	group := &data.Group{}
	tx := db.GetDBFromCtx(ctx).Where("id = ?", id).First(group)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, tx.Error
	}

	return group, nil
}

func (d *GroupDao) GetGroupByGID(ctx context.Context, gid string) (*data.Group, error) {
	group := &data.Group{}
	tx := db.GetDBFromCtx(ctx).Where("gid = ?", gid).First(group)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, tx.Error
	}

	return group, nil
}

func (d *GroupDao) ListGroups(ctx context.Context, gids []string) ([]*data.Group, error) {
	groups := make([]*data.Group, 0)
	tx := db.GetDBFromCtx(ctx).Where("gid in (?)", gids).Find(&groups)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return groups, nil
}

func (d *GroupDao) CreateGroup(ctx context.Context, group *data.Group) error {
	tx := db.GetDBFromCtx(ctx).Create(group)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *GroupDao) UpdateGroup(ctx context.Context, group *data.Group) error {
	tx := db.GetDBFromCtx(ctx).Save(group)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *GroupDao) DeleteGroup(ctx context.Context, group *data.Group) error {
	tx := db.GetDBFromCtx(ctx).Delete(group)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// IncrGroupMemberCount incr group member count by given increase.
// It will check if after increased group member count is greater than max group member count,
// if so, it will return false.
func (d *GroupDao) IncrGroupMemberCount(ctx context.Context, g *data.Group, increase uint) (bool, error) {
	tx := db.GetDBFromCtx(ctx).Model(&data.Group{}).Where("gid = ?", g.GID).
		Where("member_count + ? <= ?", increase, g.MaxMembers).
		Update("member_count", gorm.Expr("member_count + ?", increase))
	if tx.Error != nil {
		// not found means group member count is greater than max group member count,
		//  because we have already if group exists
		if tx.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, tx.Error
	}

	return true, nil
}

// DecrGroupMemberCount decr group member count by given decrease.
// It will check if after decreased group member count is less than 0,
// if so, it will return false.
func (d *GroupDao) DecrGroupMemberCount(ctx context.Context, g *data.Group, decrease uint) (bool, error) {
	tx := db.GetDBFromCtx(ctx).Model(&data.Group{}).Where("gid = ?", g.GID).
		Where("member_count - ? >= 0", decrease).
		Update("member_count", gorm.Expr("member_count - ?", decrease))
	if tx.Error != nil {
		// not found means group member count is less than 0, because we have already if group exists
		if tx.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, tx.Error
	}

	return true, nil
}
