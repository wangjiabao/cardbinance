package biz

import (
	"bytes"
	pb "cardbinance/api/user/v1"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
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
	FirstName     string
	LastName      string
	Email         string
	CountryCode   string
	Phone         string
	City          string
	Country       string
	Street        string
	PostalCode    string
	BirthDate     string
	CardUserId    string
	ProductId     string
	MaxCardQuota  uint64
	UserCount     uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	VipTwo        uint64
	CardTwo       uint64
	CanVip        uint64
}

type UserRecommend struct {
	ID            uint64
	UserId        uint64
	RecommendCode string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Config struct {
	ID      uint64
	KeyName string
	Name    string
	Value   string
}

type Withdraw struct {
	ID        int64
	UserId    int64
	Amount    float64
	RelAmount float64
	Status    string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Reward struct {
	ID        uint64
	UserId    uint64
	Amount    float64
	Reason    uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Address   string
	One       uint64
}

type CardRecord struct {
	ID         uint64
	UserId     uint64
	RecordType uint64
	Remark     string
	Code       string
	Opt        string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UserRepo interface {
	SetNonceByAddress(ctx context.Context, wallet string) (int64, error)
	GetAndDeleteWalletTimestamp(ctx context.Context, wallet string) (string, error)
	GetConfigByKeys(keys ...string) ([]*Config, error)
	GetUserByAddress(address string) (*User, error)
	GetUserById(userId uint64) (*User, error)
	GetUserRecommendByUserId(userId uint64) (*UserRecommend, error)
	CreateUser(ctx context.Context, uc *User) (*User, error)
	CreateUserRecommend(ctx context.Context, userId uint64, recommendUser *UserRecommend) (*UserRecommend, error)
	GetUserRecommendByCode(code string) ([]*UserRecommend, error)
	GetUserRecommendLikeCode(code string) ([]*UserRecommend, error)
	GetUserByUserIds(userIds []uint64) (map[uint64]*User, error)
	CreateCard(ctx context.Context, userId uint64, user *User) error
	CreateCardTwo(ctx context.Context, userId uint64, user *User) error
	GetAllUsers() ([]*User, error)
	UpdateCard(ctx context.Context, userId uint64, cardOrderId, card string) error
	CreateCardRecommend(ctx context.Context, userId uint64, amount float64, vip uint64, address string) error
	AmountToCard(ctx context.Context, userId uint64, amount float64) error
	AmountToCardReward(ctx context.Context, userId uint64, amount float64, orderId string) error
	AmountTo(ctx context.Context, userId, toUserId uint64, toAddress string, amount float64) error
	Withdraw(ctx context.Context, userId uint64, amount, amountRel float64, address string) error
	GetUserRewardByUserIdPage(ctx context.Context, b *Pagination, userId uint64, reason uint64) ([]*Reward, error, int64)
	GetUserRecordByUserIdPage(ctx context.Context, b *Pagination, userId uint64) ([]*CardRecord, error, int64)
	SetVip(ctx context.Context, userId uint64, vip uint64) error
	GetUsersOpenCard() ([]*User, error)
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

func (uuc *UserUseCase) GetUserById(userId uint64) (*pb.GetUserReply, error) {
	var (
		user                   *User
		userRecommend          *UserRecommend
		userRecommendUser      *User
		myUserRecommendUserId  uint64
		myUserRecommendAddress string
		err                    error
		withdrawRate           float64
	)

	var (
		configs []*Config
	)

	// 配置
	configs, err = uuc.repo.GetConfigByKeys("withdraw_rate")
	if nil != configs {
		for _, vConfig := range configs {
			if "withdraw_rate" == vConfig.KeyName {
				withdrawRate, _ = strconv.ParseFloat(vConfig.Value, 10)
			}
		}
	}

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

	cardStatus := uint64(0)
	var (
		cardAmount string
	)
	if "no" == user.CardOrderId {
		cardStatus = 0
	} else {
		if "no" == user.CardNumber {
			cardStatus = 1
		} else {
			cardStatus = 2
			// 查询状态。成功分红
			var (
				resCard *CardInfoResponse
			)
			resCard, err = GetCardInfoRequestWithSign(user.Card)
			if nil == resCard || 200 != resCard.Code || err != nil {

			} else {
				if "ACTIVE" == resCard.Data.CardStatus {
					cardAmount = resCard.Data.Balance
				}
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
		CardStatus:       cardStatus,
		CardAmount:       cardAmount,
		RecommendAddress: myUserRecommendAddress,
		WithdrawRate:     withdrawRate,
		CardStatusTwo:    user.CardTwo,
		CanVip:           user.CanVip,
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
		if "no" != usersMap[vMyUserRecommend.UserId].CardNumber {
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

type Pagination struct {
	PageNum  int
	PageSize int
}

func (uuc *UserUseCase) OrderList(ctx context.Context, req *pb.OrderListRequest, userId uint64) (*pb.OrderListReply, error) {
	res := make([]*pb.OrderListReply_List, 0)

	var (
		user   *User
		err    error
		cardId uint64
	)

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.OrderListReply{Status: "查询错误", Count: 0,
			List: res,
		}, nil
	}

	if 5 > len(user.Card) {
		return &pb.OrderListReply{Status: "ok", Count: 0,
			List: res,
		}, nil
	}

	if 5 > len(user.CardNumber) {
		return &pb.OrderListReply{Status: "ok", Count: 0,
			List: res,
		}, nil
	}

	cardId, err = strconv.ParseUint(user.Card, 10, 64)
	if err != nil {
		return &pb.OrderListReply{Status: "查询错误", Count: 0,
			List: res,
		}, nil
	}

	var (
		resGet *CardTransactionListResponse
	)
	resGet, err = GetCardTransactionList(cardId, req.Page, 20)
	if err != nil {
		return &pb.OrderListReply{Status: "查询错误", Count: 0,
			List: res,
		}, nil
	}

	if 200 != resGet.Code || 0 >= resGet.Total {
		return &pb.OrderListReply{
			Status: "ok",
			Count:  0,
			List:   res,
		}, nil
	}

	for _, v := range resGet.Rows {
		res = append(res, &pb.OrderListReply_List{
			Timestamp:               v.Timestamp,
			Status:                  v.Status,
			TradeAmount:             v.TradeAmount,
			ActualTransactionAmount: v.ActualTransactionAmount,
			ServiceFee:              v.ServiceFee,
			TradeDescription:        v.TradeDescription,
			CurrentBalance:          v.CurrentBalance,
			TraderNum:               v.TradeNo,
		})
	}

	return &pb.OrderListReply{
		Status: "ok",
		Count:  resGet.Total,
		List:   res,
	}, nil
}

func (uuc *UserUseCase) RecordList(ctx context.Context, req *pb.RecordListRequest, userId uint64) (*pb.RecordListReply, error) {
	res := make([]*pb.RecordListReply_List, 0)

	var (
		userRewards []*CardRecord
		count       int64
		err         error
	)

	userRewards, err, count = uuc.repo.GetUserRecordByUserIdPage(ctx, &Pagination{
		PageNum:  int(req.Page),
		PageSize: 20,
	}, userId)
	if nil != err {
		return &pb.RecordListReply{
			Status: "ok",
			Count:  uint64(count),
			List:   res,
		}, err
	}

	for _, vUserReward := range userRewards {
		res = append(res, &pb.RecordListReply_List{
			CreatedAt: vUserReward.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
			Remark:    vUserReward.Remark,
		})
	}

	return &pb.RecordListReply{
		Status: "ok",
		Count:  uint64(count),
		List:   res,
	}, nil
}

func (uuc *UserUseCase) RewardList(ctx context.Context, req *pb.RewardListRequest, userId uint64) (*pb.RewardListReply, error) {
	res := make([]*pb.RewardListReply_List, 0)

	var (
		userRewards []*Reward
		count       int64
		err         error
	)

	if 1 > req.ReqType || 6 < req.ReqType {
		return &pb.RewardListReply{
			Status: "参数错误",
			Count:  0,
			List:   res,
		}, nil
	}

	userRewards, err, count = uuc.repo.GetUserRewardByUserIdPage(ctx, &Pagination{
		PageNum:  int(req.Page),
		PageSize: 20,
	}, userId, req.ReqType)
	if nil != err {
		return &pb.RewardListReply{
			Status: "ok",
			Count:  uint64(count),
			List:   res,
		}, err
	}

	for _, vUserReward := range userRewards {
		res = append(res, &pb.RewardListReply_List{
			CreatedAt: vUserReward.CreatedAt.Add(8 * time.Hour).Format("2006-01-02 15:04:05"),
			Amount:    fmt.Sprintf("%.4f", vUserReward.Amount),
			Address:   vUserReward.Address,
		})
	}

	return &pb.RewardListReply{
		Status: "ok",
		Count:  uint64(count),
		List:   res,
	}, nil
}

// 无锁的

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
			if "abf00dd52c08a9213f225827bc3fb100other" != code {
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

				if 30 == userRecommend.VipTwo {
					u.VipTwo = 30
				}

			} else {
				u.Vip = 30
				u.VipTwo = 30
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

// 有锁的

var lockCreateNonce sync.Mutex

func (uuc *UserUseCase) CreateNonce(ctx context.Context, req *pb.CreateNonceRequest) (*pb.CreateNonceReply, error) {
	lockCreateNonce.Lock()
	defer lockCreateNonce.Unlock()

	nonce, err := uuc.repo.SetNonceByAddress(ctx, req.SendBody.Address)
	if nil != err {
		return &pb.CreateNonceReply{Nonce: "-1", Status: "生成错误"}, err
	}

	return &pb.CreateNonceReply{Nonce: strconv.FormatInt(nonce, 10), Status: "ok"}, nil
}

// 凡是操作的都涉及到这个锁
var lockNonce sync.Mutex

func (uuc *UserUseCase) GetAddressNonce(ctx context.Context, address string) (string, error) {
	lockNonce.Lock()
	defer lockNonce.Unlock()

	return uuc.repo.GetAndDeleteWalletTimestamp(ctx, address)
}

var lockVip sync.Mutex

func (uuc *UserUseCase) SetVip(ctx context.Context, req *pb.SetVipRequest, userId uint64) (*pb.SetVipReply, error) {
	lockVip.Lock()
	defer lockVip.Unlock()

	var (
		user   *User
		toUser *User
		err    error
	)

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.SetVipReply{Status: "用户不存在"}, nil
	}

	toUser, err = uuc.repo.GetUserByAddress(req.SendBody.Address)
	if nil == toUser || nil != err {
		return &pb.SetVipReply{Status: "目标用户不存在"}, nil
	}

	if toUser.VipTwo != user.VipTwo {
		return &pb.SetVipReply{Status: "vip等级不是统一系统账户"}, nil
	}

	if 0 >= toUser.VipTwo {
		if 0 > req.SendBody.Vip || 9 < req.SendBody.Vip {
			return &pb.SetVipReply{Status: "vip等级必须在0-9之间"}, nil
		}
	} else {
		if 0 > req.SendBody.Vip || 29 < req.SendBody.Vip {
			return &pb.SetVipReply{Status: "vip等级必须在0-29之间"}, nil
		}
	}

	if req.SendBody.Vip >= user.Vip {
		return &pb.SetVipReply{Status: "必须小于自己的vip等级"}, nil
	}

	if 30 > len(req.SendBody.Address) || 60 < len(req.SendBody.Address) {
		return &pb.SetVipReply{Status: "账号参数格式不正确"}, nil
	}

	if req.SendBody.Vip == toUser.Vip {
		return &pb.SetVipReply{Status: "无需修改"}, nil
	}

	var (
		users    []*User
		usersMap map[uint64]*User
	)
	users, err = uuc.repo.GetAllUsers()
	if nil == users {
		return &pb.SetVipReply{Status: "获取数据错误不存在"}, nil
	}

	usersMap = make(map[uint64]*User, 0)
	for _, vUsers := range users {
		usersMap[vUsers.ID] = vUsers
	}

	var (
		userRecommend   *UserRecommend
		myUserRecommend []*UserRecommend
	)
	// 推荐
	userRecommend, err = uuc.repo.GetUserRecommendByUserId(toUser.ID)
	if nil == userRecommend {
		return &pb.SetVipReply{Status: "目标用户不存在"}, nil
	}

	if 1 == user.CanVip {
		if "" != userRecommend.RecommendCode {
			tmpRecommendUserIds := strings.Split(userRecommend.RecommendCode, "D")
			if 2 <= len(tmpRecommendUserIds) {
				tmpMyUp := false
				for _, v := range tmpRecommendUserIds {
					myUserRecommendUserId, _ := strconv.ParseUint(v, 10, 64) // 最后一位是直推人
					if myUserRecommendUserId <= 0 {
						continue
					} else {
						if myUserRecommendUserId == userId {
							tmpMyUp = true
						}
					}
				}

				if !tmpMyUp {
					return &pb.SetVipReply{Status: "目标用户并不是你的团队用户"}, nil
				}

				// todo 随意设置以及后续分红时候的等级判断调整
			}
		} else {
			return &pb.SetVipReply{Status: "目标用户无上级"}, nil
		}

	} else {
		if "" != userRecommend.RecommendCode {
			tmpRecommendUserIds := strings.Split(userRecommend.RecommendCode, "D")
			if 2 <= len(tmpRecommendUserIds) {
				myUserRecommendUserId, _ := strconv.ParseUint(tmpRecommendUserIds[len(tmpRecommendUserIds)-1], 10, 64) // 最后一位是直推人
				if myUserRecommendUserId <= 0 || myUserRecommendUserId != userId {
					return &pb.SetVipReply{Status: "推荐人信息错误"}, nil
				}
			}

		} else {
			return &pb.SetVipReply{Status: "目标用户无上级"}, nil
		}

		// 下级比我小
		myUserRecommend, err = uuc.repo.GetUserRecommendLikeCode(userRecommend.RecommendCode + "D" + strconv.FormatUint(toUser.ID, 10))
		if nil == myUserRecommend || nil != err {
			return &pb.SetVipReply{Status: "获取数据错误不存在"}, nil
		}

		for _, v := range myUserRecommend {
			if _, ok := usersMap[v.UserId]; !ok {
				return &pb.SetVipReply{Status: "数据异常"}, nil
			}

			if req.SendBody.Vip <= usersMap[v.UserId].Vip {
				return &pb.SetVipReply{Status: "他下级的等级存在大于等于当前的设置"}, nil
			}
		}

		if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
			err = uuc.repo.SetVip(ctx, toUser.ID, req.SendBody.Vip)
			if nil != err {
				return err
			}

			return nil
		}); nil != err {
			fmt.Println(err, "设置vip写入mysql错误", user)
			return &pb.SetVipReply{
				Status: "设置vip错误，联系管理员",
			}, nil
		}
	}

	return &pb.SetVipReply{
		Status: "ok",
	}, nil
}

var lockAmount sync.Mutex

func (uuc *UserUseCase) OpenCard(ctx context.Context, req *pb.OpenCardRequest, userId uint64) (*pb.OpenCardReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	var (
		user       *User
		err        error
		cardAmount float64
	)

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.OpenCardReply{Status: "用户不存在"}, nil
	}

	if 3 <= user.UserCount {
		return &pb.OpenCardReply{Status: "提交已经3次。联系管理员"}, nil
	}

	if "no" != user.CardNumber {
		return &pb.OpenCardReply{Status: "已经开卡"}, nil
	}

	if "no" != user.CardOrderId {
		return &pb.OpenCardReply{Status: "已经提交开卡信息"}, nil
	}

	if 0 >= user.VipTwo {
		if 10 > uint64(user.Amount) {
			return &pb.OpenCardReply{Status: "账号余额不足10u"}, nil
		}
		cardAmount = 10
	} else {
		if 30 > uint64(user.Amount) {
			return &pb.OpenCardReply{Status: "账号余额不足30u"}, nil
		}
		cardAmount = 30
	}

	if 1 > len(req.SendBody.Email) || len(req.SendBody.Email) > 99 {
		return &pb.OpenCardReply{Status: "邮箱错误"}, nil
	}

	if 1 > len(req.SendBody.FirstName) || len(req.SendBody.FirstName) > 44 {
		return &pb.OpenCardReply{Status: "名字错误"}, nil
	}

	if 1 > len(req.SendBody.LastName) || len(req.SendBody.LastName) > 44 {
		return &pb.OpenCardReply{Status: "姓错误"}, nil
	}

	if 1 > len(req.SendBody.Phone) || len(req.SendBody.Phone) > 44 {
		return &pb.OpenCardReply{Status: "手机号错误"}, nil
	}

	if 1 > len(req.SendBody.CountryCode) || len(req.SendBody.CountryCode) > 44 {
		return &pb.OpenCardReply{Status: "国家代码错误"}, nil
	}

	if 1 > len(req.SendBody.Street) || len(req.SendBody.Street) > 99 {
		return &pb.OpenCardReply{Status: "街道错误"}, nil
	}

	if 1 > len(req.SendBody.City) || len(req.SendBody.City) > 99 {
		return &pb.OpenCardReply{Status: "城市错误"}, nil
	}

	//if 1 > len(req.SendBody.Country) || len(req.SendBody.Country) > 99 {
	//	return &pb.OpenCardReply{Status: "国家错误"}, nil
	//}

	if 1 > len(req.SendBody.PostalCode) || len(req.SendBody.PostalCode) > 99 {
		return &pb.OpenCardReply{Status: "邮政编码错误"}, nil
	}

	if 1 > len(req.SendBody.BirthDate) || len(req.SendBody.BirthDate) > 99 {
		return &pb.OpenCardReply{Status: "生日错误"}, nil
	}

	var (
		HolderID        string
		productIdUseTwo string
		maxCardQuotaTwo uint64
	)
	//if 5 < len(user.CardUserId) {
	//HolderID = user.CardUserId
	//productIdUseTwo = user.ProductId
	//maxCardQuotaTwo = user.MaxCardQuota
	//var productIdUseInt64 uint64
	//productIdUseInt64, err = strconv.ParseUint(user.ProductId, 10, 64)
	//if nil != err || 0 >= productIdUseInt64 {
	//	return &pb.OpenCardReply{Status: "获取产品信息错误"}, nil
	//}
	//
	//// 请求
	//var (
	//	resCreatCardholder *CreateCardholderResponse
	//)
	//resCreatCardholder, err = UpdateCardholderRequest(productIdUseInt64, &User{
	//	CardUserId:  HolderID,
	//	FirstName:   req.SendBody.FirstName,
	//	LastName:    req.SendBody.LastName,
	//	Email:       user.Email,
	//	CountryCode: req.SendBody.CountryCode,
	//	Phone:       req.SendBody.Phone,
	//	City:        req.SendBody.City,
	//	Country:     req.SendBody.Country,
	//	Street:      req.SendBody.Street,
	//	PostalCode:  req.SendBody.PostalCode,
	//	BirthDate:   req.SendBody.BirthDate,
	//})
	//if nil == resCreatCardholder || err != nil {
	//	fmt.Println("持卡人订单创建失败:", user, resCreatCardholder, err)
	//	return &pb.OpenCardReply{Status: "请求创建持卡人系统错误"}, nil
	//}
	//if 200 != resCreatCardholder.Code {
	//	fmt.Println("请求创建持卡人系统错误", user, resCreatCardholder, err)
	//	return &pb.OpenCardReply{Status: "请求创建持卡人系统错误" + resCreatCardholder.Msg}, nil
	//}
	//
	//if 0 > len(resCreatCardholder.Data.HolderID) {
	//	fmt.Println("持卡人订单信息错误", user, resCreatCardholder, err)
	//	return &pb.OpenCardReply{Status: "请求创建持卡人系统错误，信息缺失"}, nil
	//}
	//
	//if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
	//	err = uuc.repo.CreateCard(ctx, userId, &User{
	//		Amount:       10,
	//		CardUserId:   HolderID,
	//		MaxCardQuota: maxCardQuotaTwo,
	//		ProductId:    productIdUseTwo,
	//		FirstName:    req.SendBody.FirstName,
	//		LastName:     req.SendBody.LastName,
	//		Email:        user.Email,
	//		CountryCode:  req.SendBody.CountryCode,
	//		Phone:        req.SendBody.Phone,
	//		City:         req.SendBody.City,
	//		Country:      req.SendBody.Country,
	//		Street:       req.SendBody.Street,
	//		PostalCode:   req.SendBody.PostalCode,
	//		BirthDate:    req.SendBody.BirthDate,
	//	})
	//	if nil != err {
	//		return err
	//	}
	//
	//	return nil
	//}); nil != err {
	//	fmt.Println(err, "开卡写入mysql错误", user)
	//	return &pb.OpenCardReply{
	//		Status: "开卡错误，联系管理员",
	//	}, nil
	//}

	//} else {
	var (
		products          *CardProductListResponse
		productIdUse      string
		productIdUseInt64 uint64
		maxCardQuota      uint64
	)
	products, err = GetCardProducts()
	if nil == products || nil != err {
		//fmt.Println("产品信息错误1")
		return &pb.OpenCardReply{Status: "获取产品信息错误"}, nil
	}

	for _, v := range products.Rows {
		if 0 < len(v.ProductId) && "ENABLED" == v.ProductStatus {
			productIdUse = v.ProductId
			maxCardQuota = v.MaxCardQuota
			productIdUseInt64, err = strconv.ParseUint(productIdUse, 10, 64)
			if nil != err {
				//fmt.Println("产品信息错误2")
				return &pb.OpenCardReply{Status: "获取产品信息错误"}, nil
			}
			//fmt.Println("当前选择产品信息", productIdUse, maxCardQuota, v)
			break
		}
	}

	if 0 >= maxCardQuota {
		//fmt.Println("产品信息错误3")
		return &pb.OpenCardReply{Status: "获取产品信息错误,额度0"}, nil
	}

	if 0 >= productIdUseInt64 {
		//fmt.Println("产品信息错误4")
		return &pb.OpenCardReply{Status: "获取产品信息错误,产品id0"}, nil
	}

	// 请求
	var (
		resCreatCardholder *CreateCardholderResponse
	)
	resCreatCardholder, err = CreateCardholderRequest(productIdUseInt64, &User{
		FirstName:   req.SendBody.FirstName,
		LastName:    req.SendBody.LastName,
		Email:       req.SendBody.Email,
		CountryCode: req.SendBody.CountryCode,
		Phone:       req.SendBody.Phone,
		City:        req.SendBody.City,
		Country:     req.SendBody.Country,
		Street:      req.SendBody.Street,
		PostalCode:  req.SendBody.PostalCode,
		BirthDate:   req.SendBody.BirthDate,
	})
	if nil == resCreatCardholder || err != nil {
		fmt.Println("持卡人订单创建失败:", user, resCreatCardholder, err)
		return &pb.OpenCardReply{Status: "请求创建持卡人系统错误"}, nil
	}
	if 200 != resCreatCardholder.Code {
		fmt.Println("请求创建持卡人系统错误", user, resCreatCardholder, err)
		return &pb.OpenCardReply{Status: "请求创建持卡人系统错误" + resCreatCardholder.Msg}, nil
	}

	if 0 > len(resCreatCardholder.Data.HolderID) {
		fmt.Println("持卡人订单信息错误", user, resCreatCardholder, err)
		return &pb.OpenCardReply{Status: "请求创建持卡人系统错误，信息缺失"}, nil
	}

	fmt.Println("持卡人信息", user, resCreatCardholder)
	HolderID = resCreatCardholder.Data.HolderID
	maxCardQuotaTwo = maxCardQuota
	productIdUseTwo = productIdUse

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.CreateCard(ctx, userId, &User{
			Amount:       cardAmount,
			CardUserId:   HolderID,
			MaxCardQuota: maxCardQuotaTwo,
			ProductId:    productIdUseTwo,
			FirstName:    req.SendBody.FirstName,
			LastName:     req.SendBody.LastName,
			Email:        req.SendBody.Email,
			CountryCode:  req.SendBody.CountryCode,
			Phone:        req.SendBody.Phone,
			City:         req.SendBody.City,
			Country:      req.SendBody.Country,
			Street:       req.SendBody.Street,
			PostalCode:   req.SendBody.PostalCode,
			BirthDate:    req.SendBody.BirthDate,
		})
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		fmt.Println(err, "开卡写入mysql错误", user)
		return &pb.OpenCardReply{
			Status: "开卡错误，联系管理员",
		}, nil
	}
	//}

	return &pb.OpenCardReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) OpenCardTwo(ctx context.Context, req *pb.OpenCardRequest, userId uint64) (*pb.OpenCardReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	var (
		user       *User
		err        error
		cardAmount = float64(150)
	)

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.OpenCardReply{Status: "用户不存在"}, nil
	}

	if 0 < user.CardTwo {
		return &pb.OpenCardReply{Status: "已提交"}, nil
	}

	if 150 > uint64(user.Amount) {
		return &pb.OpenCardReply{Status: "账号余额不足150u"}, nil
	}

	if 1 > len(req.SendBody.Email) || len(req.SendBody.Email) > 99 {
		return &pb.OpenCardReply{Status: "邮箱错误"}, nil
	}

	if 1 > len(req.SendBody.FirstName) || len(req.SendBody.FirstName) > 44 {
		return &pb.OpenCardReply{Status: "名字错误"}, nil
	}

	if 1 > len(req.SendBody.LastName) || len(req.SendBody.LastName) > 44 {
		return &pb.OpenCardReply{Status: "姓错误"}, nil
	}

	if 1 > len(req.SendBody.Phone) || len(req.SendBody.Phone) > 44 {
		return &pb.OpenCardReply{Status: "手机号错误"}, nil
	}

	if 1 > len(req.SendBody.CountryCode) || len(req.SendBody.CountryCode) > 44 {
		return &pb.OpenCardReply{Status: "国家代码错误"}, nil
	}

	if 1 > len(req.SendBody.Street) || len(req.SendBody.Street) > 99 {
		return &pb.OpenCardReply{Status: "街道错误"}, nil
	}

	if 1 > len(req.SendBody.City) || len(req.SendBody.City) > 99 {
		return &pb.OpenCardReply{Status: "城市错误"}, nil
	}

	if 1 > len(req.SendBody.PostalCode) || len(req.SendBody.PostalCode) > 99 {
		return &pb.OpenCardReply{Status: "邮政编码错误"}, nil
	}

	if 1 > len(req.SendBody.BirthDate) || len(req.SendBody.BirthDate) > 99 {
		return &pb.OpenCardReply{Status: "生日错误"}, nil
	}

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.CreateCardTwo(ctx, userId, &User{
			Amount:      cardAmount,
			FirstName:   req.SendBody.FirstName,
			LastName:    req.SendBody.LastName,
			Email:       req.SendBody.Email,
			CountryCode: req.SendBody.CountryCode,
			Phone:       req.SendBody.Phone,
			City:        req.SendBody.City,
			Country:     req.SendBody.Country,
			Street:      req.SendBody.Street,
			PostalCode:  req.SendBody.PostalCode,
			BirthDate:   req.SendBody.BirthDate,
		})
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		fmt.Println(err, "开卡2写入mysql错误", user)
		return &pb.OpenCardReply{
			Status: "开卡错误，联系管理员",
		}, nil
	}

	return &pb.OpenCardReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) AmountToCard(ctx context.Context, req *pb.AmountToCardRequest, userId uint64) (*pb.AmountToCardReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	var (
		user *User
		err  error
	)
	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.AmountToCardReply{Status: "用户不存在"}, nil
	}

	if req.SendBody.Amount > uint64(user.Amount) {
		return &pb.AmountToCardReply{Status: "账号余额不足"}, nil
	}

	//if 100 > req.SendBody.Amount {
	//	return &pb.AmountToCardReply{Status: "划转最少100u"}, nil
	//}

	if 20 > req.SendBody.Amount {
		return &pb.AmountToCardReply{Status: "划转最少20u"}, nil
	}

	if 5 >= len(user.CardNumber) {
		return &pb.AmountToCardReply{Status: "无卡片记录"}, nil
	}

	if 5 >= len(user.Card) {
		return &pb.AmountToCardReply{Status: "无卡片记录"}, nil
	}

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.AmountToCard(ctx, userId, float64(req.SendBody.Amount))
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		fmt.Println(err, "划转写入mysql错误", user)
		return &pb.AmountToCardReply{
			Status: "划转错误，联系管理员",
		}, nil
	}

	// 划转
	var (
		cardRes *CardRechargeResponse
	)
	cardRes, err = RechargeCard(user.Card, req.SendBody.Amount)
	if nil != err || nil == cardRes || 200 != cardRes.Code {
		return &pb.AmountToCardReply{Status: "划转失败"}, nil
	}

	if "PROCESSING" != cardRes.Data.OrderStatus && "SUCCESS" != cardRes.Data.OrderStatus {
		return &pb.AmountToCardReply{
			Status: "创建订单失败，联系管理员",
		}, nil
	}

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.AmountToCardReward(ctx, userId, float64(req.SendBody.Amount), cardRes.Data.CardOrderID)
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		fmt.Println(err, "划转写入mysql错误2", user)
		return &pb.AmountToCardReply{
			Status: "划转错误2，联系管理员",
		}, nil
	}

	return &pb.AmountToCardReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) LookCard(ctx context.Context, req *pb.LookCardRequest, userId uint64) (*pb.LookCardReply, error) {
	var (
		user    *User
		err     error
		carInfo *CardSensitiveResponse
	)
	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.LookCardReply{Status: "用户不存在"}, nil
	}

	carInfo, err = GetCardSensitiveInfo(user.Card)
	if nil != err || nil == carInfo || 200 != carInfo.Code {
		return &pb.LookCardReply{Status: "获取数据失败"}, nil
	}

	return &pb.LookCardReply{
		Status:      "ok",
		CardNumber:  carInfo.Data.Pan,
		Pin:         carInfo.Data.Pin,
		Expire:      carInfo.Data.Expire,
		Cvv:         carInfo.Data.CVV,
		Email:       user.Email,
		Phone:       user.Phone,
		Country:     user.Country,
		CountryCode: user.CountryCode,
	}, nil
}

