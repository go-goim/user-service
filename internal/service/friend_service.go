package service

import (
	"context"
	"sync"

	"github.com/go-goim/api/errors"

	// api
	messagev1 "github.com/go-goim/api/message/v1"
	friendpb "github.com/go-goim/api/user/friend/v1"
	"github.com/go-goim/core/pkg/util"

	// core
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/log"
	"github.com/go-goim/core/pkg/types"
	"github.com/go-goim/core/pkg/util/retry"

	// internal
	"github.com/go-goim/user-service/internal/app"
	"github.com/go-goim/user-service/internal/dao"
	"github.com/go-goim/user-service/internal/data"
)

// FriendService implements friendpb.FriendServiceServer
type FriendService struct {
	friendDao        *dao.FriendDao
	friendRequestDao *dao.FriendRequestDao
	userDao          *dao.UserDao
	friendpb.UnimplementedFriendServiceServer
}

var (
	_                 friendpb.FriendServiceServer = &FriendService{}
	friendService     *FriendService
	friendServiceOnce sync.Once
)

func GetFriendService() *FriendService {
	friendServiceOnce.Do(func() {
		friendService = &FriendService{
			friendDao:        dao.GetUserRelationDao(),
			friendRequestDao: dao.GetFriendRequestDao(),
			userDao:          dao.GetUserDao(),
		}
	})
	return friendService
}

/*
 * handle friend request logic
 */

