package client

import (
	"context"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/message"
	"google.golang.org/grpc"
)

type grpcUserClient struct {
	client user.UserServiceClient
}

func NewUserClient(conn *grpc.ClientConn) message.UserClient {
	return &grpcUserClient{
		client: user.NewUserServiceClient(conn),
	}
}

func (g *grpcUserClient) Get(id int64) (*user.User, error) {
	return g.client.GetUser(context.Background(), &jsonapi.GetRequest{Id: id})
}

func (g *grpcUserClient) Delete(id int64) (bool, error) {
	_, err := g.client.DeleteUser(context.Background(), &jsonapi.DeleteRequest{Id: id})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *grpcUserClient) Exist(id int64) (bool, error) {
	resp, err := g.client.ExistUser(context.Background(), &jsonapi.IdRequest{Id: id})
	return resp.Exist, err
}
