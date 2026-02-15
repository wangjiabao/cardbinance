package data

import (
	"cardbinance/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type User struct {
	ID            uint64    `gorm:"primarykey;type:int"`
	Address       string    `gorm:"type:varchar(100);default:'no'"`
	Card          string    `gorm:"type:varchar(100);not null;default:'no'"`
	CardOrderId   string    `gorm:"type:varchar(100);not null;default:'no'"`
	CardIdTwo     string    `gorm:"type:varchar(100);not null;default:'no'"`
	CardNumber    string    `gorm:"type:varchar(100);not null;default:'no'"`
	CardNumberTwo string    `gorm:"type:varchar(100);not null;default:'no'"`
	CardAmount    float64   `gorm:"type:decimal(65,20);not null"`
	Amount        float64   `gorm:"type:decimal(65,20)"`
	IsDelete      uint64    `gorm:"type:int"`
	Vip           uint64    `gorm:"type:int"`
	MyTotalAmount uint64    `gorm:"type:bigint"`
	AmountTwo     uint64    `gorm:"type:bigint"`
	FirstName     string    `gorm:"type:varchar(45);not null;default:'no'"`
	LastName      string    `gorm:"type:varchar(45);not null;default:'no'"`
	Email         string    `gorm:"type:varchar(100);not null;default:'no'"`
	CountryCode   string    `gorm:"type:varchar(45);not null;default:'no'"`
	Phone         string    `gorm:"type:varchar(45);not null;default:'no'"`
	City          string    `gorm:"type:varchar(100);not null;default:'no'"`
	Country       string    `gorm:"type:varchar(100);not null;default:'no'"`
	Street        string    `gorm:"type:varchar(100);not null;default:'no'"`
	PostalCode    string    `gorm:"type:varchar(45);not null;default:'no'"`
	BirthDate     string    `gorm:"type:varchar(45);not null;default:'no'"`
	MaxCardQuota  uint64    `gorm:"type:bigint"`
	ProductId     string    `gorm:"type:varchar(45);not null;default:'0'"`
	CardUserId    string    `gorm:"type:varchar(45);not null;default:'0'"`
	CreatedAt     time.Time `gorm:"type:datetime;not null"`
	UpdatedAt     time.Time `gorm:"type:datetime;not null"`
	UserCount     uint64    `gorm:"type:int"`
	VipTwo        uint64    `gorm:"type:int"`
	CardTwo       uint64    `gorm:"type:int"`
	CanVip        uint64    `gorm:"type:int"`
	VipThree      uint64    `gorm:"type:int"`
	CardType      uint64    `gorm:"type:int"`
}

type CardTwo struct {
	ID          uint64    `gorm:"primarykey;type:int"`
	UserId      uint64    `gorm:"type:int;not null"`
	FirstName   string    `gorm:"type:varchar(45);not null;default:'no'"`
	LastName    string    `gorm:"type:varchar(45);not null;default:'no'"`
	Email       string    `gorm:"type:varchar(100);not null;default:'no'"`
	CountryCode string    `gorm:"type:varchar(45);not null;default:'no'"`
	Phone       string    `gorm:"type:varchar(45);not null;default:'no'"`
	City        string    `gorm:"type:varchar(100);not null;default:'no'"`
	Country     string    `gorm:"type:varchar(100);not null;default:'no'"`
	Street      string    `gorm:"type:varchar(100);not null;default:'no'"`
	PostalCode  string    `gorm:"type:varchar(45);not null;default:'no'"`
	BirthDate   string    `gorm:"type:varchar(45);not null;default:'no'"`
	CreatedAt   time.Time `gorm:"type:datetime;not null"`
	UpdatedAt   time.Time `gorm:"type:datetime;not null"`
}

type CardRecord struct {
	ID         uint64    `gorm:"primarykey;type:int"`
	UserId     uint64    `gorm:"type:int;not null"`
	RecordType uint64    `gorm:"type:int;not null"`
	Remark     string    `gorm:"type:varchar(500);not null"`
	Code       string    `gorm:"type:varchar(100);not null"`
	Opt        string    `gorm:"type:varchar(100);not null"`
	CreatedAt  time.Time `gorm:"type:datetime;not null"`
	UpdatedAt  time.Time `gorm:"type:datetime;not null"`
}

