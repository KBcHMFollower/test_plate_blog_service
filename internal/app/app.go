package app

import (
	"github.com/KBcHMFollower/test_plate_blog_service/config"
	database2 "github.com/KBcHMFollower/test_plate_blog_service/database"
	"github.com/KBcHMFollower/test_plate_blog_service/internal/rabbitmq"
	"log/slog"

	grpcapp "github.com/KBcHMFollower/test_plate_blog_service/internal/app/grpc"
	"github.com/KBcHMFollower/test_plate_blog_service/internal/repository"
	postService "github.com/KBcHMFollower/test_plate_blog_service/internal/services"
)

type App struct {
	GRPCServer   *grpcapp.GRPCApp
	RabbitMqConn *rabbitmq.Connection
}

func New(
	log *slog.Logger,
	cfg *config.Config,
) *App {
	op := "App.New"
	appLog := log.With(
		slog.String("op", op),
	)

	dbDriver, db, err := database2.New(cfg.Storage.ConnectionString)
	if err != nil {
		appLog.Error("db connection error: ", err)
		panic(err)
	}

	postRepository, err := repository.NewPostRepository(dbDriver)
	if err != nil {
		appLog.Error("TODO:", err)
		panic(err)
	}
	if err := database2.ForceMigrate(db, cfg.Storage.MigrationPath); err != nil {
		appLog.Error("db migrate error: ", err)
		panic(err)
	}

	mcon, err := rabbitmq.New()
	if err != nil {
		appLog.Error("can`t open rabbitmq connection: ", err)
		panic(err)
	}
	appLog.Info("RabbitMQ connection is opened")

	err = rabbitmq.DeclareExchangeForPosts(mcon)
	if err != nil {
		return nil
	}
	appLog.Info("Post Exchange is declared")

	err = rabbitmq.DeclareAndBindDeletePostQueue(mcon)
	if err != nil {
		return nil
	}
	appLog.Info("Post delete queue is created")

	postService := postService.New(postRepository, log, mcon)

	GRPCApp := grpcapp.New(cfg.GRpc.Port, log, postService)

	return &App{
		GRPCServer:   GRPCApp,
		RabbitMqConn: mcon,
	}
}
