package data

import (
	grouppb "github.com/go-goim/api/user/group/v1"
)

// GroupMember is the model of group_member table based on gorm, which contains group member info.
// GroupMember data stored in mysql.
type GroupMember struct {
	ID        int64                      `gorm:"primary_key"`
	GID       string                     `gorm:"type:varchar(64);not null"`
	UID       string                     `gorm:"type:varchar(64);not null"`
	Type      grouppb.GroupMember_Type   `gorm:"type:tinyint(1);not null"`
	Status    grouppb.GroupMember_Status `gorm:"type:tinyint(1);not null"`
	CreatedAt int64                      `gorm:"type:bigint(20);not null;autoCreateTime"`
	UpdatedAt int64                      `gorm:"type:bigint(20);not null;autoUpdateTime"`
}

// TODO: maybe should use Group.ID instead of GID and User.ID instead of UID

func (GroupMember) TableName() string {
	return "group_member"
}

func (g *GroupMember) ToProto() *grouppb.GroupMember {
	return &grouppb.GroupMember{
		Id:     g.ID,
		Gid:    g.GID,
		Uid:    g.UID,
		Type:   g.Type,
		Status: g.Status,
	}
}
