package service

import (
	"context"
	"errors"
	"sync"

	responsepb "github.com/go-goim/api/transport/response"
	friendpb "github.com/go-goim/api/user/friend/v1"
	sessionpb "github.com/go-goim/api/user/session/v1"

	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/core/pkg/log"
	"github.com/go-goim/core/pkg/util/retry"

	"github.com/go-goim/user-service/internal/app"
	"github.com/go-goim/user-service/internal/dao"
	"github.com/go-goim/user-service/internal/data"
)

// FriendService implements friendpb.FriendServiceServer
type FriendService struct {
	friendDao        *dao.FriendDao
	friendRequestDao *dao.FriendRequestDao
	userDao          *dao.UserDao
	sessionDao       *dao.SessionDao
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
			sessionDao:       dao.GetSessionDao(),
		}
	})
	return friendService
}

/*
 * handle friend request logic
 */

func (s *FriendService) AddFriend(ctx context.Context, req *friendpb.AddFriendRequest) (
	*friendpb.AddFriendResponse, error) {
	friendUser, err := s.userDao.GetUserByUID(ctx, req.GetFriendUid())
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.AddFriendResponse{
		Response: responsepb.Code_OK.BaseResponse(),
		Result:   &friendpb.AddFriendResult{},
	}

	if friendUser == nil {
		rsp.Response = responsepb.Code_UserNotExist.BaseResponse()
		return rsp, nil
	}

	me, err := s.friendDao.GetFriend(ctx, req.GetUid(), friendUser.UID)
	if err != nil {
		return nil, err
	}

	friend, err := s.friendDao.GetFriend(ctx, friendUser.UID, req.GetUid())
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
		Uid:       req.GetUid(),
		FriendUid: friendUser.UID,
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
	// load old friend request
	fr, err := s.friendRequestDao.GetFriendRequest(ctx, req.GetUid(), req.GetFriendUid())
	if err != nil {
		return err
	}

	// if the friend request is not exist, create new one
	if fr == nil {
		fr = &data.FriendRequest{
			UID:       req.GetUid(),
			FriendUID: req.GetFriendUid(),
			Status:    friendpb.FriendRequestStatus_REQUESTED,
		}

		if err := s.friendRequestDao.CreateFriendRequest(ctx, fr); err != nil {
			return err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_SEND_REQUEST_SUCCESS
		return nil
	}

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
	}

	if fr.IsRejected() {
		// reject the friend request, resend friend request
		fr.SetRequested()
		if err := s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return err
		}

		rsp.Result.Status = friendpb.AddFriendStatus_SEND_REQUEST_SUCCESS
	}

	return nil
}

