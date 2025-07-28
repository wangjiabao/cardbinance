package data

import (
	"cardbinance/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type User struct {
	ID            uint64    `gorm:"primarykey;type:int"`
	Address       string    `gorm:"type:varchar(100)"`
	Card          string    `gorm:"type:varchar(100)"`
	CardOrderId   string    `gorm:"type:varchar(100)"`
	CardNumber    string    `gorm:"type:varchar(100)"`
	CardAmount    float64   `gorm:"type:decimal(65,20);not null"`
	Amount        float64   `gorm:"type:decimal(65,20);"`
	IsDelete      uint64    `gorm:"type:int"`
	Vip           uint64    `gorm:"type:int"`
	MyTotalAmount uint64    `gorm:"type:bigint"`
	AmountTwo     uint64    `gorm:"type:bigint"`
	CreatedAt     time.Time `gorm:"type:datetime;not null"`
	UpdatedAt     time.Time `gorm:"type:datetime;not null"`
}

type UserRecommend struct {
	ID            uint64    `gorm:"primarykey;type:int"`
	UserId        uint64    `gorm:"type:int;not null"`
	RecommendCode string    `gorm:"type:varchar(10000);not null"`
	CreatedAt     time.Time `gorm:"type:datetime;not null"`
	UpdatedAt     time.Time `gorm:"type:datetime;not null"`
}

type Config struct {
	ID        int64     `gorm:"primarykey;type:int"`
	Name      string    `gorm:"type:varchar(45);not null"`
	KeyName   string    `gorm:"type:varchar(45);not null"`
	Value     string    `gorm:"type:varchar(1000);not null"`
	CreatedAt time.Time `gorm:"type:datetime;not null"`
	UpdatedAt time.Time `gorm:"type:datetime;not null"`
}

type UserRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &UserRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (u *UserRepo) GetUserByAddress(address string) (*biz.User, error) {
	var user User
	if err := u.data.db.Where("address=?", address).Table("user").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.New(500, "USER ERROR", err.Error())
	}

	return &biz.User{
		CardAmount:    user.CardAmount,
		MyTotalAmount: user.MyTotalAmount,
		AmountTwo:     user.AmountTwo,
		IsDelete:      user.IsDelete,
		Vip:           user.Vip,
		ID:            user.ID,
		Address:       user.Address,
		Card:          user.Card,
		Amount:        user.Amount,
		CardNumber:    user.CardNumber,
		CardOrderId:   user.CardOrderId,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}, nil
}

func (u *UserRepo) GetUserById(userId uint64) (*biz.User, error) {
	var user User
	if err := u.data.db.Where("id=?", userId).Table("user").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.New(500, "USER ERROR", err.Error())
	}

	return &biz.User{
		CardAmount:    user.CardAmount,
		MyTotalAmount: user.MyTotalAmount,
		AmountTwo:     user.AmountTwo,
		IsDelete:      user.IsDelete,
		Vip:           user.Vip,
		ID:            user.ID,
		Address:       user.Address,
		Card:          user.Card,
		Amount:        user.Amount,
		CardNumber:    user.CardNumber,
		CardOrderId:   user.CardOrderId,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}, nil
}

// GetUserRecommendByUserId .
func (u *UserRepo) GetUserRecommendByUserId(userId uint64) (*biz.UserRecommend, error) {
	var userRecommend UserRecommend
	if err := u.data.db.Where("user_id=?", userId).Table("user_recommend").First(&userRecommend).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.New(500, "USER RECOMMEND ERROR", err.Error())
	}

	return &biz.UserRecommend{
		UserId:        userRecommend.UserId,
		RecommendCode: userRecommend.RecommendCode,
	}, nil
}

