package cmd

import (
	"context"

	friendpb "github.com/go-goim/api/user/friend/v1"
	userv1 "github.com/go-goim/api/user/v1"

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

	// TODO: add registered grpc services to metadata in service registry.
	userv1.RegisterUserServiceServer(application.GrpcSrv, service.GetUserService())
	friendpb.RegisterFriendServiceServer(application.GrpcSrv, service.GetFriendService())

	if err = application.Run(); err != nil {
		log.Error("application run error", "error", err)
	}

	graceful.Register(application.Shutdown)
	if err = graceful.Shutdown(context.TODO()); err != nil {
		log.Error("graceful shutdown error", "error", err)
	}
}
