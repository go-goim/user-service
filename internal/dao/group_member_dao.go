package dao

import (
	"context"
	"sync"

	"gorm.io/gorm"

	"github.com/go-goim/core/pkg/db"
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
