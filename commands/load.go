package commands

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dictyBase/apihelpers/aphfile"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"google.golang.org/grpc"
	"gopkg.in/urfave/cli.v1"
)

type UserStatus struct {
	Exist bool
	User  *pb.User
}

func LoadUser(c *cli.Context) error {
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
	defer conn.Close()
	client := pb.NewUserServiceClient(conn)
	log := getLogger(c)
	s3Client, err := aphfile.GetS3Client(
		fmt.Sprintf("%s:%s", c.String("s3-server"), c.String("s3-server-port")),
		c.String("access-key"),
		c.String("secret-key"),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("error in getting s3 client %s", err),
			2,
		)
	}
	tmpDir, err := aphfile.FetchAndDecompress(
		s3Client,
		c.String("s3-bucket"),
		c.String("remote-path"),
		"users",
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("error in fetching remote file %s", err),
			2,
		)
	}
	log.Debugf("retrieved the remote file in dir %s", tmpDir)
	// open the csv file for reading
	usersFile := filepath.Join(tmpDir, c.String("data-file"))
	handler, err := os.Open(usersFile)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to open file %s %s", usersFile, err),
			2,
		)
	}
	defer handler.Close()
	r := csv.NewReader(handler)
	_, err = r.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to read header from csv file %s", err),
			2,
		)
	}
	// variable for records
	total := 0
	inserted := 0
	updated := 0
	// read the file and insert record as needed
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("Unable to read from csv file %s", err),
				2,
			)
		}
		ustatus, err := findOrCreateUser(client, record)
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("error in finding or creating user %s %s", record[0], err),
				2,
			)
		}
		if ustatus.Exist {
			err := updateUser(client, ustatus)
			if err != nil {
				return cli.NewExitError(
					fmt.Sprintf("error in updating user %s %s", record[0], err),
					2,
				)
			}
			updated++
			total++
			log.Debugf("updated record with email %s\n", record[0])
			continue
		} else {
			log.Debugf("created record with email %s\n", record[0])
		}
		inserted++
		total++
	}
	log.Infof("records total:%d new:%d updated%d", total, inserted, updated)
	return nil
}

func updateUser(client pb.UserServiceClient, ustatus *UserStatus) error {
	_, err := client.UpdateUser(
		context.Background(),
		newUpdateUserReq(ustatus.User),
	)
	return err
}

func findOrCreateUser(client pb.UserServiceClient, record []string) (*UserStatus, error) {
	u, err := client.GetUserByEmail(context.Background(), &jsonapi.GetEmailRequest{Email: record[0]})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound { // create user
				nuser, err := client.CreateUser(context.Background(), newUser(record))
				if err != nil {
					return &UserStatus{
						Exist: false,
					}, err
				}
				return &UserStatus{
					Exist: false,
					User:  nuser,
				}, nil
			}
		}
		return &UserStatus{
			Exist: false,
		}, err
	}
	return &UserStatus{
		Exist: true,
		User:  u,
	}, nil
}

func newUpdateUserReq(existingUser *pb.User) *pb.UpdateUserRequest {
	attr := existingUser.Data.Attributes
	return &pb.UpdateUserRequest{
		Id: existingUser.Data.Id,
		Data: &pb.UpdateUserRequest_Data{
			Id:   existingUser.Data.Id,
			Type: existingUser.Data.Type,
			Attributes: &pb.UserAttributes{
				FirstName:    attr.FirstName,
				LastName:     attr.LastName,
				Organization: attr.Organization,
				GroupName:    attr.GroupName,
				FirstAddress: attr.FirstAddress,
				City:         attr.City,
				State:        attr.State,
				Zipcode:      attr.Zipcode,
				Country:      attr.Country,
				Phone:        attr.Phone,
				IsActive:     attr.IsActive,
			},
		},
	}
}

func newUser(record []string) *pb.CreateUserRequest {
	attr := &pb.UserAttributes{}
	for i, v := range record {
		switch i {
		case 6:
			if len(v) > 0 {
				attr.Organization = v
			}
		case 7:
			if len(v) > 0 {
				attr.FirstAddress = v
			}
		case 8:
			if len(v) > 0 {
				attr.SecondAddress = v
			}
		case 9:
			if len(v) > 0 {
				attr.City = v
			}
		case 10:
			if len(v) > 0 {
				attr.State = v
			}
		case 12:
			if len(v) > 0 {
				attr.Country = v
			}
		case 13:
			if len(v) > 0 {
				attr.Zipcode = v
			}
		case 15:
			if len(v) > 0 {
				attr.Phone = v
			}
		}
	}
	attr.IsActive = getActiveStatus(record)
	attr.Email = record[0]
	attr.FirstName = record[1]
	attr.LastName = record[2]
	return &pb.CreateUserRequest{
		Data: &pb.CreateUserRequest_Data{
			Type:       "users",
			Attributes: attr,
		},
	}
}

func getActiveStatus(record []string) bool {
	if len(record[14]) > 0 {
		if record[14] == "Y" {
			return true
		}
	}
	return false
}
