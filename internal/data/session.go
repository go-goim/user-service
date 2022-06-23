package data

import (
	sessionpb "github.com/go-goim/api/user/session/v1"
)

// Session is represents a conversation between user-user or user-group.
// If the session is single chat, the from_user_id is less than to_user_id.
// If the session is group chat, the from_user_id and to_user_id are the both group id.
type Session struct {
	ID         int64                 `gorm:"primary_key"` // session id
	FromUserID string                `gorm:"type:varchar(64);not null"`
	ToUserID   string                `gorm:"type:varchar(64);not null"`
	Type       sessionpb.SessionType `gorm:"type:tinyint(1);not null"`  // 0: single chat, 1: group chat
	CreatedBy  string                `gorm:"type:varchar(64);not null"` // user id who created the session
	CreatedAt  int64                 `gorm:"type:bigint(20);not null;autoCreateTime"`
	UpdatedAt  int64                 `gorm:"type:bigint(20);not null;autoUpdateTime"`
}

func (Session) TableName() string {
	return "session"
}

func (s *Session) IsSingleChat() bool {
	return s.Type == sessionpb.SessionType_SingleChat
}
