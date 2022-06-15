package dao

import (
	"context"
	"strings"
	"sync"

	"gorm.io/gorm"

	sessionpb "github.com/go-goim/api/user/session/v1"
	"github.com/go-goim/core/pkg/db"
	"github.com/go-goim/user-service/internal/data"
)

type SessionDao struct{}

var (
	sessionDao     *SessionDao
	sessionDaoOnce sync.Once
)

func GetSessionDao() *SessionDao {
	sessionDaoOnce.Do(func() {
		sessionDao = &SessionDao{}
	})
	return sessionDao
}

func (d *SessionDao) GetSession(ctx context.Context, sessionID string) (*data.Session, error) {
	session := &data.Session{}
	err := db.GetDBFromCtx(ctx).Where("id = ?", sessionID).First(session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return session, nil
}

// TODO: add cache
func (d *SessionDao) GetSessionByUID(ctx context.Context, fromUID, toUID string) (*data.Session, error) {
	if fromUID > toUID {
		fromUID, toUID = toUID, fromUID
	}

	session := &data.Session{}
	err := db.GetDBFromCtx(ctx).Where("from_user_id = ? AND to_user_id = ?", fromUID, toUID).First(session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return session, nil
}

func (d *SessionDao) CreateGroupSession(ctx context.Context, uid, groupID string, ignoreDuplicate ...bool) (
	*data.Session, error) {
	session, err := d.createSession(ctx, uid, groupID, sessionpb.SessionType_GroupChat)
	if err == nil {
		return session, nil
	}

	if len(ignoreDuplicate) > 0 && ignoreDuplicate[0] {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, nil
		}
	}

	return nil, err
}

func (d *SessionDao) CreateSingleChatSession(ctx context.Context, fromUID, toUID string, ignoreDuplicate ...bool) (
	*data.Session, error) {
	session, err := d.createSession(ctx, fromUID, toUID, sessionpb.SessionType_SingleChat)
	if err == nil {
		return session, nil
	}

	if len(ignoreDuplicate) > 0 && ignoreDuplicate[0] {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, nil
		}
	}

	return nil, err
}

func (d *SessionDao) createSession(ctx context.Context, fromUID, toUID string, sessionType sessionpb.SessionType) (*data.Session, error) {
	session := &data.Session{
		FromUserID: fromUID,
		ToUserID:   toUID,
		Type:       sessionType,
	}

	if session.IsSingleChat() && fromUID > toUID {
		session.FromUserID, session.ToUserID = session.ToUserID, session.FromUserID
	}

	err := db.GetDBFromCtx(ctx).Create(session).Error
	if err != nil {
		return nil, err
	}

	return session, nil
}