// CreateUser .
func (u *UserRepo) CreateUser(ctx context.Context, uc *biz.User) (*biz.User, error) {
	var user User
	user.Address = uc.Address
	if 0 < uc.Vip {
		user.Vip = uc.Vip
	}

	res := u.data.DB(ctx).Table("user").Create(&user)
	if res.Error != nil || 0 >= res.RowsAffected {
		return nil, errors.New(500, "CREATE_USER_ERROR", "用户创建失败")
	}

	return &biz.User{
		CardAmount:    user.CardAmount,
		MyTotalAmount: user.MyTotalAmount,
		AmountTwo:     user.AmountTwo,
		IsDelete:      user.IsDelete,
		Vip:           user.Vip,
		ID:            user.ID,
		Address:       user.Address,
		Card:          user.Card,
		Amount:        user.Amount,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		CardNumber:    user.CardNumber,
		CardOrderId:   user.CardOrderId,
	}, nil
}

// CreateUserRecommend .
func (u *UserRepo) CreateUserRecommend(ctx context.Context, userId uint64, recommendUser *biz.UserRecommend) (*biz.UserRecommend, error) {
	var tmpRecommendCode string
	if nil != recommendUser && 0 < recommendUser.UserId {
		tmpRecommendCode = "D" + strconv.FormatUint(recommendUser.UserId, 10)
		if "" != recommendUser.RecommendCode {
			tmpRecommendCode = recommendUser.RecommendCode + tmpRecommendCode
		}
	}

	var userRecommend UserRecommend
	userRecommend.UserId = userId
	userRecommend.RecommendCode = tmpRecommendCode

	res := u.data.DB(ctx).Table("user_recommend").Create(&userRecommend)
	if res.Error != nil || 0 >= res.RowsAffected {
		return nil, errors.New(500, "CREATE_USER_RECOMMEND_ERROR", "用户推荐关系创建失败")
	}

	return &biz.UserRecommend{
		ID:            userRecommend.ID,
		UserId:        userRecommend.UserId,
		RecommendCode: userRecommend.RecommendCode,
	}, nil
}

// GetUserRecommendByCode .
func (u *UserRepo) GetUserRecommendByCode(code string) ([]*biz.UserRecommend, error) {
	var (
		userRecommends []*UserRecommend
	)
	res := make([]*biz.UserRecommend, 0)

	instance := u.data.db.Table("user_recommend").Where("recommend_code=?", code)
	if err := instance.Find(&userRecommends).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}

		return nil, errors.New(500, "USER RECOMMEND ERROR", err.Error())
	}

	for _, userRecommend := range userRecommends {
		res = append(res, &biz.UserRecommend{
			UserId:        userRecommend.UserId,
			RecommendCode: userRecommend.RecommendCode,
			CreatedAt:     userRecommend.CreatedAt,
		})
	}

	return res, nil
}

// GetUserByUserIds .
func (u *UserRepo) GetUserByUserIds(userIds []uint64) (map[uint64]*biz.User, error) {
	var users []*User

	res := make(map[uint64]*biz.User, 0)
	if err := u.data.db.Table("user").Where("id IN (?)", userIds).Find(&users).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, errors.NotFound("USER_NOT_FOUND", "user not found")
		}

		return nil, errors.New(500, "USER ERROR", err.Error())
	}

	for _, user := range users {
		res[user.ID] = &biz.User{
			CardAmount:    user.CardAmount,
			MyTotalAmount: user.MyTotalAmount,
			AmountTwo:     user.AmountTwo,
			IsDelete:      user.IsDelete,
			Vip:           user.Vip,
			ID:            user.ID,
			Address:       user.Address,
			Card:          user.Card,
			Amount:        user.Amount,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
			CardNumber:    user.CardNumber,
			CardOrderId:   user.CardOrderId,
		}
	}

	return res, nil
}

// GetConfigByKeys .
func (u *UserRepo) GetConfigByKeys(keys ...string) ([]*biz.Config, error) {
	var configs []*Config
	res := make([]*biz.Config, 0)
	if err := u.data.db.Where("key_name IN (?)", keys).Table("config").Find(&configs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.New(500, "Config ERROR", err.Error())
	}

	for _, config := range configs {
		res = append(res, &biz.Config{
			ID:      config.ID,
			KeyName: config.KeyName,
			Name:    config.Name,
			Value:   config.Value,
		})
	}

	return res, nil
}
