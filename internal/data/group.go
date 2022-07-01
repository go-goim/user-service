package data

import (
	grouppb "github.com/go-goim/api/user/group/v1"
)

// Group is the model of group table based on gorm, which contains group basic info.
// Group data stored in mysql.
type Group struct {
	ID          int64               `gorm:"primary_key"`
	GID         string              `gorm:"type:varchar(64);unique_index;not null;column:gid"`
	Name        string              `gorm:"type:varchar(32);not null"`
	Description string              `gorm:"type:varchar(128);not null"`
	Avatar      string              `gorm:"type:varchar(128);not null"`
	MaxMembers  int                 `gorm:"type:int(11);not null"`
	MemberCount int                 `gorm:"type:int(11);not null"`
	Status      grouppb.GroupStatus `gorm:"type:tinyint(1);not null"`
	OwnerUID    string              `gorm:"type:varchar(64);not null;column:owner_uid"`
	CreatedAt   int64               `gorm:"type:bigint(20);not null;autoCreateTime"`
	UpdatedAt   int64               `gorm:"type:bigint(20);not null;autoUpdateTime"`
}

func (Group) TableName() string {
	return "group"
}

func (g *Group) IsSilent() bool {
	return g.Status == grouppb.GroupStatus_Silent
}

func (g *Group) ToProto() *grouppb.Group {
	return &grouppb.Group{
		Id:          g.ID,
		Gid:         g.GID,
		Name:        g.Name,
		Description: g.Description,
		Avatar:      g.Avatar,
		MaxMembers:  int32(g.MaxMembers),
		MemberCount: int32(g.MemberCount),
		Status:      g.Status,
	}
}
