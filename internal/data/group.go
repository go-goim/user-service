package data

import (
	grouppb "github.com/go-goim/api/user/group/v1"
	"github.com/go-goim/core/pkg/types"
)

// Group is the model of group table based on gorm, which contains group basic info.
// Group data stored in mysql.
type Group struct {
	ID          uint64              `gorm:"primary_key"`
	GID         *types.ID           `gorm:"column:gid"`
	Name        string              `gorm:"column:name"`
	Description string              `gorm:"column:description"`
	Avatar      string              `gorm:"column:avatar"`
	MaxMembers  int                 `gorm:"column:max_members"`
	MemberCount int                 `gorm:"column:member_count"`
	Status      grouppb.GroupStatus `gorm:"column:status"`
	OwnerUID    *types.ID           `gorm:"column:owner_uid"`
	CreatedAt   int64               `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   int64               `gorm:"column:updated_at;autoUpdateTime"`
}

func (Group) TableName() string {
	return "group"
}

func (g *Group) IsSilent() bool {
	return g.Status == grouppb.GroupStatus_Silent
}

func (g *Group) ToProto() *grouppb.Group {
	return &grouppb.Group{
		Gid:         g.GID.Int64(),
		Name:        g.Name,
		Description: g.Description,
		Avatar:      g.Avatar,
		MaxMembers:  int32(g.MaxMembers),
		MemberCount: int32(g.MemberCount),
		Status:      g.Status,
	}
}
