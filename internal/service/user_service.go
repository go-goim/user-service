package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-goim/api/errors"
	userv1 "github.com/go-goim/api/user/v1"
	"github.com/go-goim/core/pkg/types"

	"github.com/go-goim/core/pkg/util"

	"github.com/go-goim/user-service/internal/dao"
	"github.com/go-goim/user-service/internal/data"
)

// UserService implements userv1.UserServiceServer
type UserService struct {
	userDao *dao.UserDao
	userv1.UnimplementedUserServiceServer
}

var (
	_               userv1.UserServiceServer = &UserService{}
	userService     *UserService
	userServiceOnce sync.Once
)

func GetUserService() *UserService {
	userServiceOnce.Do(func() {
		userService = &UserService{
			userDao: dao.GetUserDao(),
		}
	})
	return userService
}

func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserInfoRequest) (*userv1.UserResponse, error) {
	user, err := s.userDao.GetUserByUID(ctx, types.ID(req.Uid))
	if err != nil {
		return nil, err
	}

	rsp := &userv1.UserResponse{
		Error: errors.ErrorOK(),
	}

	if user == nil || user.IsDeleted() {
		rsp.Error = errors.NewErrorWithCode(errors.ErrorCode_UserNotExist)
		return rsp, nil
	}

	rsp.User = user.ToProto()
	return rsp, nil
}

func (s *UserService) QueryUser(ctx context.Context, req *userv1.QueryUserRequest) (*userv1.UserResponse, error) {
	user, err := s.loadUserByEmailOrPhone(ctx, req.GetEmail(), req.GetPhone())
	if err != nil {
		return nil, err
	}

	rsp := &userv1.UserResponse{
		Error: errors.ErrorOK(),
	}

	if user == nil || user.IsDeleted() {
		rsp.Error = errors.NewErrorWithCode(errors.ErrorCode_UserNotExist)
		return rsp, nil
	}

	rsp.User = user.ToProto()
	return rsp, nil
}

func (s *UserService) loadUserByEmailOrPhone(ctx context.Context, email, phone string) (*data.User, error) {
	var (
		value   string
		getFunc func(ctx context.Context, v string) (*data.User, error)
	)

	switch {
	case email != "":
		value = email
		getFunc = s.userDao.GetUserByEmail
	case phone != "":
		value = phone
		getFunc = s.userDao.GetUserByPhone
	default:
		return nil, fmt.Errorf("invalid query user request, email: %s, phone: %s", email, phone)
	}

	user, err := getFunc(ctx, value)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.UserResponse, error) {
	user, err := s.loadUserByEmailOrPhone(ctx, req.GetEmail(), req.GetPhone())
	if err != nil {
		return nil, err
	}

	rsp := &userv1.UserResponse{
		Error: errors.ErrorOK(),
	}

	if user == nil {
		user = &data.User{
			UID:      types.NewID(),
			Name:     req.GetName(),
			Password: util.HashString(req.GetPassword()),
		}

		user.SetPhone(req.GetPhone())
		user.SetEmail(req.GetEmail())

		err = s.userDao.CreateUser(ctx, user)
		if err != nil {
			return nil, err
		}

		rsp.User = user.ToProto()
		return rsp, nil
	}

	// user exists
	if user.IsDeleted() {
		// undo delete
		// 这里会出现被删除用户 undo 后使用的是旧密码的情况,需要更新密码
		user.Password = util.HashString(req.GetPassword())
		err = s.userDao.UndoDelete(ctx, user)
		if err != nil {
			return nil, err
		}

		rsp.User = user.ToProto()
		return rsp, nil
	}

	rsp.Error = errors.NewErrorWithCode(errors.ErrorCode_UserExist)
	return rsp, nil

}

func (s *UserService) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UserResponse, error) {
	user, err := s.userDao.GetUserByUID(ctx, types.ID(req.Uid))
	if err != nil {
		return nil, err
	}

	rsp := &userv1.UserResponse{
		Error: errors.ErrorOK(),
	}

	if user == nil || user.IsDeleted() {
		rsp.Error = errors.NewErrorWithCode(errors.ErrorCode_UserNotExist)
		return rsp, nil
	}

	user.SetEmail(req.GetEmail())
	user.SetPhone(req.GetPhone())

	if req.GetName() != "" {
		user.Password = req.GetName()
	}

	if req.GetAvatar() != "" {
		user.Avatar = req.GetAvatar()
	}

	err = s.userDao.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	rsp.User = user.ToProto()
	return rsp, nil
}