type UserRecommend struct {
	ID            uint64    `gorm:"primarykey;type:int"`
	UserId        uint64    `gorm:"type:int;not null"`
	RecommendCode string    `gorm:"type:varchar(10000);not null"`
	CreatedAt     time.Time `gorm:"type:datetime;not null"`
	UpdatedAt     time.Time `gorm:"type:datetime;not null"`
}

type Config struct {
	ID        uint64    `gorm:"primarykey;type:int"`
	Name      string    `gorm:"type:varchar(45);not null"`
	KeyName   string    `gorm:"type:varchar(45);not null"`
	Value     string    `gorm:"type:varchar(1000);not null"`
	CreatedAt time.Time `gorm:"type:datetime;not null"`
	UpdatedAt time.Time `gorm:"type:datetime;not null"`
}

type Reward struct {
	ID        uint64    `gorm:"primarykey;type:int"`
	UserId    uint64    `gorm:"type:int;not null"`
	Amount    float64   `gorm:"type:decimal(65,20);not null"`
	Reason    uint64    `gorm:"type:int;not null"`
	CreatedAt time.Time `gorm:"type:datetime;not null"`
	UpdatedAt time.Time `gorm:"type:datetime;not null"`
	Address   string    `gorm:"type:varchar(100);not null"`
	One       uint64    `gorm:"type:int;not null"`
}

type Withdraw struct {
	ID        uint64    `gorm:"primarykey;type:int"`
	UserId    uint64    `gorm:"type:int"`
	Amount    float64   `gorm:"type:decimal(65,20);not null"`
	RelAmount float64   `gorm:"type:decimal(65,20);not null"`
	Status    string    `gorm:"type:varchar(45);not null"`
	Address   string    `gorm:"type:varchar(45);not null"`
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

func (u *UserRepo) SetNonceByAddress(ctx context.Context, wallet string) (int64, error) {
	key := "wallet:" + wallet

	val, err := u.data.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// 设置键值，60 秒后自动过期
		timestamp := time.Now().Unix()
		return timestamp, u.data.rdb.Set(ctx, key, timestamp, 60*time.Second).Err()

	} else if err != nil {
		return -1, err
	}

	// 转换为 int64 时间戳
	t, errThree := strconv.ParseInt(val, 10, 64)
	if errThree != nil {
		return 0, errThree
	}

	return t, nil
}

// GetAndDeleteWalletTimestamp 获取并删除，确保只用一次（无并发可用）
func (u *UserRepo) GetAndDeleteWalletTimestamp(ctx context.Context, wallet string) (string, error) {
	key := "wallet:" + wallet

	val, err := u.data.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", err
	}

	// 删除
	if errTwo := u.data.rdb.Del(ctx, key).Err(); errTwo != nil {
		return "", errTwo
	}

	return val, nil
}

func (u *UserRepo) SetLockAmountToCardByAddress(ctx context.Context, wallet string) error {
	return u.data.rdb.Set(ctx, "wallet:"+wallet+"amountocard", "lock", 60*time.Second).Err()
}

func (u *UserRepo) GetLockAmountToCardByAddress(ctx context.Context, wallet string) (string, error) {
	key := "wallet:" + wallet + "amountocard"

	val, err := u.data.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}

	return val, nil
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
		CardUserId:    user.CardUserId,
		ProductId:     user.ProductId,
		MaxCardQuota:  user.MaxCardQuota,
		Email:         user.Email,
		UserCount:     user.UserCount,
		Country:       user.Country,
		CountryCode:   user.CountryCode,
		Phone:         user.Phone,
		VipTwo:        user.VipTwo,
		CardTwo:       user.CardTwo,
		CanVip:        user.CanVip,
		VipThree:      user.VipThree,
		CardNumberTwo: user.CardNumberTwo,
		CardType:      user.CardType,
		CardIdTwo:     user.CardIdTwo,
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
		CardUserId:    user.CardUserId,
		ProductId:     user.ProductId,
		MaxCardQuota:  user.MaxCardQuota,
		Email:         user.Email,
		UserCount:     user.UserCount,
		Country:       user.Country,
		CountryCode:   user.CountryCode,
		Phone:         user.Phone,
		VipTwo:        user.VipTwo,
		CardTwo:       user.CardTwo,
		CanVip:        user.CanVip,
		VipThree:      user.VipThree,
		CardNumberTwo: user.CardNumberTwo,
		CardType:      user.CardType,
		CardIdTwo:     user.CardIdTwo,
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
	user.Card = "no"
	user.CardNumber = "no"
	user.CardOrderId = "no"
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

