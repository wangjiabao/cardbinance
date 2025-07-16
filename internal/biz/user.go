package biz

import (
	pb "cardbinance/api/user/v1"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type User struct {
	ID      uint64
	Address string
	//Card      string
	Amount    uint64
	IsDelete  uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo interface {
	GetUserByAddress(address string) (*User, error)
}

type UserUseCase struct {
	repo UserRepo
	tx   Transaction
	log  *log.Helper
}

func NewUserUseCase(repo UserRepo, tx Transaction, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		repo: repo,
		tx:   tx,
		log:  log.NewHelper(logger),
	}
}

func (uuc *UserUseCase) GetUserByAddress(ctx context.Context, address string) (*pb.GetUserReply, error) {
	var (
		user *User
		err  error
	)

	user, err = uuc.repo.GetUserByAddress(address)
	if nil == user || nil != err {
		return &pb.GetUserReply{Status: "-1"}, nil
	}

	return &pb.GetUserReply{Status: "ok", Address: user.Address}, nil
}
