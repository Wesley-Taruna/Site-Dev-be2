package handler

import (
	"fami-boilerplate-service/global"
	"fami-boilerplate-service/internal/app/service"

	// pb "git.edenfarm.id/project-version3/erp-services/erp-protobuf/gen/proto/boilerplate_service"
	pb "gitlab.com/fami-app-v2/fami-datamodel/fami-protobuf/generated/proto/account_service"
)

type BoilerplateGrpcHandler struct {
	Option         global.HandlerOptions
	ServicesPerson service.IPersonService
	pb.UnimplementedAccountServiceServer
}

// func (h *BoilerplateGrpcHandler) GetPerson(ctx context.Context, req *pb.GetPersonRequest) (res *pb.GetPersonResponse, err error) {
// 	ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.GetPerson")
// 	defer span.End()

// 	var persons []dto.PersonResponseGet

// 	persons, _, err = h.ServicesPerson.Get(ctx, int(req.Offset), int(req.Limit), req.Search)
// 	if err != nil {
// 		err = status.New(codes.NotFound, err.Error()).Err()
// 		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
// 		return
// 	}

// 	var pbPerson []*pb.Person
// 	for _, person := range persons {
// 		pbPerson = append(pbPerson, &pb.Person{
// 			Id:        person.ID,
// 			Name:      person.Name,
// 			City:      person.City,
// 			Country:   person.Country,
// 			CreatedAt: timestamppb.New(person.CreatedAt),
// 			UpdatedAt: timestamppb.New(person.UpdatedAt),
// 		})
// 	}

// 	res = &pb.GetPersonResponse{
// 		Data: pbPerson,
// 	}
// 	return
// }

// func (h *BoilerplateGrpcHandler) GetPersonByID(ctx context.Context, req *pb.GetPersonByIDRequest) (res *pb.GetPersonByIDResponse, err error) {
// 	ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.GetPersonByID")
// 	defer span.End()

// 	var person dto.PersonResponseGet

// 	person, err = h.ServicesPerson.GetByID(ctx, req.Id)
// 	if err != nil {
// 		err = status.New(codes.NotFound, err.Error()).Err()
// 		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
// 		return
// 	}

// 	res = &pb.GetPersonByIDResponse{
// 		Code:    int32(codes.OK),
// 		Message: codes.OK.String(),
// 		Data: &pb.Person{
// 			Id:        person.ID,
// 			Name:      person.Name,
// 			City:      person.City,
// 			Country:   person.Country,
// 			CreatedAt: timestamppb.New(person.CreatedAt),
// 			UpdatedAt: timestamppb.New(person.UpdatedAt),
// 		},
// 	}
// 	return
// }

// func (h *BoilerplateGrpcHandler) CreatePerson(ctx context.Context, req *pb.CreatePersonRequest) (res *pb.CreatePersonResponse, err error) {
// 	ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.CreatePerson")
// 	defer span.End()

// 	person, err := h.ServicesPerson.Create(ctx, dto.PersonRequestCreate{
// 		Name:    req.Person.Name,
// 		City:    req.Person.City,
// 		Country: req.Person.Country,
// 	})
// 	if err != nil {
// 		err = status.New(codes.NotFound, err.Error()).Err()
// 		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
// 		return
// 	}

// 	res = &pb.CreatePersonResponse{
// 		Code:    int32(codes.OK),
// 		Message: codes.OK.String(),
// 		Data: &pb.Person{
// 			Id:        person.ID,
// 			Name:      person.Name,
// 			City:      person.City,
// 			Country:   person.Country,
// 			CreatedAt: timestamppb.New(person.CreatedAt),
// 			UpdatedAt: timestamppb.New(person.UpdatedAt),
// 		},
// 	}
// 	return
// }

// func (h *BoilerplateGrpcHandler) UpdatePerson(ctx context.Context, req *pb.UpdatePersonRequest) (res *pb.UpdatePersonResponse, err error) {
// 	ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.UpdatePerson")
// 	defer span.End()

// 	person, err := h.ServicesPerson.Update(ctx, dto.PersonRequestUpdate{
// 		Name:    req.Person.Name,
// 		City:    req.Person.City,
// 		Country: req.Person.Country,
// 	})
// 	if err != nil {
// 		err = status.New(codes.NotFound, err.Error()).Err()
// 		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
// 		return
// 	}

// 	res = &pb.UpdatePersonResponse{
// 		Code:    int32(codes.OK),
// 		Message: codes.OK.String(),
// 		Data: &pb.Person{
// 			Id:        person.ID,
// 			Name:      person.Name,
// 			City:      person.City,
// 			Country:   person.Country,
// 			CreatedAt: timestamppb.New(person.CreatedAt),
// 			UpdatedAt: timestamppb.New(person.UpdatedAt),
// 		},
// 	}
// 	return
// }

// func (h *BoilerplateGrpcHandler) DeletePerson(ctx context.Context, req *pb.DeletePersonRequest) (res *pb.DeletePersonResponse, err error) {
// 	ctx, span := h.Option.Common.Trace.Start(ctx, "Grpc.DeletePerson")
// 	defer span.End()

// 	_, err = h.ServicesPerson.Delete(ctx, dto.PersonRequestDelete{
// 		ID:   req.Id,
// 		Note: req.Note,
// 	})
// 	if err != nil {
// 		err = status.New(codes.NotFound, err.Error()).Err()
// 		h.Option.Common.Logger.AddMessage(log.ErrorLevel, err).Print()
// 		return
// 	}

// 	res = &pb.DeletePersonResponse{
// 		Code:    int32(codes.OK),
// 		Message: codes.OK.String(),
// 	}

// 	return
// }
