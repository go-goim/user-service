package service

import (
	"context"
	"sync"

	"github.com/go-goim/api/errors"
	grouppb "github.com/go-goim/api/user/group/v1"
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/types"
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
		Error: errors.ErrorOK(),
	}

	var (
		gid      = types.ID(req.Gid)
		ownerUID = types.ID(req.OwnerUid)
	)

	group, err := s.groupDao.GetGroupByGID(ctx, gid)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Error = errors.ErrorCode_GroupNotExist.Err2()
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	if !req.GetWithMembers() {
		return rsp, nil
	}

	gmList, err := s.groupMemberDao.ListGroupMembersByGID(ctx, ownerUID)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	var (
		uidList = make([]types.ID, len(gmList))
		gmMap   = make(map[int64]*grouppb.GroupMember)
	)
	for i, gm := range gmList {
		uidList[i] = gm.UID
		gmMap[gm.UID.Int64()] = gm.ToProto()
	}

	userList, err := s.userDao.ListUsers(ctx, uidList...)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	for _, u := range userList {
		gm := &grouppb.GroupMember{
			Gid:  group.GID.Int64(),
			Uid:  u.UID.Int64(),
			User: u.ToProto(),
		}

		if temp, ok := gmMap[u.UID.Int64()]; ok {
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
		Error: errors.ErrorOK(),
	}

	gmList, err := s.groupMemberDao.ListGroupByUID(ctx, types.ID(req.Uid))
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	var gidList = make([]types.ID, len(gmList))
	for i, gm := range gmList {
		gidList[i] = gm.GID
	}

	groupList, err := s.groupDao.ListGroups(ctx, gidList)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	for _, g := range groupList {
		rsp.Groups = append(rsp.Groups, g.ToProto())
	}

	return rsp, nil
}

func (s *GroupService) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	group := &data.Group{
		GID:         types.NewID(),
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		OwnerUID:    types.ID(req.OwnerUid),
		MaxMembers:  500, // todo: use config
		MemberCount: len(req.MembersUid) + 1,
	}

	var members = make([]*data.GroupMember, 0, len(req.MembersUid)+1)

	members = append(members, &data.GroupMember{
		GID:  group.GID,
		UID:  group.OwnerUID,
		Type: grouppb.GroupMember_TypeOwner,
	})

	for _, uid := range req.MembersUid {
		members = append(members, &data.GroupMember{
			GID:  group.GID,
			UID:  types.ID(uid),
			Type: grouppb.GroupMember_TypeMember,
		})
	}

	rsp := &grouppb.CreateGroupResponse{
		Error: errors.ErrorOK(),
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
		rsp.Error = errors.ErrorCode_DBError.Err2().WithError(err)
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	return rsp, nil
}

func (s *GroupService) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	rsp := &grouppb.UpdateGroupResponse{
		Error: errors.ErrorOK(),
	}

	var (
		gid = types.ID(req.Gid)
	)

	group, err := s.groupDao.GetGroupByGID(ctx, gid)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Error = errors.ErrorCode_GroupNotExist.Err2()
		return rsp, nil
	}

	if group.OwnerUID.Int64() != req.OwnerUid {
		rsp.Error = errors.ErrorCode_NotGroupOwner.Err2()
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
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	rsp.Group = group.ToProto()
	return rsp, nil
}

