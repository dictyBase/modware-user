package nats

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	"github.com/dictyBase/modware-user/message"
	gnats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/encoders/protobuf"
)

type natsReply struct {
	econn *gnats.EncodedConn
	sub   *gnats.Subscription
}

func NewReply(host, port string, options ...gnats.Option) (message.Reply, error) {
	nc, err := gnats.Connect(fmt.Sprintf("nats://%s:%s", host, port), options...)
	if err != nil {
		return &natsReply{}, err
	}
	ec, err := gnats.NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return &natsReply{}, err
	}
	return &natsReply{econn: ec}, nil
}

func (n *natsReply) Publish(subj string, urep *pubsub.UserReply) {
	n.econn.Publish(subj, urep)
}

func (n *natsReply) Start(subj string, client message.UserClient, replyFn message.ReplyFn) error {
	sub, err := n.econn.Subscribe(subj, func(s, rep string, req *pubsub.IdRequest) {
		n.Publish(rep, replyFn(s, client, req))
	})
	if err != nil {
		return err
	}
	if err := n.econn.Flush(); err != nil {
		return err
	}
	if err := n.econn.LastError(); err != nil {
		return err
	}
	n.sub = sub
	return nil
}

func (n *natsReply) Stop() error {
	if n.sub != nil {
		n.sub.Unsubscribe()
	}
	n.econn.Close()
	return nil
}