func (uuc *UserUseCase) AmountTo(ctx context.Context, req *pb.AmountToRequest, userId uint64) (*pb.AmountToReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	var (
		user   *User
		toUser *User
		err    error
	)
	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.AmountToReply{Status: "用户不存在"}, nil
	}

	if req.SendBody.Amount > uint64(user.Amount) {
		return &pb.AmountToReply{Status: "账号余额不足"}, nil
	}

	if 30 > len(req.SendBody.Address) || 60 < len(req.SendBody.Address) {
		return &pb.AmountToReply{Status: "账号参数格式不正确"}, nil
	}

	toUser, err = uuc.repo.GetUserByAddress(req.SendBody.Address)
	if nil == toUser || nil != err {
		return &pb.AmountToReply{Status: "目标用户不存在"}, nil
	}

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.AmountTo(ctx, userId, toUser.ID, toUser.Address, float64(req.SendBody.Amount))
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		fmt.Println(err, "划转写入mysql错误", user)
		return &pb.AmountToReply{
			Status: "划转错误，联系管理员",
		}, nil
	}

	return &pb.AmountToReply{
		Status: "ok",
	}, nil
}

func (uuc *UserUseCase) Withdraw(ctx context.Context, req *pb.WithdrawRequest, userId uint64) (*pb.WithdrawReply, error) {
	lockAmount.Lock()
	defer lockAmount.Unlock()

	var (
		user         *User
		err          error
		configs      []*Config
		withdrawRate float64
	)

	// 配置
	configs, err = uuc.repo.GetConfigByKeys("withdraw_rate")
	if nil != configs {
		for _, vConfig := range configs {
			if "withdraw_rate" == vConfig.KeyName {
				withdrawRate, _ = strconv.ParseFloat(vConfig.Value, 10)
			}
		}
	}

	user, err = uuc.repo.GetUserById(userId)
	if nil == user || nil != err {
		return &pb.WithdrawReply{Status: "用户不存在"}, nil
	}

	if req.SendBody.Amount > uint64(user.Amount) {
		return &pb.WithdrawReply{Status: "账号余额不足"}, nil
	}

	amountFloatSubFee := float64(req.SendBody.Amount) - float64(req.SendBody.Amount)*withdrawRate
	if 0 >= amountFloatSubFee {
		return &pb.WithdrawReply{Status: "手续费错误"}, nil
	}

	if err = uuc.tx.ExecTx(ctx, func(ctx context.Context) error { // 事务
		err = uuc.repo.Withdraw(ctx, userId, float64(req.SendBody.Amount), amountFloatSubFee, user.Address)
		if nil != err {
			return err
		}

		return nil
	}); nil != err {
		return &pb.WithdrawReply{
			Status: "提现错误，联系管理员",
		}, nil
	}

	return &pb.WithdrawReply{
		Status: "ok",
	}, nil
}

