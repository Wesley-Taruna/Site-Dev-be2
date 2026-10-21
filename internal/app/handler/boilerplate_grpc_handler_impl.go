package handler

import (
	"context"
	"fmt"
	"time"

	pb "gitlab.com/fami-app-v2/fami-datamodel/fami-protobuf/generated/proto/account_service"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// pb "git.edenfarm.id/project-version3/erp-services/erp-protobuf/gen/proto/boilerplate_service"

func (h *BoilerplateGrpcHandler) GetUserList(ctx context.Context, req *pb.GetUserListRequest) (res *pb.GetUserListResponse, err error) {
	// ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.GetUserList")
	// defer span.End()
	fmt.Println("Masuk SINI NIH")
	// person, err := h.ServicesPerson.GetByID(ctx, 1)
	// if err != nil {

	// 	err = status.New(codes.NotFound, err.Error()).Err()
	// 	h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
	// 	return
	// }
	// _ = person

	res = &pb.GetUserListResponse{
		Code:    int32(codes.OK),
		Message: codes.OK.String(),
		Data: []*pb.User{
			{
				Id:          1,
				Email:       "user1@example.com",
				Name:        "User One",
				Nickname:    "U1",
				PhoneNumber: "1234567890",
				Password:    "password123",
				Status:      1,
				Timezone:    0,
				CreatedAt:   timestamppb.New(time.Now()),
				UpdatedAt:   timestamppb.New(time.Now()),
			},
			{
				Id:          2,
				Email:       "user2@example.com",
				Name:        "User Two",
				Nickname:    "U2",
				PhoneNumber: "0987654321",
				Password:    "password456",
				Status:      1,
				Timezone:    0,
				CreatedAt:   timestamppb.New(time.Now()),
				UpdatedAt:   timestamppb.New(time.Now()),
			},
			{
				Id:          3,
				Email:       "user3@example.com",
				Name:        "User Three",
				Nickname:    "U3",
				PhoneNumber: "5555555555",
				Password:    "password789",
				Status:      1,
				Timezone:    0,
				CreatedAt:   timestamppb.New(time.Now()),
				UpdatedAt:   timestamppb.New(time.Now()),
			},
		},
	}
	return
}
