package cmd

import (
	"context"

	friendpb "github.com/go-goim/api/user/friend/v1"
	grouppb "github.com/go-goim/api/user/group/v1"
	userv1 "github.com/go-goim/api/user/v1"

	"github.com/go-goim/core/pkg/cache"
	"github.com/go-goim/core/pkg/cmd"
	"github.com/go-goim/core/pkg/graceful"
	"github.com/go-goim/core/pkg/log"

	"github.com/go-goim/user-service/internal/app"
	"github.com/go-goim/user-service/internal/service"
)

func Main() {
	if err := cmd.ParseFlags(); err != nil {
		panic(err)
	}

	application, err := app.InitApplication()
	if err != nil {
		log.Fatal("InitApplication got err", "error", err)
	}

	userv1.RegisterUserServiceServer(application.GrpcSrv, service.GetUserService())
	friendpb.RegisterFriendServiceServer(application.GrpcSrv, service.GetFriendService())
	grouppb.RegisterGroupServiceServer(application.GrpcSrv, service.GetGroupService())

	cache.SetGlobalCache(cache.NewRedisCache(application.Redis))

	if err = application.Run(); err != nil {
		log.Error("application run error", "error", err)
	}

	graceful.Register(application.Shutdown)
	if err = graceful.Shutdown(context.TODO()); err != nil {
		log.Error("graceful shutdown error", "error", err)
	}
}
