package data

import (
	userv1 "github.com/go-goim/api/user/v1"
	"github.com/go-goim/core/pkg/types"
)

// User is the model of user table based on gorm, which contains user basic info.
// User data stored in mysql.
type User struct {
	ID        uint64   `gorm:"primary_key"`
	UID       types.ID `gorm:"column:uid"`
	Name      string   `gorm:"column:name"`
	Password  string   `gorm:"column:password"`
	Email     *string  `gorm:"column:email"`
	Phone     *string  `gorm:"column:phone"`
	Avatar    string   `gorm:"column:avatar"`
	Status    int      `gorm:"column:status"`
	CreatedAt int64    `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt int64    `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string {
	return "user"
}

const (
	UserStatusNormal int = iota
	UserStatusDeleted
)

const (
	UserCacheExpire = 60 * 60 * 24 // 1 day
)

func (u *User) IsDeleted() bool {
	return u.Status == UserStatusDeleted
}

func (u *User) SetEmail(email string) {
	if email == "" {
		return
	}
	u.Email = &email
}

func (u *User) SetPhone(phone string) {
	if phone == "" {
		return
	}
	u.Phone = &phone
}

func (u *User) ToProto() *userv1.User {
	return &userv1.User{
		Uid:      u.UID.Int64(),
		Name:     u.Name,
		Password: u.Password,
		Email:    u.Email,
		Phone:    u.Phone,
		Avatar:   u.Avatar,
	}
}
