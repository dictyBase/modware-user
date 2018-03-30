package message

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
)

type UserClient interface {
	Get(int64) (*user.User, error)
	Delete(int64) (bool, error)
	Exist(int64) (bool, error)
}

type ReplyFn func(string, UserClient, *pubsub.IdRequest) *pubsub.UserReply

type Reply interface {
	Publish(string, *pubsub.UserReply)
	Start(string, UserClient, ReplyFn) error
	Stop() error
}