type CreateCardResponse struct {
	CardID      string `json:"cardId"`
	CardOrderID string `json:"cardOrderId"`
	CreateTime  string `json:"createTime"`
	CardStatus  string `json:"cardStatus"`
	OrderStatus string `json:"orderStatus"`
}

func GenerateSign(params map[string]interface{}, signKey string) string {
	// 1. 排除 sign 字段
	var keys []string
	for k := range params {
		if k != "sign" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 2. 拼接 key + value 字符串
	var sb strings.Builder
	sb.WriteString(signKey)

	for _, k := range keys {
		sb.WriteString(k)
		value := params[k]

		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case float64, int, int64, bool:
			strValue = fmt.Sprintf("%v", v)
		default:
			// map、slice 等复杂类型用 JSON 编码
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				strValue = ""
			} else {
				strValue = string(jsonBytes)
			}
		}
		sb.WriteString(strValue)
	}

	signString := sb.String()
	//fmt.Println("md5前字符串", signString)

	// 3. 进行 MD5 加密
	hash := md5.Sum([]byte(signString))
	return hex.EncodeToString(hash[:])
}

func CreateCardRequestWithSign() (*CreateCardResponse, error) {
	//url := "https://test-api.ispay.com/dev-api/vcc/api/v1/cards/create"
	//url := "https://www.ispay.com/prod-api/vcc/api/v1/cards/create"
	url := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/create"

	reqBody := map[string]interface{}{
		"merchantId":    "322338",
		"cardCurrency":  "USD",
		"cardAmount":    1000000,
		"cardholderId":  10001,
		"cardProductId": 20001,
		"cardSpendRule": map[string]interface{}{
			"dailyLimit":   250000,
			"monthlyLimit": 1000000,
		},
		"cardRiskControl": map[string]interface{}{
			"allowedMerchants": []string{"ONLINE"},
			"blockedCountries": []string{},
		},
	}

	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")
	// 请求体（包括嵌套结构）
	reqBody["sign"] = sign

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	//fmt.Println("请求报文:", string(jsonData))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	fmt.Println("响应报文:", string(body)) // ← 打印响应内容

	var result *CreateCardResponse
	if err = json.Unmarshal(body, result); err != nil {
		return nil, err
	}

	return result, nil
}

type CreateCardholderResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		HolderID    string `json:"holderId"`
		Email       string `json:"email"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		BirthDate   string `json:"birthDate"`
		CountryCode string `json:"countryCode"`
		PhoneNumber string `json:"phoneNumber"`

		DeliveryAddress DeliveryAddress `json:"deliveryAddress"`
		//ProofFile       ProofFile       `json:"proofFile"`
	} `json:"data"`
}

type DeliveryAddress struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Street  string `json:"street"`
}

type ProofFile struct {
	FileBase64 string `json:"fileBase64"`
	FileType   string `json:"fileType"`
}

func CreateCardholderRequest(productId uint64, user *User) (*CreateCardholderResponse, error) {
	//baseURL := "https://www.ispay.com/prod-api/vcc/api/v1/cards/holders/create"
	baseURL := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/holders/create"

	reqBody := map[string]interface{}{
		"productId":   productId,
		"merchantId":  "322338",
		"email":       user.Email,
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"birthDate":   user.BirthDate,
		"countryCode": user.CountryCode,
		"phoneNumber": user.Phone,
		"deliveryAddress": map[string]interface{}{
			"city":       user.City,
			"country":    user.CountryCode,
			"street":     user.Street,
			"postalCode": user.PostalCode,
		},
	}

	// 生成签名
	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9") // 用你的密钥替换
	reqBody["sign"] = sign

	// 构造请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json marshal error: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("new request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do error: %v", err)
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("响应报文:", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status not ok: %v", resp.StatusCode)
	}

	var result CreateCardholderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}

	return &result, nil
}

func UpdateCardholderRequest(productId uint64, user *User) (*CreateCardholderResponse, error) {
	//baseURL := "https://www.ispay.com/prod-api/vcc/api/v1/cards/holders/create"
	baseURL := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/holders/update"

	reqBody := map[string]interface{}{
		"holderId":    user.CardUserId,
		"productId":   productId,
		"merchantId":  "322338",
		"email":       user.Email,
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"birthDate":   user.BirthDate,
		"countryCode": user.CountryCode,
		"phoneNumber": user.Phone,
		"deliveryAddress": map[string]interface{}{
			"city":       user.City,
			"country":    user.CountryCode,
			"street":     user.Street,
			"postalCode": user.PostalCode,
		},
	}

	// 生成签名
	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9") // 用你的密钥替换
	reqBody["sign"] = sign

	// 构造请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json marshal error: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("new request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do error: %v", err)
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("响应报文:", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status not ok: %v", resp.StatusCode)
	}

	var result CreateCardholderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}

	return &result, nil
}

type CardProductListResponse struct {
	Total int           `json:"total"`
	Rows  []CardProduct `json:"rows"`
	Code  int           `json:"code"`
	Msg   string        `json:"msg"`
}

type CardProduct struct {
	ProductId          string       `json:"productId"` // ← 改成 string
	ProductName        string       `json:"productName"`
	ModeType           string       `json:"modeType"`
	CardBin            string       `json:"cardBin"`
	CardForm           []string     `json:"cardForm"`
	MaxCardQuota       uint64       `json:"maxCardQuota"`
	CardScheme         string       `json:"cardScheme"`
	NoPinPaymentAmount []AmountItem `json:"noPinPaymentAmount"`
	CardCurrency       []string     `json:"cardCurrency"`
	CreateTime         string       `json:"createTime"`
	UpdateTime         string       `json:"updateTime"`
	ProductStatus      string       `json:"productStatus"`
}

type AmountItem struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func GetCardProducts() (*CardProductListResponse, error) {
	baseURL := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/products/all"

	reqBody := map[string]interface{}{
		"merchantId": "322338",
	}

	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")

	params := url.Values{}
	params.Set("merchantId", "322338")
	params.Set("sign", sign)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//fmt.Println("响应报文:", string(body))

	var result CardProductListResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("JSON 解析失败:", err)
		return nil, err
	}

	//fmt.Println(result)

	return &result, nil
}

type CardSensitiveResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Pan    string `json:"pan"`
		Pin    string `json:"pin"`
		CVV    string `json:"cvv"`
		Expire string `json:"expire"`
	} `json:"data"`
}

func GetCardSensitiveInfo(cardId string) (*CardSensitiveResponse, error) {
	//baseUrl := "https://www.ispay.com/prod-api/vcc/api/v1/cards/sensitive"
	baseUrl := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/sensitive"

	reqBody := map[string]interface{}{
		"merchantId": "322338",
		"cardId":     cardId,
	}

	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")
	reqBody["sign"] = sign

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", baseUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败: %s", string(body))
	}

	//fmt.Println("响应报文:", string(body))

	var result CardSensitiveResponse
	if err = json.Unmarshal(body, &result); err != nil {
		fmt.Println("敏感信息 JSON 解析失败:", err)
		return nil, err
	}

	return &result, nil
}

type CardRechargeResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		CardID       string `json:"cardId"`
		CardOrderID  string `json:"cardOrderId"`
		OrderType    string `json:"orderType"`
		CardCurrency string `json:"cardCurrency"`
		CreateTime   string `json:"createTime"`
		UpdateTime   string `json:"updateTime"`
		CompleteTime string `json:"completeTime"`
		OrderStatus  string `json:"orderStatus"`
	} `json:"data"`
}

func RechargeCard(cardId string, rechargeAmount uint64) (*CardRechargeResponse, error) {
	//baseUrl := "https://www.ispay.com/prod-api/vcc/api/v1/cards/recharge"
	baseUrl := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/recharge"

	reqBody := map[string]interface{}{
		"merchantId":     "322338",
		"cardId":         cardId,
		"rechargeAmount": rechargeAmount,
	}

	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")
	reqBody["sign"] = sign

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", baseUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败: %s", string(body))
	}

	fmt.Println("响应报文:", string(body))

	var result CardRechargeResponse
	if err = json.Unmarshal(body, &result); err != nil {
		fmt.Println("充值响应解析失败:", err)
		return nil, err
	}

	return &result, nil
}

type CardTransactionListResponse struct {
	Code  uint64                  `json:"code"`  // 接口状态码
	Msg   string                  `json:"msg"`   // 返回消息
	Total uint64                  `json:"total"` // 总条数
	Rows  []CardTransactionRecord `json:"rows"`  // 交易列表
}

type CardTransactionRecord struct {
	ID                      string                 `json:"id"`
	Pan                     string                 `json:"pan"`
	TradeNo                 string                 `json:"tradeNo"`
	Type                    string                 `json:"type"`
	Status                  string                 `json:"status"`
	TradeAmount             string                 `json:"tradeAmount"`
	TradeCurrency           string                 `json:"tradeCurrency"`
	Timestamp               string                 `json:"timestamp"`
	ServiceFee              string                 `json:"serviceFee"`
	ActualTransactionAmount string                 `json:"actualTransactionAmount"`
	CurrentBalance          string                 `json:"currentBalance"`
	CreateTime              string                 `json:"createTime"`
	TradeDescription        string                 `json:"tradeDescription"`
	MerchantData            map[string]interface{} `json:"merchantData"` // 用 map 保证兼容性
}

func GetCardTransactionList(cardId, pageNum, pageSize uint64) (*CardTransactionListResponse, error) {
	baseUrl := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/transactions/list"

	// 1. 构造参数（全部为一级扁平字段）
	reqParams := map[string]interface{}{
		"merchantId":    "322338",
		"cardId":        cardId,
		"pageSize":      pageSize,
		"pageNum":       pageNum,
		"orderByColumn": "createTime",
		"isAsc":         "desc",
	}

	// 2. 生成签名（假设你有此函数）
	sign := GenerateSign(reqParams, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")
	reqParams["sign"] = sign

	// 3. 构造 query string
	query := url.Values{}
	for k, v := range reqParams {
		query.Set(k, fmt.Sprintf("%v", v))
	}

	fullUrl := baseUrl + "?" + query.Encode()
	//fmt.Println("请求 URL:", fullUrl)

	// 4. 发起 GET 请求
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	//fmt.Println("响应报文:", string(body))

	// 5. 解析响应
	var result CardTransactionListResponse
	if err = json.Unmarshal(body, &result); err != nil {
		fmt.Println("JSON 解析失败:", err)
		return nil, err
	}

	return &result, nil
}

type CardInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		CardID     string `json:"cardId"`
		Pan        string `json:"pan"`
		CardStatus string `json:"cardStatus"`
		Balance    string `json:"balance"`
		Holder     struct {
			HolderID string `json:"holderId"`
		} `json:"holder"`
	} `json:"data"`
}

func GetCardInfoRequestWithSign(cardId string) (*CardInfoResponse, error) {
	baseUrl := "http://120.79.173.55:9102/prod-api/vcc/api/v1/cards/info"
	//baseUrl := "https://www.ispay.com/prod-api/vcc/api/v1/cards/info"

	reqBody := map[string]interface{}{
		"merchantId": "322338",
		"cardId":     cardId, // 如果需要传 cardId，根据实际接口文档添加
	}

	sign := GenerateSign(reqBody, "j4gqNRcpTDJr50AP2xd9obKWZIKWbeo9")
	reqBody["sign"] = sign

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", baseUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Language", "zh_CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		errTwo := Body.Close()
		if errTwo != nil {

		}
	}(resp.Body)

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", string(body))
	}

	//fmt.Println("响应报文:", string(body))

	var result CardInfoResponse
	if err = json.Unmarshal(body, &result); err != nil {
		fmt.Println("卡信息 JSON 解析失败:", err)
		return nil, err
	}

	return &result, nil
}
