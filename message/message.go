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

type Reply interface {
	PublishAction(string, pubsub.Reply)
	PublishUser(string, *user.User)
	Start(string, *UserClient) error
	Stop() error
}
