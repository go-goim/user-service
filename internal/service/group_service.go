package service

import (
	"context"
	"fmt"
	"sync"

	responsepb "github.com/go-goim/api/transport/response"
	grouppb "github.com/go-goim/api/user/group/v1"
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/util"
	"github.com/go-goim/user-service/internal/dao"
	"github.com/go-goim/user-service/internal/data"
)

type GroupService struct {
	groupDao       *dao.GroupDao
	groupMemberDao *dao.GroupMemberDao
	userDao        *dao.UserDao

	grouppb.UnimplementedGroupServiceServer
}

var (
	_                grouppb.GroupServiceServer = &GroupService{}
	groupService     *GroupService
	groupServiceOnce sync.Once
)

func GetGroupService() *GroupService {
	groupServiceOnce.Do(func() {
		groupService = &GroupService{
			groupDao:       dao.GetGroupDao(),
			groupMemberDao: dao.GetGroupMemberDao(),
			userDao:        dao.GetUserDao(),
		}
	})
	return groupService
}

func (s *GroupService) GetGroup(ctx context.Context, req *grouppb.GetGroupRequest) (*grouppb.GetGroupResponse, error) {
	rsp := &grouppb.GetGroupResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	group, err := s.groupDao.GetGroupByGID(ctx, req.GetGid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Response = responsepb.Code_GroupNotExist.BaseResponse()
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	if !req.GetWithMembers() {
		return rsp, nil
	}

	gmList, err := s.groupMemberDao.ListGroupMembersByGID(ctx, req.GetGid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	var (
		uidList []string
		gmMap   = make(map[string]*grouppb.GroupMember)
	)
	for _, gm := range gmList {
		uidList = append(uidList, gm.UID)
		gmMap[gm.UID] = gm.ToProto()
	}

	userList, err := s.userDao.ListUsers(ctx, uidList...)
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	for _, u := range userList {
		gm := &grouppb.GroupMember{
			Gid:  group.GID,
			Uid:  u.UID,
			User: u.ToProto(),
		}

		if temp, ok := gmMap[u.UID]; ok {
			gm.Type = temp.Type
			gm.Status = temp.Status
		}

		rsp.Group.Members = append(rsp.Group.Members, gm)
		if u.UID == group.OwnerUID {
			rsp.Group.Owner = gm
		}
	}

	return rsp, nil
}

func (s *GroupService) ListGroups(ctx context.Context, req *grouppb.ListGroupsRequest) (*grouppb.ListGroupsResponse, error) {
	rsp := &grouppb.ListGroupsResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	gmList, err := s.groupMemberDao.ListGroupByUID(ctx, req.GetUid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	var gidList []string
	for _, gm := range gmList {
		gidList = append(gidList, gm.GID)
	}

	groupList, err := s.groupDao.ListGroups(ctx, gidList)
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	for _, g := range groupList {
		rsp.Groups = append(rsp.Groups, g.ToProto())
	}

	return rsp, nil
}

func (s *GroupService) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	group := &data.Group{
		GID:         fmt.Sprintf("g_%s", util.UUID()),
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Avatar:      req.GetAvatar(),
		OwnerUID:    req.GetOwnerUid(),
		MaxMembers:  500, // todo: use config
		MemberCount: len(req.GetMembersUid()) + 1,
	}

	var members = make([]*data.GroupMember, 0, len(req.GetMembersUid())+1)

	members = append(members, &data.GroupMember{
		GID:  group.GID,
		UID:  group.OwnerUID,
		Type: grouppb.GroupMember_TypeOwner,
	})

	for _, uid := range req.GetMembersUid() {
		members = append(members, &data.GroupMember{
			GID:  group.GID,
			UID:  uid,
			Type: grouppb.GroupMember_TypeMember,
		})
	}

	rsp := &grouppb.CreateGroupResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	err := db.Transaction(ctx, func(ctx2 context.Context) error {
		if err := s.groupDao.CreateGroup(ctx2, group); err != nil {
			return err
		}

		if err := s.groupMemberDao.CreateGroupMember(ctx2, members...); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	return rsp, nil
}

func (s *GroupService) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	rsp := &grouppb.UpdateGroupResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	group, err := s.groupDao.GetGroupByGID(ctx, req.GetGid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Response = responsepb.Code_GroupNotExist.BaseResponse()
		return rsp, nil
	}

	if group.OwnerUID != req.GetUid() {
		rsp.Response = responsepb.Code_NotGroupOwner.BaseResponse()
		return rsp, nil
	}

	if req.GetName() != "" {
		group.Name = req.GetName()
	}

	if req.GetDescription() != "" {
		group.Description = req.GetDescription()
	}

	if req.GetAvatar() != "" {
		group.Avatar = req.GetAvatar()
	}

	err = s.groupDao.UpdateGroup(ctx, group)
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	return rsp, nil
}

func (s *GroupService) DeleteGroup(ctx context.Context, req *grouppb.DeleteGroupRequest) (*responsepb.BaseResponse, error) {
	rsp := responsepb.Code_OK.BaseResponse()

	group, err := s.groupDao.GetGroupByGID(ctx, req.GetGid())
	if err != nil {
		rsp = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp = responsepb.Code_GroupNotExist.BaseResponse()
		return rsp, nil
	}

	if group.OwnerUID != req.GetUid() {
		rsp = responsepb.Code_NotGroupOwner.BaseResponse()
		return rsp, nil
	}

	err = s.groupDao.DeleteGroup(ctx, group)
	if err != nil {
		rsp = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	return rsp, nil
}

func (s *GroupService) AddGroupMember(ctx context.Context, req *grouppb.AddGroupMemberRequest) (
	*grouppb.AddGroupMemberResponse, error) {
	// todo: should limit the count of adding members, like only add 10 members per time.
	//  cause the add member operation is a heavy operation.
	rsp := &grouppb.AddGroupMemberResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	group, err := s.groupDao.GetGroupByGID(ctx, req.GetGid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Response = responsepb.Code_GroupNotExist.BaseResponse()
		return rsp, nil
	}

	uids, err := s.groupMemberDao.ListInGroupUIDs(ctx, group.GID, req.GetUid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	// if all users are already in the group, return
	if len(uids) == len(req.GetUid()) {
		return rsp, nil
	}

	// filter out the users already in the group
	var (
		inGroupUIDs = util.NewSet[string]().Add(uids...)
		newUIDs     []string
	)

	for _, uid := range req.GetUid() {
		if !inGroupUIDs.Contains(uid) {
			newUIDs = append(newUIDs, uid)
		}
	}

	// check if new users can add to the group, because the group has max member limit
	if len(newUIDs)+group.MemberCount > group.MaxMembers {
		rsp.Response = responsepb.Code_GroupLimitExceed.BaseResponse()
		return rsp, nil
	}

	var (
		gmList = make([]*data.GroupMember, len(newUIDs))
	)

	for i, uid := range newUIDs {
		gmList[i] = &data.GroupMember{
			GID:  group.GID,
			UID:  uid,
			Type: grouppb.GroupMember_TypeMember,
		}
	}

	// create group members and increase the member count
	err = db.Transaction(ctx, func(ctx2 context.Context) error {
		// increase the member count first
		success, err := s.groupDao.IncrGroupMemberCount(ctx2, group, uint(len(newUIDs)))
		if err != nil {
			return err
		}

		if !success {
			return responsepb.Code_GroupLimitExceed.BaseResponse()
		}

		if err := s.groupMemberDao.CreateGroupMember(ctx2, gmList...); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	rsp.Added = int32(len(newUIDs))
	return rsp, nil
}

func (s *GroupService) RemoveGroupMember(ctx context.Context, req *grouppb.RemoveGroupMemberRequest) (
	*grouppb.RemoveGroupMemberResponse, error) {
	rsp := &grouppb.RemoveGroupMemberResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	group, err := s.groupDao.GetGroupByGID(ctx, req.GetGid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Response = responsepb.Code_GroupNotExist.BaseResponse()
		return rsp, nil
	}

	uids, err := s.groupMemberDao.ListInGroupUIDs(ctx, group.GID, req.GetUid())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	// if all users are not in the group, return
	if len(uids) == 0 {
		return rsp, nil
	}

	// filter out the users not in the group
	var (
		inGroupUIDs    = util.NewSet[string]().Add(uids...)
		needRemoveUIDs []string
	)

	for _, uid := range req.GetUid() {
		if inGroupUIDs.Contains(uid) {
			needRemoveUIDs = append(needRemoveUIDs, uid)
		}
	}

	// delete group members and decrease the member count
	err = db.Transaction(ctx, func(ctx2 context.Context) error {
		// decrease the member count first
		success, err := s.groupDao.DecrGroupMemberCount(ctx2, group, uint(len(needRemoveUIDs)))
		if err != nil {
			return err
		}

		if !success {
			return responsepb.Code_GroupLimitExceed.BaseResponse()
		}

		if err := s.groupMemberDao.DeleteGroupMembers(ctx2, group.GID, needRemoveUIDs); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	rsp.Removed = int32(len(needRemoveUIDs))
	return rsp, nil
}

func (s *GroupService) isInGroup(ctx context.Context, gid, uid string) (bool, error) {
	// todo: check cache first
	gm, err := s.groupMemberDao.GetGroupMemberByGIDUID(ctx, gid, uid)
	if err != nil {
		return false, err
	}

	return gm != nil, nil
}