func (s *FriendService) ConfirmFriendRequest(ctx context.Context, req *friendpb.ConfirmFriendRequestReq) (
	*responsepb.BaseResponse, error) {
	fr, err := s.friendRequestDao.GetFriendRequestByID(ctx, req.GetFriendRequestId())
	if err != nil {
		return nil, err
	}

	if fr == nil {
		return responsepb.Code_FriendRequestNotExist.BaseResponse(), nil
	}

	// check if the friend request is send to me
	if fr.FriendUID != req.GetUid() {
		return responsepb.Code_FriendRequestNotExist.BaseResponse(), nil
	}

	// cannot confirm friend request if the friend request is not requested
	// it means the friend request is accepted or rejected
	if !fr.IsRequested() {
		return responsepb.NewBaseResponseWithMessage(responsepb.Code_FriendRequestStatusError,
			"current friend request status cannot be confirmed"), nil
	}

	if req.GetAction() == friendpb.ConfirmFriendRequestAction_REJECT {
		fr.SetRejected()
		if err = s.friendRequestDao.UpdateFriendRequest(ctx, fr); err != nil {
			return nil, err
		}

		return responsepb.Code_OK.BaseResponse(), nil
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

		return responsepb.Code_OK.BaseResponse(), nil
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
		return responsepb.NewBaseResponseWithError(err), nil
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

	return responsepb.Code_OK.BaseResponse(), nil

}

func (s *FriendService) createOrSetFriend(ctx context.Context, uid, friendUID string, f *data.Friend) error {
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
	fr, err := s.friendRequestDao.GetFriendRequest(ctx, req.GetUid(), req.GetFriendUid())
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.GetFriendRequestResponse{
		Response: responsepb.Code_OK.BaseResponse(),
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

	frList, err := s.friendRequestDao.GetFriendRequests(ctx, req.GetUid(), int(req.GetStatus()))
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.QueryFriendRequestListResponse{
		Response:          responsepb.Code_OK.BaseResponse(),
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
	*responsepb.BaseResponse, error) {
	ok, err := s.friendDao.CheckIsFriend(ctx, req.GetUid(), req.GetFriendUid())
	if err != nil {
		return responsepb.NewBaseResponseWithMessage(responsepb.Code_CacheError, err.Error()), nil
	}

	if ok {
		return responsepb.Code_OK.BaseResponse(), nil
	}

	return responsepb.Code_RelationNotExist.BaseResponse(), nil
}

func (s *FriendService) GetFriend(ctx context.Context, req *friendpb.BaseFriendRequest) (
	*friendpb.GetFriendResponse, error) {
	f, err := s.friendDao.GetFriend(ctx, req.GetUid(), req.GetFriendUid())
	if err != nil {
		return nil, err
	}

	rsp := &friendpb.GetFriendResponse{
		Response: responsepb.Code_OK.BaseResponse(),
	}

	if f != nil {
		rsp.Friend = f.ToProtoFriend()
	}

	return rsp, nil
}

func (s *FriendService) QueryFriendList(ctx context.Context, req *friendpb.QueryFriendListRequest) (
	*friendpb.QueryFriendListResponse, error) {
	friends, err := s.friendDao.GetFriends(ctx, req.GetUid())
	if err != nil {
		return nil, err
	}

	var (
		rsp = &friendpb.QueryFriendListResponse{
			Response: responsepb.Code_OK.BaseResponse(),
		}
		friendUIDList = make([]string, len(friends))
		friendMap     = make(map[string]*data.User)
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
		friendMap[friendInfo.UID] = friendInfoList[i]
	}

	for _, ur := range rsp.FriendList {
		if friendInfo, ok := friendMap[ur.FriendUid]; ok {
			ur.FriendName = friendInfo.Name
			ur.FriendAvatar = friendInfo.Avatar
		}
	}

	return rsp, nil
}

func (s *FriendService) UpdateFriendStatus(ctx context.Context, req *friendpb.UpdateFriendStatusRequest) (
	*responsepb.BaseResponse, error) {
	info := req.GetInfo()
	f, err := s.friendDao.GetFriend(ctx, info.GetUid(), info.GetFriendUid())
	if err != nil {
		return nil, err
	}

	if f == nil {
		return responsepb.Code_RelationNotExist.BaseResponse(), nil
	}

	ok := f.Status.CanUpdateStatus(req.GetStatus())
	if !ok {
		return responsepb.Code_InvalidUpdateRelationAction.BaseResponse(), nil
	}

	if f.Status == req.GetStatus() {
		return responsepb.Code_OK.BaseResponse(), nil
	}

	// unfriend action, need remove friend status from cache
	if req.GetStatus() == friendpb.FriendStatus_STRANGER || req.GetStatus() == friendpb.FriendStatus_BLOCKED {
		err = s.onUnfriend(ctx, info.GetUid(), info.GetFriendUid())
		if err != nil {
			return responsepb.NewBaseResponseWithMessage(responsepb.Code_CacheError, err.Error()), nil
		}
	}

	// restore friend status to cache
	if req.GetStatus() == friendpb.FriendStatus_UNBLOCKED {
		err = s.onUnblock(ctx, info.GetUid(), info.GetFriendUid())
		if err != nil {
			return responsepb.NewBaseResponseWithMessage(responsepb.Code_CacheError, err.Error()), nil
		}
	}

	f.SetStatus(req.GetStatus())
	if err := s.friendDao.UpdateFriendStatus(ctx, f); err != nil {
		return responsepb.NewBaseResponseWithError(err), nil
	}

	return responsepb.Code_OK.BaseResponse(), nil
}

// delete or block friend.
func (s *FriendService) onUnfriend(ctx context.Context, uid, friendUID string) error {
	return s.friendDao.DeleteFriendStatusFromCache(ctx, uid, friendUID)
}

func (s *FriendService) onUnblock(ctx context.Context, uid, friendUID string) error {
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
		Response: responsepb.Code_OK.BaseResponse(),
	}

	// is friend
	if req.GetSessionType() == sessionpb.SessionType_SingleChat {
		ok, err := s.friendDao.CheckIsFriend(ctx, req.GetFromUid(), req.GetToUid())
		if err != nil {
			return nil, err
		}

		if !ok {
			rsp.Response = responsepb.Code_RelationNotExist.BaseResponse()
			return rsp, nil
		}
	}

	if req.GetSessionType() == sessionpb.SessionType_GroupChat {
		ok, err := GetGroupService().isInGroup(ctx, req.GetToUid(), req.GetFromUid())
		if err != nil {
			return nil, err
		}

		if !ok {
			rsp.Response = responsepb.Code_RelationNotExist.BaseResponse()
			return rsp, nil
		}
	}

	sessionID, err := s.getSessionID(ctx, req.GetFromUid(), req.GetToUid(), req.GetSessionType())
	if err != nil {
		rsp.Response = responsepb.NewBaseResponseWithError(err)
		return rsp, nil
	}

	rsp.SessionId = &sessionID
	return rsp, nil
}

func (s *FriendService) getSessionID(ctx context.Context, fromUId, toUID string, sessionType sessionpb.SessionType) (
	int64, error) {
	var (
		tryCount = 0
	)

CHECK:
	if tryCount > 1 {
		return 0, errors.New("get/create session failed")
	}

	// check session
	session, err := s.sessionDao.GetSessionByUID(ctx, fromUId, toUID)
	if err != nil {
		return 0, err
	}

	if session != nil {
		return session.ID, nil
	}

	// create session
	switch sessionType {
	case sessionpb.SessionType_GroupChat:
		session, err = s.sessionDao.CreateGroupSession(ctx, toUID)
	case sessionpb.SessionType_SingleChat:
		session, err = s.sessionDao.CreateSingleChatSession(ctx, fromUId, toUID)
	default:
		return 0, errors.New("invalid session type")
	}
	if err != nil {
		return 0, err
	}
	// err nil
	if session == nil {
		tryCount++
		// ???????????? ??????????????????????????????????????????????????????session,??????????????????????????? duplicate ??????
		// ?????????????????????,?????????????????????,???????????????
		goto CHECK
	}

	return session.ID, nil
}