func (s *GroupService) DeleteGroup(ctx context.Context, req *grouppb.DeleteGroupRequest) (*errors.Error, error) {
	rsp := errors.ErrorOK()

	group, err := s.groupDao.GetGroupByGID(ctx, types.ID(req.Gid))
	if err != nil {
		rsp = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp = errors.ErrorCode_GroupNotExist.Err2()
		return rsp, nil
	}

	if group.OwnerUID.Int64() != req.OwnerUid {
		rsp = errors.ErrorCode_NotGroupOwner.Err2()
		return rsp, nil
	}

	err = s.groupDao.DeleteGroup(ctx, group)
	if err != nil {
		rsp = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	return rsp, nil
}

func (s *GroupService) AddGroupMember(ctx context.Context, req *grouppb.ChangeGroupMemberRequest) (
	*grouppb.ChangeGroupMemberResponse, error) {
	// todo: should limit the count of adding members, like only add 10 members per time.
	//  cause the add member operation is a heavy operation.
	rsp := &grouppb.ChangeGroupMemberResponse{
		Error: errors.ErrorOK(),
	}

	var (
		gid  = types.ID(req.Gid)
		uids = make([]types.ID, 0, len(req.Uids))
	)

	group, err := s.groupDao.GetGroupByGID(ctx, gid)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Error = errors.ErrorCode_GroupNotExist.Err2()
		return rsp, nil
	}

	for _, uid := range req.Uids {
		uids = append(uids, types.ID(uid))
	}

	uids, err = s.groupMemberDao.ListInGroupUIDs(ctx, group.GID, uids)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	// if all users are already in the group, return
	if len(uids) == len(req.Uids) {
		return rsp, nil
	}

	// filter out the users already in the group
	var (
		inGroupUIDs = util.NewSet[types.ID]()
		newUIDs     []types.ID
	)

	for _, uid := range uids {
		inGroupUIDs.Add(uid)
	}

	for _, uid := range req.Uids {
		id := types.ID(uid)
		if !inGroupUIDs.Contains(id) {
			newUIDs = append(newUIDs, id)
		}
	}

	// check if new users can add to the group, because the group has max member limit
	if len(newUIDs)+group.MemberCount > group.MaxMembers {
		rsp.Error = errors.ErrorCode_GroupLimitExceed.Err2()
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
		success, err1 := s.groupDao.IncrGroupMemberCount(ctx2, group, uint(len(newUIDs)))
		if err1 != nil {
			return err1
		}

		if !success {
			return errors.ErrorCode_GroupLimitExceed.Err2()
		}

		if err1 = s.groupMemberDao.CreateGroupMember(ctx2, gmList...); err1 != nil {
			return err1
		}

		return nil
	})

	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	rsp.Count = int32(len(newUIDs))
	return rsp, nil
}

func (s *GroupService) RemoveGroupMember(ctx context.Context, req *grouppb.ChangeGroupMemberRequest) (
	*grouppb.ChangeGroupMemberResponse, error) {
	rsp := &grouppb.ChangeGroupMemberResponse{
		Error: errors.ErrorOK(),
	}

	var (
		gid      = types.ID(req.Gid)
		ownerUID = types.ID(req.OwnerUid)
		uids     = make([]types.ID, 0, len(req.Uids))
	)

	group, err := s.groupDao.GetGroupByGID(ctx, gid)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	if group == nil {
		rsp.Error = errors.ErrorCode_GroupNotExist.Err2()
		return rsp, nil
	}

	if group.OwnerUID != ownerUID {
		rsp.Error = errors.ErrorCode_NotGroupOwner.Err2()
		return rsp, nil
	}

	uids, err = s.groupMemberDao.ListInGroupUIDs(ctx, group.GID, uids)
	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	// if all users are not in the group, return
	if len(uids) == 0 {
		return rsp, nil
	}

	// filter out the users not in the group
	var (
		inGroupUIDs    = util.NewSet[types.ID]()
		needRemoveUIDs []types.ID
	)

	for _, uid := range uids {
		inGroupUIDs.Add(uid)
	}

	for _, uid := range req.Uids {
		id := types.ID(uid)
		if inGroupUIDs.Contains(id) {
			needRemoveUIDs = append(needRemoveUIDs, id)
		}
	}

	// delete group members and decrease the member count
	err = db.Transaction(ctx, func(ctx2 context.Context) error {
		// decrease the member count first
		success, err1 := s.groupDao.DecrGroupMemberCount(ctx2, group, uint(len(needRemoveUIDs)))
		if err1 != nil {
			return err1
		}

		if !success {
			return errors.ErrorCode_GroupLimitExceed
		}

		if err1 = s.groupMemberDao.DeleteGroupMembers(ctx2, group.GID, needRemoveUIDs); err1 != nil {
			return err1
		}

		return nil
	})

	if err != nil {
		rsp.Error = errors.ErrorCode_DBError.WithError(err)
		return rsp, nil
	}

	rsp.Count = int32(len(needRemoveUIDs))
	return rsp, nil
}

func (s *GroupService) isInGroup(ctx context.Context, gid, uid types.ID) (bool, error) {
	gm, err := s.groupMemberDao.IsMemberOfGroup(ctx, gid, uid)
	if err != nil {
		return false, err
	}

	if gm == nil {
		return false, nil
	}

	return gm.Status == grouppb.GroupMember_StatusActive, nil
}
