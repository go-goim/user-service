package data

import (
	friendpb "github.com/go-goim/api/user/friend/v1"
	"github.com/go-goim/core/pkg/types"
)

// Friend is the model of user relation table based on gorm, which is used for user relation management.
// Friend data stored in mysql.
type Friend struct {
	ID uint64 `gorm:"primary_key"`
	// UID is the user uid of the user.
	UID types.ID `gorm:"column:uid"`
	// FriendUID is the user uid of the friend.
	FriendUID types.ID `gorm:"column:friend_uid"`
	// Status is the status of the relation.
	Status friendpb.FriendStatus `gorm:"column:status"`
	// CreatedAt is the creation time of the relation.
	CreatedAt int64 `gorm:"column:created_at;autoCreateTime"`
	// UpdatedAt is the update time of the relation.
	UpdatedAt int64 `gorm:"column:updated_at;autoUpdateTime"`
}

func (Friend) TableName() string {
	return "friend"
}

const (
	UserMaxFriendCount = 2000 // UserMaxRelationCount is the max count of user relation.
)

func (ur *Friend) IsFriend() bool {
	return ur.Status == friendpb.FriendStatus_FRIEND
}

func (ur *Friend) IsStranger() bool {
	return ur.Status == friendpb.FriendStatus_STRANGER
}

func (ur *Friend) IsBlocked() bool {
	return ur.Status == friendpb.FriendStatus_BLOCKED
}

func (ur *Friend) SetStatus(status friendpb.FriendStatus) bool {
	if ur.Status.CanUpdateStatus(status) {
		ur.Status = status
		return true
	}

	return false
}

func (ur *Friend) SetFriend() bool {
	return ur.SetStatus(friendpb.FriendStatus_FRIEND)
}

func (ur *Friend) SetStranger() bool {
	return ur.SetStatus(friendpb.FriendStatus_STRANGER)
}

func (ur *Friend) SetBlocked() bool {
	return ur.SetStatus(friendpb.FriendStatus_BLOCKED)
}

func (ur *Friend) ToProtoFriend() *friendpb.Friend {
	return &friendpb.Friend{
		Uid:       ur.UID.Int64(),
		FriendUid: ur.FriendUID.Int64(),
		Status:    ur.Status,
		CreatedAt: ur.CreatedAt,
		UpdatedAt: ur.UpdatedAt,
	}
}
