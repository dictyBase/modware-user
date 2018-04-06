package commands

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"github.com/dictyBase/modware-user/message"
	gclient "github.com/dictyBase/modware-user/message/grpc-client"
	"github.com/dictyBase/modware-user/message/nats"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/urfave/cli.v1"
)

func shutdown(r message.Reply, logger *logrus.Entry) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	logger.Errorf("received kill signal %s", <-ch)
	if err := r.Stop(); err != nil {
		logger.Fatalf("unable to close the subscription %s\n", err)
	}
}

func replyUser(subj string, c message.UserClient, req *pubsub.IdRequest) *pubsub.UserReply {
	switch subj {
	case "UserService.Get":
		u, err := c.Get(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  false,
			}
		}
		return &pubsub.UserReply{
			Exist: true,
			User:  u,
		}
	case "UserService.Exist":
		exist, err := c.Exist(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  exist,
			}
		}
		return &pubsub.UserReply{
			Exist: exist,
		}
	case "UserService.Delete":
		deleted, err := c.Delete(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  deleted,
			}
		}
		return &pubsub.UserReply{
			Exist: deleted,
		}
	default:
		return &pubsub.UserReply{
			Status: status.Newf(codes.Internal, "subject %s is not supported", subj).Proto(),
		}
	}
}

func RunUserReply(c *cli.Context) error {
	reply, err := nats.NewReply(
		c.String("messaging-host"),
		c.String("messaging-port"),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to reply server %s", err),
			2,
		)
	}
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%s", c.String("user-grpc-host"), c.String("user-grpc-port")),
		grpc.WithInsecure(),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to grpc server for user microservice %s", err),
			2,
		)
	}
	err = reply.Start(
		"UserService.*",
		gclient.NewUserClient(conn),
		replyUser,
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot start the reply server %s", err),
			2,
		)
	}
	logger := getLogger(c)
	go shutdown(reply, logger)
	logger.Info("starting the reply messaging backend")
	runtime.Goexit()
	return nil
}
