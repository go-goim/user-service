package dao

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"gorm.io/gorm"

	grouppb "github.com/go-goim/api/user/group/v1"
	"github.com/go-goim/core/pkg/cache"
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/log"
	"github.com/go-goim/user-service/internal/data"
)

type GroupMemberDao struct{}

var (
	groupMemberDao     *GroupMemberDao
	groupMemberDaoOnce sync.Once
)

func GetGroupMemberDao() *GroupMemberDao {
	groupMemberDaoOnce.Do(func() {
		groupMemberDao = &GroupMemberDao{}
	})
	return groupMemberDao
}

func (d *GroupMemberDao) GetGroupMember(ctx context.Context, id int64) (*data.GroupMember, error) {
	groupMember := &data.GroupMember{}
	tx := db.GetDBFromCtx(ctx).Where("id = ?", id).First(groupMember)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, tx.Error
	}

	return groupMember, nil
}

func (d *GroupMemberDao) IsMemberOfGroup(ctx context.Context, gid, uid string) (*data.GroupMember, error) {
	// load from cache
	status, err := d.IsMemberOfGroupFromCache(ctx, gid, uid)
	if err != nil && err != cache.ErrCacheMiss {
		return nil, err
	}

	if err == nil {
		return &data.GroupMember{
			GID:    gid,
			UID:    uid,
			Status: grouppb.GroupMember_Status(status),
		}, nil
	}

	gm := &data.GroupMember{}
	tx := db.GetDBFromCtx(ctx).Where("gid = ? AND uid = ?", gid, uid).First(gm)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, tx.Error
	}

	if err := d.SetMemberStatusToCache(ctx, gid, uid, int(gm.Status)); err != nil {
		log.Error("SetMemberStatusToCache", "gid", gid, "uid", uid, "err", err)
	}

	return gm, nil
}

// IsMemberOfGroupFromCache return member status in group of given uid. 0: active , 1: silent
func (d *GroupMemberDao) IsMemberOfGroupFromCache(ctx context.Context, gid, uid string) (int, error) {
	key := fmt.Sprintf("group_members_%s", gid)
	b, err := cache.GetFromHash(ctx, key, uid)
	if err != nil {
		return 0, err
	}

	i, _ := strconv.Atoi(string(b)) // nolint: errcheck
	return i, nil
}

func (d *GroupMemberDao) SetMemberStatusToCache(ctx context.Context, gid, uid string, status int) error {
	key := fmt.Sprintf("group_members_%s", gid)
	return cache.SetToHash(ctx, key, uid, []byte(strconv.Itoa(status)))
}

func (d *GroupMemberDao) GetGroupMemberByGIDUID(ctx context.Context, gid, uid string) (*data.GroupMember, error) {
	groupMember := &data.GroupMember{}
	tx := db.GetDBFromCtx(ctx).Where("gid = ? AND uid = ?", gid, uid).First(groupMember)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, tx.Error
	}

	return groupMember, nil
}

func (d *GroupMemberDao) ListGroupMembersByGID(ctx context.Context, gid string) ([]*data.GroupMember, error) {
	groupMembers := make([]*data.GroupMember, 0)
	tx := db.GetDBFromCtx(ctx).Where("gid = ?", gid).Find(&groupMembers)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return groupMembers, nil
}

func (d *GroupMemberDao) ListGroupByUID(ctx context.Context, uid string) ([]*data.GroupMember, error) {
	groupMembers := make([]*data.GroupMember, 0)
	tx := db.GetDBFromCtx(ctx).Where("uid = ?", uid).Find(&groupMembers)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return groupMembers, nil
}

// ListInGroupUIDs returns uid list in group of given uids,
func (d *GroupMemberDao) ListInGroupUIDs(ctx context.Context, gid string, uids []string) ([]string, error) {
	result := make([]string, 0)
	tx := db.GetDBFromCtx(ctx).Table("group_member").Select("uid").
		Where("gid = ? AND uid in (?)", gid, uids).Find(&result)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return result, nil
}

func (d *GroupMemberDao) CreateGroupMember(ctx context.Context, groupMember ...*data.GroupMember) error {
	tx := db.GetDBFromCtx(ctx).CreateInBatches(groupMember, len(groupMember))
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *GroupMemberDao) DeleteGroupMember(ctx context.Context, groupMember *data.GroupMember) error {
	tx := db.GetDBFromCtx(ctx).Delete(groupMember)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (d *GroupMemberDao) DeleteGroupMembers(ctx context.Context, gid string, uids []string) error {
	tx := db.GetDBFromCtx(ctx).Where("gid = ? AND uid in (?)", gid, uids).Delete(&data.GroupMember{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
