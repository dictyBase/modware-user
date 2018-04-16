package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/dictyBase/apihelpers/aphgrpc"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/server"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
	"gopkg.in/urfave/cli.v1"
)

func RunRoleServer(c *cli.Context) error {
	dbh, err := getPgWrapper(c)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to create database connection %s", err.Error()),
			2,
		)
	}
	grpcS := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
	pb.RegisterRoleServiceServer(grpcS, server.NewRoleService(dbh, aphgrpc.BaseURLOption(setApiHost(c))))
	reflection.Register(grpcS)

	// http requests muxer
	runtime.HTTPError = aphgrpc.CustomHTTPError
	httpMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(aphgrpc.HandleCreateResponse),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	endP := fmt.Sprintf(":%s", c.String("port"))
	err = pb.RegisterRoleServiceHandlerFromEndpoint(context.Background(), httpMux, endP, opts)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("unable to register http endpoint for user microservice %s", err),
			2,
		)
	}

	// create listener
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			2,
		)
	}
	// create the cmux object that will multiplex 2 protocols on same port
	m := cmux.New(lis)
	// match gRPC requests, otherwise regular HTTP requests
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	httpS := &http.Server{Handler: httpMux}
	// collect on this channel the exits of each protocol's .Serve() call
	ech := make(chan error, 2)
	// start the listeners for each protocol
	go func() { ech <- grpcS.Serve(grpcL) }()
	go func() { ech <- httpS.Serve(httpL) }()
	log.Printf("starting multiplexed  server on %s", endP)
	var failed bool
	if err := m.Serve(); err != nil {
		log.Printf("cmux serve error: %v", err)
		failed = true
	}
	i := 0
	for err := range ech {
		if err != nil {
			log.Printf("protocol serve error:%v", err)
			failed = true
		}
		i++
		if cap(ech) == i {
			close(ech)
			break
		}
	}
	if failed {
		return cli.NewExitError("error in running cmux server", 2)
	}
	return nil
}

func RunUserServer(c *cli.Context) error {
	dbh, err := getPgWrapper(c)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to create database connection %s", err.Error()),
			2,
		)
	}
	grpcS := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
	pb.RegisterUserServiceServer(grpcS, server.NewUserService(dbh, aphgrpc.BaseURLOption(setApiHost(c))))
	reflection.Register(grpcS)

	// http requests muxer
	runtime.HTTPError = aphgrpc.CustomHTTPError
	httpMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(aphgrpc.HandleCreateResponse),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	endP := fmt.Sprintf(":%s", c.String("port"))
	err = pb.RegisterUserServiceHandlerFromEndpoint(context.Background(), httpMux, endP, opts)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("unable to register http endpoint for user microservice %s", err),
			2,
		)
	}

	// create listener
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			2,
		)
	}
	// create the cmux object that will multiplex 2 protocols on same port
	m := cmux.New(lis)
	// match gRPC requests, otherwise regular HTTP requests
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	httpS := &http.Server{Handler: httpMux}
	// collect on this channel the exits of each protocol's .Serve() call
	ech := make(chan error, 2)
	// start the listeners for each protocol
	go func() { ech <- grpcS.Serve(grpcL) }()
	go func() { ech <- httpS.Serve(httpL) }()
	log.Printf("starting multiplexed  server on %s", endP)
	var failed bool
	if err := m.Serve(); err != nil {
		log.Printf("cmux serve error: %v", err)
		failed = true
	}
	i := 0
	for err := range ech {
		if err != nil {
			log.Printf("protocol serve error:%v", err)
			failed = true
		}
		i++
		if cap(ech) == i {
			close(ech)
			break
		}
	}
	if failed {
		return cli.NewExitError("error in running cmux server", 2)
	}
	return nil
}

func RunPermissionServer(c *cli.Context) error {
	dbh, err := getPgWrapper(c)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("Unable to create database connection %s", err.Error()),
			2,
		)
	}
	grpcS := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
	pb.RegisterPermissionServiceServer(grpcS, server.NewPermissionService(dbh, aphgrpc.BaseURLOption(setApiHost(c))))
	reflection.Register(grpcS)

	// http requests muxer
	runtime.HTTPError = aphgrpc.CustomHTTPError
	httpMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(aphgrpc.HandleCreateResponse),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	endP := fmt.Sprintf(":%s", c.String("port"))
	err = pb.RegisterPermissionServiceHandlerFromEndpoint(context.Background(), httpMux, endP, opts)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("unable to register http endpoint for user microservice %s", err),
			2,
		)
	}

	// create listener
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			2,
		)
	}
	// create the cmux object that will multiplex 2 protocols on same port
	m := cmux.New(lis)
	// match gRPC requests, otherwise regular HTTP requests
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	httpS := &http.Server{Handler: httpMux}
	// collect on this channel the exits of each protocol's .Serve() call
	ech := make(chan error, 2)
	// start the listeners for each protocol
	go func() { ech <- grpcS.Serve(grpcL) }()
	go func() { ech <- httpS.Serve(httpL) }()
	log.Printf("starting multiplexed  server on %s", endP)
	var failed bool
	if err := m.Serve(); err != nil {
		log.Printf("cmux serve error: %v", err)
		failed = true
	}
	i := 0
	for err := range ech {
		if err != nil {
			log.Printf("protocol serve error:%v", err)
			failed = true
		}
		i++
		if cap(ech) == i {
			close(ech)
			break
		}
	}
	if failed {
		return cli.NewExitError("error in running cmux server", 2)
	}
	return nil
}

func getPgxDbHandler(c *cli.Context) (*sql.DB, error) {
	cStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.String("dictyuser-user"),
		c.String("dictyuser-pass"),
		c.String("dictyuser-host"),
		c.String("dictyuser-port"),
		c.String("dictyuser-db"),
	)
	return sql.Open("pgx", cStr)
}

func getPgWrapper(c *cli.Context) (*runner.DB, error) {
	var dbh *runner.DB
	h, err := getPgxDbHandler(c)
	if err != nil {
		return dbh, err
	}
	return runner.NewDB(h, "postgres"), nil
}

func setApiHost(c *cli.Context) string {
	if len(c.String("user-api-http-host")) > 0 {
		return c.String("user-api-http-host")
	}
	return fmt.Sprintf("http://localhost:%s", c.String("port"))
}

func getLogger(c *cli.Context) *logrus.Entry {
	log := logrus.New()
	log.Out = os.Stderr
	switch c.GlobalString("log-format") {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
		}
	case "json":
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
		}
	}
	l := c.GlobalString("log-level")
	switch l {
	case "debug":
		log.Level = logrus.DebugLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	case "fatal":
		log.Level = logrus.FatalLevel
	case "panic":
		log.Level = logrus.PanicLevel
	}
	return logrus.NewEntry(log)
}