func (s *FriendService) AddFriend(ctx context.Context, req *friendpb.BaseFriendRequest) (
	*friendpb.AddFriendResponse, error) {
	var (
		uid  = types.ID(req.Uid)
		fuid = types.ID(req.FriendUid)
	)
	log.Info("add friend request", "uid", uid, "fuid", fuid)
	friendUser, err := s.userDao.GetUserByUID(ctx, fuid)
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.AddFriendResponse{
		Error:  errors.ErrorOK(),
		Result: &friendpb.AddFriendResult{},
	}

	if friendUser == nil {
		rsp.Error = errors.ErrorCode_UserNotExist.Err2()
		return rsp, nil
	}

	me, err := s.friendDao.GetFriend(ctx, uid, friendUser.UID)
	if err != nil {
		return nil, err
	}

	friend, err := s.friendDao.GetFriend(ctx, friendUser.UID, uid)
	if err != nil {
		return nil, err
	}

	// friend had blocked me or me had blocked friend
	if !s.canAddFriend(ctx, me, friend, rsp) {
		return rsp, nil
	}

	ok, err := s.addAutomatically(ctx, me, friend, rsp)
	if err != nil {
		return nil, err
	}

	// had added friend
	if ok {
		return rsp, nil
	}

	base := &friendpb.BaseFriendRequest{
		Uid:       req.Uid,
		FriendUid: friendUser.UID.Int64(),
	}
	// send friend request
	err = s.sendFriendRequest(ctx, base, rsp)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func (s *FriendService) canAddFriend(_ context.Context, me, friend *data.Friend,
	rsp *friendpb.AddFriendResponse) bool {
	// check if me blocked the friend
	if friend != nil && friend.IsBlocked() {
		rsp.Result.Status = friendpb.AddFriendStatus_BLOCKED_BY_FRIEND
		return false
	}

	// check if me has blocked the friend
	if me != nil && me.IsBlocked() {
		rsp.Result.Status = friendpb.AddFriendStatus_BLOCKED_BY_ME
		return false
	}

	return true
}

func (s *FriendService) addAutomatically(ctx context.Context, me, friend *data.Friend,
	rsp *friendpb.AddFriendResponse) (bool, error) {
	if friend == nil || friend.IsStranger() {
		return false, nil
	}

	// checked friend is not blocked me
	if me == nil {
		// create me -> friend relation
		me = &data.Friend{
			UID:       me.UID,
			FriendUID: friend.UID,
			Status:    friendpb.FriendStatus_FRIEND,
		}

		if err := s.friendDao.CreateFriend(ctx, me); err != nil {
			return false, err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_ADD_FRIEND_SUCCESS
		return true, nil
	}

	me.SetFriend()
	if err := s.friendDao.UpdateFriendStatus(ctx, me); err != nil {
		return false, err
	}

	rsp.Result.Status = friendpb.AddFriendStatus_ADD_FRIEND_SUCCESS
	return true, nil
}

// friend has not blocked me and has no relation with me(no data or status is stranger)
// me has not blocked the friend and may have relation with the friend(no data or status in [friend, stranger])
func (s *FriendService) sendFriendRequest(ctx context.Context, req *friendpb.BaseFriendRequest,
	rsp *friendpb.AddFriendResponse) error {
	var (
		uid  = types.ID(req.Uid)
		fuid = types.ID(req.FriendUid)
	)
	// load old friend request
	fr, err := s.friendRequestDao.GetFriendRequest(ctx, uid, fuid)
	if err != nil {
		return err
	}

	// if the friend request is not exist, create new one
	if fr == nil {
		fr = &data.FriendRequest{
			UID:       types.ID(req.Uid),
			FriendUID: types.ID(req.FriendUid),
			Status:    friendpb.FriendRequestStatus_REQUESTED,
		}

		if err := s.friendRequestDao.CreateFriendRequest(ctx, fr); err != nil {
			return err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_SEND_REQUEST_SUCCESS
		rsp.Result.FriendRequest = fr.ToProto()
		return nil
	}

	rsp.Result.FriendRequest = fr.ToProto()
	// if the friend request is exist, check the status
	if fr.IsRequested() {
		rsp.Result.Status = friendpb.AddFriendStatus_ALREADY_SENT_REQUEST
		return nil
	}

	if fr.IsAccepted() {
		// me and friend were friends before, no relation now, send friend request again
		fr.SetRequested()
		if err := s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_SEND_REQUEST_SUCCESS
		rsp.Result.FriendRequest = fr.ToProto()
	}

	if fr.IsRejected() {
		// reject the friend request, resend friend request
		fr.SetRequested()
		if err := s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_SEND_REQUEST_SUCCESS
		rsp.Result.FriendRequest = fr.ToProto()
	}

	return nil
}

func (s *FriendService) ConfirmFriendRequest(ctx context.Context, req *friendpb.ConfirmFriendRequestRequest) (
	*errors.Error, error) {
	fr, err := s.friendRequestDao.GetFriendRequestByID(ctx, req.FriendRequestId)
	if err != nil {
		return nil, err
	}

	if fr == nil {
		return errors.ErrorCode_FriendRequestNotExist.Err2(), nil
	}

	// check if the friend request is send to me
	if fr.FriendUID.Int64() != req.Uid {
		return errors.ErrorCode_FriendRequestNotExist.Err2(), nil
	}

	// cannot confirm friend request if the friend request is not requested
	// it means the friend request is accepted or rejected
	if !fr.IsRequested() {
		return errors.ErrorCode_FriendRequestStatusError.
			WithMessage("current friend request status cannot be confirmed"), nil
	}

	if req.Action == friendpb.ConfirmFriendRequestAction_REJECT {
		fr.SetRejected()
		if err = s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return nil, err
		}

		return errors.ErrorOK(), nil
	}

	// accept the friend request

	me, err := s.friendDao.GetFriend(ctx, fr.UID, fr.FriendUID)
	if err != nil {
		return nil, err
	}

	friend, err := s.friendDao.GetFriend(ctx, fr.FriendUID, fr.UID)
	if err != nil {
		return nil, err
	}

	if me != nil && me.IsFriend() && friend != nil && friend.IsFriend() {
		fr.SetAccepted()
		if err = s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return nil, err
		}

		return errors.ErrorOK(), nil
	}

	// big transaction here
	err = db.Transaction(ctx, func(ctx2 context.Context) error {
		// step 1: update friend request status to accepted
		fr.SetAccepted()
		if err = s.friendRequestDao.UpdateFriendRequest(ctx2, fr); err != nil {
			return err
		}

		// step 2: create or update friend relationship me -> friend
		if err = s.createOrSetFriend(ctx2, fr.UID, fr.FriendUID, me); err != nil {
			return err
		}

		// step 3: create or update friend relationship friend -> me
		if err = s.createOrSetFriend(ctx2, fr.FriendUID, fr.UID, friend); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errors.ErrorCode_DBError.WithError(err), nil
	}

	// set friend status in the cache
	// only set when the friend request is accepted.
	if err = s.friendDao.SetFriendStatusToCache(ctx, fr.UID, fr.FriendUID); err != nil {
		log.Error("set friend status to cache error",
			"err", err, "uid", fr.UID, "friend_uid", fr.FriendUID)

		// too complicated handling of retry, need to think about it
		err1 := retry.RetryWithQueue(func() error {
			return s.friendDao.SetFriendStatusToCache(ctx, fr.UID, fr.FriendUID)
		}, app.GetApplication().Producer, "retry_event_topic", map[string]interface{}{
			"uid":        fr.UID,
			"friend_uid": fr.FriendUID,
			"event":      "set_friend_status_to_cache",
		})

		if err1 != nil {
			log.Error("retry set friend status to cache error", "err", err1, "uid", fr.UID, "friend_uid", fr.FriendUID)
		}
	}

	return errors.ErrorOK(), nil
}

func (s *FriendService) createOrSetFriend(ctx context.Context, uid, friendUID types.ID, f *data.Friend) error {
	if f != nil {
		f.SetFriend()
		return s.friendDao.UpdateFriendStatus(ctx, f)
	}

	f = &data.Friend{
		UID:       uid,
		FriendUID: friendUID,
		Status:    friendpb.FriendStatus_FRIEND,
	}

	return s.friendDao.CreateFriend(ctx, f)
}

func (s *FriendService) GetFriendRequest(ctx context.Context, req *friendpb.BaseFriendRequest) (
	*friendpb.GetFriendRequestResponse, error) {
	var (
		uid  = types.ID(req.Uid)
		fuid = types.ID(req.FriendUid)
	)
	fr, err := s.friendRequestDao.GetFriendRequest(ctx, uid, fuid)
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.GetFriendRequestResponse{
		Error: errors.ErrorOK(),
	}

	if fr == nil {
		// rsp.Response = responsepb.NOT_FOUND

		return rsp, nil
	}

	rsp.FriendRequest = fr.ToProto()
	return rsp, nil
}

func (s *FriendService) QueryFriendRequestList(ctx context.Context, req *friendpb.QueryFriendRequestListRequest) (
	*friendpb.QueryFriendRequestListResponse, error) {
	var (
		uid = types.ID(req.Uid)
	)

	frList, err := s.friendRequestDao.GetFriendRequests(ctx, uid, int(req.Status))
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.QueryFriendRequestListResponse{
		Error:             errors.ErrorOK(),
		FriendRequestList: make([]*friendpb.FriendRequest, 0, len(frList)),
	}

	for _, fr := range frList {
		rsp.FriendRequestList = append(rsp.FriendRequestList, fr.ToProto())
	}

	return rsp, nil
}

/*
 * handle friend logic
 */

func (s *FriendService) IsFriend(ctx context.Context, req *friendpb.BaseFriendRequest) (
	*errors.Error, error) {
	ok, err := s.friendDao.CheckIsFriend(ctx, types.ID(req.Uid), types.ID(req.FriendUid))
	if err != nil {
		return errors.ErrorCode_CacheError.WithError(err), nil
	}

	if ok {
		return errors.ErrorOK(), nil
	}

	return errors.ErrorCode_RelationNotExist.Err2(), nil
}

func (s *FriendService) GetFriend(ctx context.Context, req *friendpb.BaseFriendRequest) (
	*friendpb.GetFriendResponse, error) {
	f, err := s.friendDao.GetFriend(ctx, types.ID(req.Uid), types.ID(req.FriendUid))
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.GetFriendResponse{
		Error: errors.ErrorOK(),
	}

	if f != nil {
		rsp.Friend = f.ToProtoFriend()
	}

	return rsp, nil
}

func (s *FriendService) QueryFriendList(ctx context.Context, req *friendpb.QueryFriendListRequest) (
	*friendpb.QueryFriendListResponse, error) {
	friends, err := s.friendDao.GetFriends(ctx, types.ID(req.Uid))
	if err != nil {
		return nil, err
	}

	var (
		rsp = &friendpb.QueryFriendListResponse{
			Error: errors.ErrorOK(),
		}
		friendUIDList = make([]types.ID, len(friends))
		friendMap     = make(map[int64]*data.User)
	)
	for i, f := range friends {
		rsp.FriendList = append(rsp.FriendList, f.ToProtoFriend())
		friendUIDList[i] = f.FriendUID
	}

	// get friend info
	friendInfoList, err := s.userDao.ListUsers(ctx, friendUIDList...)
	if err != nil {
		return nil, err
	}

	for i, friendInfo := range friendInfoList {
		friendMap[friendInfo.UID.Int64()] = friendInfoList[i]
	}

	for _, ur := range rsp.FriendList {
		if friendInfo, ok := friendMap[ur.FriendUid]; ok {
			ur.FriendName = friendInfo.Name
			ur.FriendAvatar = friendInfo.Avatar
		}
	}

	return rsp, nil
}

// UpdateFriendStatus update friend status.
// Second error is grpc error, not business error.
func (s *FriendService) UpdateFriendStatus(ctx context.Context, req *friendpb.UpdateFriendStatusRequest) (
	*errors.Error, error) {
	var (
		uid  = types.ID(req.Info.Uid)
		fuid = types.ID(req.Info.FriendUid)
	)
	f, err := s.friendDao.GetFriend(ctx, uid, fuid)
	if err != nil {
		return errors.ErrorCode_DBError.WithError(err), nil
	}

	if f == nil {
		return errors.ErrorCode_RelationNotExist.Err2(), nil
	}

	ok := f.Status.CanUpdateStatus(req.Status)
	if !ok {
		return errors.ErrorCode_InvalidUpdateRelationAction.Err2(), nil
	}

	if f.Status == req.Status {
		return errors.ErrorOK(), nil
	}

	// unfriend action, need remove friend status from cache
	if req.Status == friendpb.FriendStatus_STRANGER || req.Status == friendpb.FriendStatus_BLOCKED {
		err = s.onUnfriend(ctx, uid, fuid)
		if err != nil {
			return errors.ErrorCode_CacheError.WithError(err), nil
		}
	}

	// restore friend status to cache
	if req.Status == friendpb.FriendStatus_UNBLOCKED {
		err = s.onUnblock(ctx, uid, fuid)
		if err != nil {
			return errors.ErrorCode_CacheError.WithError(err), nil
		}
	}

	f.SetStatus(req.Status)
	if err := s.friendDao.UpdateFriendStatus(ctx, f); err != nil {
		return errors.ErrorCode_DBError.WithError(err), nil
	}

	return errors.ErrorOK(), nil
}

// delete or block friend.
func (s *FriendService) onUnfriend(ctx context.Context, uid, friendUID types.ID) error {
	return s.friendDao.DeleteFriendStatusFromCache(ctx, uid, friendUID)
}

func (s *FriendService) onUnblock(ctx context.Context, uid, friendUID types.ID) error {
	// check if friend is blocked me.
	friend, err := s.friendDao.GetFriend(ctx, friendUID, uid)
	if err != nil {
		return err
	}

	if friend != nil && friend.IsFriend() {
		// set cache.
		return s.friendDao.SetFriendStatusToCache(ctx, friendUID, uid)
	}

	return nil
}

/*
* handle friend send message ability
 */

func (s *FriendService) CheckSendMessageAbility(ctx context.Context, req *friendpb.CheckSendMessageAbilityRequest) (
	*friendpb.CheckSendMessageAbilityResponse, error) {
	rsp := &friendpb.CheckSendMessageAbilityResponse{
		Error: errors.ErrorOK(),
	}
	// TODO: should check if there a session id in request.
	//  If there is, check if the session id is valid.

	var (
		from = types.ID(req.FromUid)
		to   = types.ID(req.ToUid)
	)

	// is friend
	if req.SessionType == messagev1.SessionType_SingleChat {
		ok, err := s.friendDao.CheckIsFriend(ctx, from, to)
		if err != nil {
			return nil, err
		}

		if !ok {
			rsp.Error = errors.ErrorCode_RelationNotExist.Err2()
			return rsp, nil
		}
	}

	if req.SessionType == messagev1.SessionType_GroupChat {
		ok, err := GetGroupService().isInGroup(ctx, to, from)
		if err != nil {
			return nil, err
		}

		if !ok {
			rsp.Error = errors.ErrorCode_RelationNotExist.Err2()
			return rsp, nil
		}
	}

	// todo: check whether user subscribed if session type is channel.

	sid := util.Session(req.SessionType, from, to)
	rsp.SessionId = &sid
	return rsp, nil
}