// GetUserRecommendLikeCode .
func (u *UserRepo) GetUserRecommendLikeCode(code string) ([]*biz.UserRecommend, error) {
	var (
		userRecommends []*UserRecommend
	)
	res := make([]*biz.UserRecommend, 0)

	instance := u.data.db.Table("user_recommend").Where("recommend_code Like ?", code+"%")
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
			VipThree:      user.VipThree,
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

// CreateCard .
func (u *UserRepo) CreateCard(ctx context.Context, userId uint64, user *biz.User) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).
		Where("amount>=?", user.Amount).
		Where("card_user_id=?", "0").
		Where("card_order_id=?", "no").
		Updates(map[string]interface{}{
			"amount":         gorm.Expr("amount - ?", user.Amount),
			"user_count":     gorm.Expr("user_count + ?", 1),
			"card_order_id":  "do",
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"email":          user.Email,
			"phone":          user.Phone,
			"country_code":   user.CountryCode,
			"country":        user.Country,
			"city":           user.City,
			"street":         user.Street,
			"postal_code":    user.PostalCode,
			"birth_date":     user.BirthDate,
			"max_card_quota": user.MaxCardQuota,
			"card_user_id":   user.CardUserId,
			"product_id":     user.ProductId,
			"updated_at":     time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = user.Amount
	reward.Reason = 3 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

func (u *UserRepo) UpdateCardNew(ctx context.Context, userId uint64, user *biz.User) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).
		Where("amount>=?", user.Amount).
		Where("card_order_id=?", "no").
		Updates(map[string]interface{}{
			"amount":         gorm.Expr("amount - ?", user.Amount),
			"user_count":     gorm.Expr("user_count + ?", 1),
			"card_order_id":  "do",
			"max_card_quota": user.MaxCardQuota,
			"product_id":     user.ProductId,
			"updated_at":     time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = user.Amount
	reward.Reason = 3 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// CreateCardTwoNew .
func (u *UserRepo) CreateCardTwoNew(ctx context.Context, userId uint64, user *biz.User) error {
	res := u.data.DB(ctx).Table("user").
		Where("id=?", userId).
		Where("card_user_id=?", "0").
		Where("amount>=?", user.Amount).
		Where("card_two=?", 0).
		Updates(map[string]interface{}{
			"amount":             gorm.Expr("amount - ?", user.Amount),
			"user_count":         gorm.Expr("user_count + ?", 1),
			"first_name":         user.FirstName,
			"last_name":          user.LastName,
			"email":              user.Email,
			"phone":              user.Phone,
			"country_code":       user.CountryCode,
			"country":            user.Country,
			"city":               user.City,
			"street":             user.Street,
			"postal_code":        user.PostalCode,
			"birth_date":         user.BirthDate,
			"max_card_quota_two": user.MaxCardQuota,
			"card_user_id":       user.CardUserId,
			"product_id_two":     user.ProductId,
			"updated_at":         time.Now().Format("2006-01-02 15:04:05"),
			"card_two":           1,
			"card_number_two":    user.CardNumberTwo,
			"card_type":          user.CardType,
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = user.Amount
	reward.Reason = 9 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// UpdateCardTwoNew .
func (u *UserRepo) UpdateCardTwoNew(ctx context.Context, userId uint64, user *biz.User) error {
	res := u.data.DB(ctx).Table("user").
		Where("id=?", userId).
		Where("amount>=?", user.Amount).
		Where("card_two=?", 0).
		Updates(map[string]interface{}{
			"amount":             gorm.Expr("amount - ?", user.Amount),
			"user_count":         gorm.Expr("user_count + ?", 1),
			"max_card_quota_two": user.MaxCardQuota,
			"product_id_two":     user.ProductId,
			"updated_at":         time.Now().Format("2006-01-02 15:04:05"),
			"card_two":           1,
			"card_number_two":    user.CardNumberTwo,
			"card_type":          user.CardType,
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = user.Amount
	reward.Reason = 9 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// UpdateCardTwoActive .
func (u *UserRepo) UpdateCardTwoActive(ctx context.Context, userId uint64) error {
	res := u.data.DB(ctx).Table("user").
		Where("id=?", userId).
		Updates(map[string]interface{}{
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
			"card_two":   3,
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	return nil
}

// CreateCardTwo .
func (u *UserRepo) CreateCardTwo(ctx context.Context, userId uint64, user *biz.User) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("amount>=?", user.Amount).Where("card_two=?", 0).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount - ?", user.Amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
			"card_two":   1,
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = user.Amount
	reward.Reason = 9 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	var (
		cardTwo CardTwo
	)

	cardTwo.UserId = userId
	cardTwo.Phone = user.Phone
	cardTwo.Street = user.Street
	cardTwo.PostalCode = user.PostalCode
	cardTwo.BirthDate = user.BirthDate
	cardTwo.FirstName = user.FirstName
	cardTwo.LastName = user.LastName
	cardTwo.Email = user.Email
	cardTwo.CountryCode = user.CountryCode
	cardTwo.Country = user.Country
	cardTwo.City = user.City

	resInsertTwo := u.data.DB(ctx).Table("card_two").Create(&cardTwo)
	if resInsertTwo.Error != nil || 0 >= resInsertTwo.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// SetVip .
func (u *UserRepo) SetVip(ctx context.Context, userId uint64, vip uint64) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).
		Updates(map[string]interface{}{
			"vip":        vip,
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	return nil
}

// UpdateCard .
func (u *UserRepo) UpdateCard(ctx context.Context, userId uint64, cardOrderId, card string) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("card_order_id=?", "no").
		Updates(map[string]interface{}{
			"card_order_id": cardOrderId,
			"card":          card,
			"updated_at":    time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	return nil
}

// GetAllUsers .
func (u *UserRepo) GetAllUsers() ([]*biz.User, error) {
	var users []*User
	if err := u.data.db.Table("user").Order("id asc").Find(&users).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.New(500, "USER ERROR", err.Error())
	}

	res := make([]*biz.User, 0)
	for _, user := range users {
		res = append(res, &biz.User{
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
		})
	}
	return res, nil
}

// GetUsersOpenCard .
func (u *UserRepo) GetUsersOpenCard() ([]*biz.User, error) {
	var users []*User

	res := make([]*biz.User, 0)
	if err := u.data.db.Table("user").Where("card_order_id=?", "do").Order("id asc").Find(&users).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}

		return nil, errors.New(500, "USER ERROR", err.Error())
	}

	for _, user := range users {
		res = append(res, &biz.User{
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
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			Email:         user.Email,
			CountryCode:   user.CountryCode,
			Phone:         user.Phone,
			City:          user.City,
			Country:       user.Country,
			Street:        user.Street,
			PostalCode:    user.PostalCode,
			BirthDate:     user.BirthDate,
		})
	}
	return res, nil
}

// CreateCardRecommend .
func (u *UserRepo) CreateCardRecommend(ctx context.Context, userId uint64, amount float64, vip uint64, address string) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("vip=?", vip).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount + ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}
	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.One = vip
	reward.Reason = 6 // 给我分红的理由
	reward.Address = address
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// AmountToCard .
func (u *UserRepo) AmountToCard(ctx context.Context, userId uint64, amount float64) (uint64, error) {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("amount>=?", amount).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount - ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return 0, errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 14 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return 0, errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return reward.ID, nil
}

// AmountToCardTwo .
func (u *UserRepo) AmountToCardTwo(ctx context.Context, userId uint64, amount float64) (uint64, error) {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("amount>=?", amount).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount - ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return 0, errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 144 // 给我分红的理由
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return 0, errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return reward.ID, nil
}

// AmountToCardReward .
func (u *UserRepo) AmountToCardReward(ctx context.Context, userId uint64, amount float64, orderId string, rewardId uint64) error {
	res := u.data.DB(ctx).Table("reward").Where("id=?", rewardId).
		Updates(map[string]interface{}{
			"one":        1,
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_REWARD_ERROR", "划转信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 4 // 给我分红的理由
	reward.Address = orderId
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// AmountToCardTwoReward .
func (u *UserRepo) AmountToCardTwoReward(ctx context.Context, userId uint64, amount float64, orderId string, rewardId uint64) error {
	res := u.data.DB(ctx).Table("reward").Where("id=?", rewardId).
		Updates(map[string]interface{}{
			"one":        1,
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_REWARD_ERROR", "划转信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 1444 // 给我分红的理由
	reward.Address = orderId
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// AmountTo .
func (u *UserRepo) AmountTo(ctx context.Context, userId, toUserId uint64, toAddress string, amount float64) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("amount>=?", amount).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount - ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	resTwo := u.data.DB(ctx).Table("user").Where("id=?", toUserId).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount + ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if resTwo.Error != nil || 0 >= resTwo.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 5 // 给我分红的理由
	reward.Address = toAddress
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// Withdraw .
func (u *UserRepo) Withdraw(ctx context.Context, userId uint64, amount, amountRel float64, address string) error {
	res := u.data.DB(ctx).Table("user").Where("id=?", userId).Where("amount>=?", amount).
		Updates(map[string]interface{}{
			"amount":     gorm.Expr("amount - ?", amount),
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil || 0 >= res.RowsAffected {
		return errors.New(500, "UPDATE_USER_ERROR", "用户信息修改失败")
	}

	var withdraw Withdraw
	withdraw.UserId = userId
	withdraw.Amount = amount
	withdraw.RelAmount = amountRel
	withdraw.Status = "rewarded"
	withdraw.Address = address
	resTwo := u.data.DB(ctx).Table("withdraw").Create(&withdraw)
	if resTwo.Error != nil || 0 >= resTwo.RowsAffected {
		return errors.New(500, "CREATE_WITHDRAW_ERROR", "提现记录创建失败")
	}

	var (
		reward Reward
	)

	reward.UserId = userId
	reward.Amount = amount
	reward.Reason = 2 // 给我分红的理由
	reward.Address = address
	resInsert := u.data.DB(ctx).Table("reward").Create(&reward)
	if resInsert.Error != nil || 0 >= resInsert.RowsAffected {
		return errors.New(500, "CREATE_LOCATION_ERROR", "信息创建失败")
	}

	return nil
}

// GetUserRewardByUserIdPage .
func (u *UserRepo) GetUserRewardByUserIdPage(ctx context.Context, b *biz.Pagination, userId uint64, reason uint64) ([]*biz.Reward, error, int64) {
	var (
		count   int64
		rewards []*Reward
	)

	res := make([]*biz.Reward, 0)

	instance := u.data.db.Where("user_id", userId).Table("reward").Order("id desc")
	if 0 < reason {
		instance = instance.Where("reason=?", reason)
	}

	instance = instance.Count(&count)

	if err := instance.Scopes(Paginate(b.PageNum, b.PageSize)).Find(&rewards).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, errors.NotFound("REWARD_NOT_FOUND", "reward not found"), 0
		}

		return nil, errors.New(500, "REWARD ERROR", err.Error()), 0
	}

	for _, reward := range rewards {
		res = append(res, &biz.Reward{
			ID:        reward.ID,
			UserId:    reward.UserId,
			Amount:    reward.Amount,
			Reason:    reward.Reason,
			CreatedAt: reward.CreatedAt,
			Address:   reward.Address,
			One:       reward.One,
			UpdatedAt: reward.UpdatedAt,
		})
	}

	return res, nil, count
}

// GetUserRecordByUserIdPage .
func (u *UserRepo) GetUserRecordByUserIdPage(ctx context.Context, b *biz.Pagination, userId uint64) ([]*biz.CardRecord, error, int64) {
	var (
		count   int64
		records []*CardRecord
	)

	res := make([]*biz.CardRecord, 0)

	instance := u.data.db.Where("user_id", userId).Table("card_record").Order("id desc")
	instance = instance.Count(&count)

	if err := instance.Scopes(Paginate(b.PageNum, b.PageSize)).Find(&records).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, errors.NotFound("REWARD_NOT_FOUND", "reward not found"), 0
		}

		return nil, errors.New(500, "REWARD ERROR", err.Error()), 0
	}

	for _, r := range records {
		res = append(res, &biz.CardRecord{
			ID:         r.ID,
			UserId:     r.UserId,
			RecordType: 0,
			Remark:     r.Remark,
			Code:       "",
			Opt:        "",
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  time.Time{},
		})
	}

	return res, nil, count
}
