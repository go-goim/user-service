package data

import (
	grouppb "github.com/go-goim/api/user/group/v1"
	"github.com/go-goim/core/pkg/types"
)

// GroupMember is the model of group_member table based on gorm, which contains group member info.
// GroupMember data stored in mysql.
type GroupMember struct {
	ID        uint64                     `gorm:"primary_key"`
	GID       types.ID                   `gorm:"column:gid"`
	UID       types.ID                   `gorm:"column:uid"`
	Type      grouppb.GroupMember_Type   `gorm:"column:type"`
	Status    grouppb.GroupMember_Status `gorm:"column:status"`
	CreatedAt int64                      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt int64                      `gorm:"column:updated_at;autoUpdateTime"`
}

// TODO: maybe should use Group.ID instead of GID and User.ID instead of UID

func (GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) ToProto() *grouppb.GroupMember {
	return &grouppb.GroupMember{
		Gid:    g.GID.Int64(),
		Uid:    g.UID.Int64(),
		Type:   g.Type,
		Status: g.Status,
	}
}
