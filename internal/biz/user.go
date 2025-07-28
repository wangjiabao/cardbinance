package biz

import (
	pb "cardbinance/api/user/v1"
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID            uint64
	Address       string
	Card          string
	CardNumber    string
	CardOrderId   string
	CardAmount    float64
	Amount        float64
	AmountTwo     uint64
	MyTotalAmount uint64
	IsDelete      uint64
	Vip           uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UserRecommend struct {
	ID            uint64
	UserId        uint64
	RecommendCode string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Config struct {
	ID      int64
	KeyName string
	Name    string
	Value   string
}

type UserRepo interface {
	GetConfigByKeys(keys ...string) ([]*Config, error)
	GetUserByAddress(address string) (*User, error)
	GetUserById(userId uint64) (*User, error)
	GetUserRecommendByUserId(userId uint64) (*UserRecommend, error)
	CreateUser(ctx context.Context, uc *User) (*User, error)
	CreateUserRecommend(ctx context.Context, userId uint64, recommendUser *UserRecommend) (*UserRecommend, error)
	GetUserRecommendByCode(code string) ([]*UserRecommend, error)
	GetUserByUserIds(userIds []uint64) (map[uint64]*User, error)
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

func (uuc *UserUseCase) GetExistUserByAddressOrCreate(ctx context.Context, u *User, req *pb.EthAuthorizeRequest) (*User, error, string) {
	var (
		user          *User
		recommendUser *UserRecommend
		err           error
		configs       []*Config
		vipMax        uint64
	)

	// 配置
	configs, err = uuc.repo.GetConfigByKeys("vip_max")
	if nil != configs {
		for _, vConfig := range configs {
			if "vip_max" == vConfig.KeyName {
				vipMax, _ = strconv.ParseUint(vConfig.Value, 10, 64)
			}
		}
	}

	recommendUser = &UserRecommend{
		ID:            0,
		UserId:        0,
		RecommendCode: "",
	}

	user, err = uuc.repo.GetUserByAddress(u.Address) // 查询用户
	if nil == user && nil == err {
		code := req.SendBody.Code // 查询推荐码 abf00dd52c08a9213f225827bc3fb100 md5 dhbmachinefirst
		if "abf00dd52c08a9213f225827bc3fb100" != code {
			if 1 >= len(code) {
				return nil, errors.New(500, "USER_ERROR", "无效的推荐码1"), "无效的推荐码"
			}
			var (
				userRecommend *User
			)

			userRecommend, err = uuc.repo.GetUserByAddress(code)
			if nil == userRecommend || err != nil {
				return nil, errors.New(500, "USER_ERROR", "无效的推荐码1"), "无效的推荐码"
			}

			// 查询推荐人的相关信息
			recommendUser, err = uuc.repo.GetUserRecommendByUserId(userRecommend.ID)
			if nil == recommendUser || err != nil {
				return nil, errors.New(500, "USER_ERROR", "无效的推荐码3"), "无效的推荐码3"
			}
		} else {
			u.Vip = vipMax
		}

		if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
			user, err = uuc.repo.CreateUser(ctx, u) // 用户创建
			if err != nil {
				return err
			}

			_, err = uuc.repo.CreateUserRecommend(ctx, user.ID, recommendUser) // 创建用户推荐信息
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			return nil, err, "错误"
		}
	}

	return user, err, ""
}

func (uuc *UserUseCase) GetUserById(userId uint64) (*pb.GetUserReply, error) {
	var (
		user                   *User
		userRecommend          *UserRecommend
		userRecommendUser      *User
		myUserRecommendUserId  uint64
		myUserRecommendAddress string
		err                    error
	)

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.GetUserReply{Status: "-1"}, nil
	}

	// 推荐
	userRecommend, err = uuc.repo.GetUserRecommendByUserId(userId)
	if nil == userRecommend {
		return &pb.GetUserReply{Status: "-1"}, nil
	}

	if "" != userRecommend.RecommendCode {
		tmpRecommendUserIds := strings.Split(userRecommend.RecommendCode, "D")
		if 2 <= len(tmpRecommendUserIds) {
			myUserRecommendUserId, _ = strconv.ParseUint(tmpRecommendUserIds[len(tmpRecommendUserIds)-1], 10, 64) // 最后一位是直推人

			if 0 < myUserRecommendUserId {
				userRecommendUser, err = uuc.repo.GetUserById(myUserRecommendUserId)
				if nil == userRecommendUser || nil != err {
					return &pb.GetUserReply{Status: "-1"}, nil
				}

				myUserRecommendAddress = userRecommendUser.Address
			}
		}

	}

	return &pb.GetUserReply{
		Status:           "ok",
		Address:          user.Address,
		Amount:           fmt.Sprintf("%.2f", user.Amount),
		MyTotalAmount:    user.MyTotalAmount,
		Vip:              user.Vip,
		CardNum:          user.CardNumber,
		CardAmount:       fmt.Sprintf("%.2f", user.CardAmount),
		RecommendAddress: myUserRecommendAddress,
	}, nil
}

func (uuc *UserUseCase) GetUserDataById(userId uint64) (*User, error) {
	return uuc.repo.GetUserById(userId)
}

func (uuc *UserUseCase) GetUserRecommend(ctx context.Context, req *pb.RecommendListRequest) (*pb.RecommendListReply, error) {
	var (
		userRecommend   *UserRecommend
		myUserRecommend []*UserRecommend
		user            *User
		err             error
	)

	res := make([]*pb.RecommendListReply_List, 0)

	if 0 >= len(req.Address) {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	user, err = uuc.repo.GetUserByAddress(req.Address)
	if nil == user || nil != err {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	// 推荐
	userRecommend, err = uuc.repo.GetUserRecommendByUserId(user.ID)
	if nil == userRecommend {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	myUserRecommend, err = uuc.repo.GetUserRecommendByCode(userRecommend.RecommendCode + "D" + strconv.FormatUint(user.ID, 10))
	if nil == myUserRecommend || nil != err {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	if 0 >= len(myUserRecommend) {
		return &pb.RecommendListReply{
			Status:     "ok",
			Recommends: res,
		}, nil
	}

	tmpUserIds := make([]uint64, 0)
	for _, vMyUserRecommend := range myUserRecommend {
		tmpUserIds = append(tmpUserIds, vMyUserRecommend.UserId)
	}
	if 0 >= len(tmpUserIds) {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	var (
		usersMap map[uint64]*User
	)

	usersMap, err = uuc.repo.GetUserByUserIds(tmpUserIds)
	if nil == usersMap || nil != err {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	if 0 >= len(usersMap) {
		return &pb.RecommendListReply{
			Status:     "错误",
			Recommends: res,
		}, nil
	}

	for _, vMyUserRecommend := range myUserRecommend {
		if _, ok := usersMap[vMyUserRecommend.UserId]; !ok {
			continue
		}

		cardOpen := uint64(0)
		if 10 < len(usersMap[vMyUserRecommend.UserId].CardNumber) {
			cardOpen = 1
		}

		res = append(res, &pb.RecommendListReply_List{
			Address:  usersMap[vMyUserRecommend.UserId].Address,
			Amount:   usersMap[vMyUserRecommend.UserId].AmountTwo + usersMap[vMyUserRecommend.UserId].MyTotalAmount,
			Vip:      usersMap[vMyUserRecommend.UserId].Vip,
			CardOpen: cardOpen,
		})
	}

	return &pb.RecommendListReply{
		Status:     "ok",
		Recommends: res,
	}, nil
}

func (uuc *UserUseCase) OrderList(ctx context.Context, req *pb.OrderListRequest, userId uint64) (*pb.OrderListReply, error) {

	return &pb.OrderListReply{
		Status: "ok",
		Count:  0,
		List:   nil,
	}, nil
}

func (uuc *UserUseCase) RewardList(ctx context.Context, req *pb.RewardListRequest, userId uint64) (*pb.RewardListReply, error) {

	return &pb.RewardListReply{
		Status: "ok",
		Count:  0,
		List:   nil,
	}, nil
}

var lockAmount sync.Mutex

func (uuc *UserUseCase) OpenCard(ctx context.Context, req *pb.OpenCardRequest, userId uint64) (*pb.OpenCardReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	return &pb.OpenCardReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) AmountToCard(ctx context.Context, req *pb.AmountToCardRequest, userId uint64) (*pb.AmountToCardReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	return &pb.AmountToCardReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) AmountTo(ctx context.Context, req *pb.AmountToRequest, userId uint64) (*pb.AmountToReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	return &pb.AmountToReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) Withdraw(ctx context.Context, req *pb.WithdrawRequest, userId uint64) (*pb.WithdrawReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	return &pb.WithdrawReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) SetVip(ctx context.Context, req *pb.SetVipRequest, userId uint64) (*pb.SetVipReply, error) {

	return &pb.SetVipReply{
		Status: "ok",
	}, nil
}
