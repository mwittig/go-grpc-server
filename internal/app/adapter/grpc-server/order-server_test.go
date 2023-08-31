package grpc_server

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"reflect"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pb "go-grpc-server/internal/app/protos/orderservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type dialFunc func(context.Context, string) (net.Conn, error)

func initMockupServer(_ *testing.T, opts ...grpc.ServerOption) (dialFunc, func()) {
	lis := bufconn.Listen(1024 * 1024)

	grpcServer := grpc.NewServer(opts...)

	pb.RegisterOrderServiceServer(grpcServer, &Server{
		UnimplementedOrderServiceServer: pb.UnimplementedOrderServiceServer{},
	})

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				log.Fatalf("failed to serve: %s", err)
			}
		}
	}()

	dial := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	shutdown := func() {
		grpcServer.GracefulStop()
	}

	return dial, shutdown
}

type Option interface{}

func isExported(id string) bool {
	r, _ := utf8.DecodeRuneInString(id)

	return unicode.IsUpper(r)
}

func isVisitedSubType(typeDef reflect.Type, visitedSubTypes map[string]bool) bool {
	if typeDef.Name() == "" {
		// in this case we have a Pointer Type
		return false
	}

	s := typeDef.PkgPath() + "." + typeDef.Name()
	if _, ok := visitedSubTypes[s]; ok {
		return true
	}

	visitedSubTypes[s] = true

	return false
}

func indirectVal(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Interface {
		return value.Elem()
	}

	return value
}
func addOptionsForIndirectValue(value reflect.Value, visitedSubTypes map[string]bool, opts cmp.Options) cmp.Options {
	if iv := indirectVal(value); iv.IsValid() && iv.CanInterface() {
		opts = addOptionsForType(iv.Interface(), visitedSubTypes, opts)
	}

	return opts
}
func addOptionsForType(i interface{}, visitedSubTypes map[string]bool, opts cmp.Options) cmp.Options {
	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return opts
	}

	t := v.Type()
	if t == nil {
		return opts
	}

	if isVisitedSubType(t, visitedSubTypes) {
		return opts
	}

	switch t.Kind() { //nolint:exhaustive
	case reflect.Slice, reflect.Array:
		// check for indirect value in case of value type is an interface
		for i := 0; i < v.Len(); i++ {
			opts = addOptionsForIndirectValue(v.Index(i), visitedSubTypes, opts)
		}

		opts = addOptionsForType(reflect.Zero(t.Elem()).Interface(), visitedSubTypes, opts)
	case reflect.Map:
		// keys of a map are evaluated differently so structs with unexported fields can be used as keys
		for _, k := range v.MapKeys() {
			opts = addOptionsForIndirectValue(k, visitedSubTypes, opts)
			opts = addOptionsForIndirectValue(v.MapIndex(k), visitedSubTypes, opts)
		}

		opts = addOptionsForType(reflect.Zero(t.Elem()).Interface(), visitedSubTypes, opts)
	case reflect.Ptr:
		if !v.IsZero() && v.Elem().CanInterface() {
			opts = addOptionsForType(v.Elem().Interface(), visitedSubTypes, opts)
		}

		opts = addOptionsForType(reflect.Zero(t.Elem()).Interface(), visitedSubTypes, opts)
	case reflect.Struct:
		opts = append(opts, cmpopts.IgnoreUnexported(reflect.Zero(t).Interface()))

		for j := 0; j < t.NumField(); j++ {
			if isExported(t.Field(j).Name) {
				if !v.Field(j).IsZero() && v.Field(j).CanInterface() {
					opts = addOptionsForType(v.Field(j).Interface(), visitedSubTypes, opts)
				}

				opts = addOptionsForType(reflect.Zero(t.Field(j).Type).Interface(), visitedSubTypes, opts)
			}
		}
	}

	return opts
}
func evaluateReceived(t *testing.T, exp interface{}, rcv interface{}, ignoreUnexported bool, options ...Option) {
	t.Helper()

	opts := make(cmp.Options, 0)
	opts = append(opts, cmpopts.EquateEmpty())

	for _, o := range options {
		cmpOpt, ok := o.(cmp.Option)
		if ok {
			opts = append(opts, cmpOpt)
		}
	}

	if ignoreUnexported {
		opts = addOptionsForType(exp, map[string]bool{}, opts)
	}

	diff := cmp.Diff(exp, rcv, opts...)
	if diff != "" {
		t.Errorf("Expected to match, but expected differs from received:\n%s", diff)
	}
}
func TestServer_DeployService(t *testing.T) {
	tests := []struct {
		name     string
		inputReq *pb.DeployServiceRequest
		want     *pb.DeployServiceResponse
		wantErr  bool
	}{
		{
			name: "Happy Case",
			inputReq: &pb.DeployServiceRequest{
				ServiceIds: []string{"1", "2", "3"},
			},
			want: &pb.DeployServiceResponse{
				ServiceIds: []string{"1", "2", "3"},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dial, shutdown := initMockupServer(t)

			conn, err := grpc.DialContext(context.TODO(), "",
				grpc.WithContextDialer(dial), grpc.WithTransportCredentials(insecure.NewCredentials()))

			if err != nil {
				t.Fatal(err)
			}

			if shutdown != nil {
				defer shutdown()
			}

			client := pb.NewOrderServiceClient(conn)

			got, err := client.DeployService(context.TODO(), tc.inputReq)
			if (err != nil) != tc.wantErr {
				t.Errorf("DeployService() error = %v, wantErr %v", err, tc.wantErr)

				return
			}

			evaluateReceived(t, tc.want, got, true)
		})
	}
}

func FuzzServer_DeployService(f *testing.F) {
	dial, shutdown := initMockupServer(nil)

	conn, err := grpc.DialContext(context.TODO(), "",
		grpc.WithContextDialer(dial), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		f.Fatal(err)
	}

	if shutdown != nil {
		defer shutdown()
	}

	client := pb.NewOrderServiceClient(conn)

	f.Fuzz(func(t *testing.T, id []byte, length uint) {
		ids := make([]string, length)

		for it := range ids {
			ids[it] = base64.StdEncoding.EncodeToString(id)
		}

		t.Log(id, length, len(ids))

		_, err := client.DeployService(context.TODO(), &pb.DeployServiceRequest{ServiceIds: ids})
		if err != nil {
			t.Errorf("DeployService() error = %v, wantErr %v", err, false)

			return
		}
	})
}
