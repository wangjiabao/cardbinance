package service

import (
	pb "cardbinance/api/user/v1"
	"cardbinance/internal/biz"
	"cardbinance/internal/conf"
	"cardbinance/internal/pkg/middleware/auth"
	"context"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"regexp"
	"strings"
	"time"
)

type UserService struct {
	pb.UnimplementedUserServer

	uuc *biz.UserUseCase
	log *log.Helper
	ca  *conf.Auth
}

func NewUserService(uuc *biz.UserUseCase, logger log.Logger, ca *conf.Auth) *UserService {
	return &UserService{uuc: uuc, log: log.NewHelper(logger), ca: ca}
}

func (u *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	// 在上下文 context 中取出 claims 对象
	var userId uint64
	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.GetUserReply{
				Status:        "获取用户信息错误",
				Address:       "",
				Amount:        "",
				MyTotalAmount: 0,
				Vip:           0,
				CardNum:       "",
				CardAmount:    "",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	return u.uuc.GetUserById(userId)
}

func (u *UserService) UserRecommend(ctx context.Context, req *pb.RecommendListRequest) (*pb.RecommendListReply, error) {
	return u.uuc.GetUserRecommend(ctx, req)
}

func (u *UserService) OrderList(ctx context.Context, req *pb.OrderListRequest) (*pb.OrderListReply, error) {
	// 在上下文 context 中取出 claims 对象
	var userId uint64
	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.OrderListReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	return u.uuc.OrderList(ctx, req, userId)
}

func (u *UserService) RewardList(ctx context.Context, req *pb.RewardListRequest) (*pb.RewardListReply, error) {
	// 在上下文 context 中取出 claims 对象
	var userId uint64
	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.RewardListReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	return u.uuc.RewardList(ctx, req, userId)
}

// CreateNonce createNonce.
func (u *UserService) CreateNonce(ctx context.Context, req *pb.CreateNonceRequest) (*pb.CreateNonceReply, error) {
	userAddress := req.SendBody.Address // 以太坊账户

	if "" == userAddress || 20 > len(userAddress) ||
		strings.EqualFold("0x000000000000000000000000000000000000dead", userAddress) {
		return &pb.CreateNonceReply{
			Nonce:  "",
			Status: "账户地址参数错误",
		}, nil
	}

	// 验证
	var (
		res bool
		err error
	)

	res, err = addressCheck(userAddress)
	if nil != err {
		return &pb.CreateNonceReply{
			Nonce:  "",
			Status: "地址验证失败",
		}, nil
	}
	if !res {
		return &pb.CreateNonceReply{
			Nonce:  "",
			Status: "地址验证失败",
		}, nil
	}

	return u.uuc.CreateNonce(ctx, req)
}

// EthAuthorize ethAuthorize.
func (u *UserService) EthAuthorize(ctx context.Context, req *pb.EthAuthorizeRequest) (*pb.EthAuthorizeReply, error) {
	userAddress := req.SendBody.Address // 以太坊账户

	if "" == userAddress || 20 > len(userAddress) ||
		strings.EqualFold("0x000000000000000000000000000000000000dead", userAddress) {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "账户地址参数错误",
		}, nil
	}

	if 10 >= len(req.SendBody.Sign) {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "签名错误",
		}, nil
	}

	// 验证
	var (
		res  bool
		err  error
		user *biz.User
		msg  string
	)

	res, err = addressCheck(userAddress)
	if nil != err {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "地址验证失败",
		}, nil
	}
	if !res {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "地址格式错误",
		}, nil
	}

	var (
		addressFromSign string
		content         = []byte(userAddress) // todo 签名内容修改
	)

	// todo
	//var (
	//	contentStr string
	//)
	//contentStr, err = u.uuc.GetAddressNonce(ctx, userAddress)
	//if nil != err {
	//	return &pb.EthAuthorizeReply{
	//		Token:  "",
	//		Status: "错误",
	//	}, nil
	//}
	//content = []byte(contentStr)

	res, addressFromSign = verifySig(req.SendBody.Sign, content)
	if !res || addressFromSign != userAddress {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "地址签名错误",
		}, nil
	}

	// 根据地址查询用户，不存在时则创建
	user, err, msg = u.uuc.GetExistUserByAddressOrCreate(ctx, &biz.User{
		Address: userAddress,
	}, req)
	if nil == user || nil != err {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: msg,
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.EthAuthorizeReply{
			Token:  "",
			Status: "用户已禁用",
		}, nil
	}

	claims := auth.CustomClaims{
		UserId:   user.ID,
		UserType: "user",
		RegisteredClaims: jwt2.RegisteredClaims{
			NotBefore: jwt2.NewNumericDate(time.Now()),                      // 签名的生效时间
			ExpiresAt: jwt2.NewNumericDate(time.Now().Add(100 * time.Hour)), // 2天过期
			Issuer:    "user",
		},
	}
	token, err := auth.CreateToken(claims, u.ca.JwtKey)
	if err != nil {
		return &pb.EthAuthorizeReply{
			Token:  token,
			Status: "生成token失败",
		}, nil
	}

	userInfoRsp := pb.EthAuthorizeReply{
		Token:  token,
		Status: "ok",
	}

	return &userInfoRsp, nil
}

func (u *UserService) SetVip(ctx context.Context, req *pb.SetVipRequest) (*pb.SetVipReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.SetVipReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.SetVipReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.SetVipReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.SetVipReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.SetVipReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.SetVip(ctx, req, userId)
}

func (u *UserService) OpenCard(ctx context.Context, req *pb.OpenCardRequest) (*pb.OpenCardReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.OpenCardReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.OpenCardReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.OpenCardReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.OpenCardReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.OpenCardReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.OpenCard(ctx, req, userId)
}

func (u *UserService) AmountToCard(ctx context.Context, req *pb.AmountToCardRequest) (*pb.AmountToCardReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.AmountToCardReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.AmountToCardReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.AmountToCardReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.AmountToCardReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.AmountToCardReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.AmountToCard(ctx, req, userId)
}

func (u *UserService) AmountTo(ctx context.Context, req *pb.AmountToRequest) (*pb.AmountToReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.AmountToReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.AmountToReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.AmountToReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.AmountToReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.AmountToReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.AmountTo(ctx, req, userId)
}

func (u *UserService) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.WithdrawReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.WithdrawReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.WithdrawReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.WithdrawReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.WithdrawReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.WithdrawReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.Withdraw(ctx, req, userId)
}

func (u *UserService) LookCard(ctx context.Context, req *pb.LookCardRequest) (*pb.LookCardReply, error) {
	// 在上下文 context 中取出 claims 对象
	var (
		err    error
		userId uint64
	)

	if claims, ok := jwt.FromContext(ctx); ok {
		c := claims.(jwt2.MapClaims)
		if c["UserId"] == nil {
			return &pb.LookCardReply{
				Status: "无效TOKEN",
			}, nil
		}

		userId = uint64(c["UserId"].(float64))
	}

	var (
		user *biz.User
	)
	user, err = u.uuc.GetUserDataById(userId)
	if nil != err {
		return &pb.LookCardReply{
			Status: "无效TOKEN",
		}, nil
	}

	if 1 == user.IsDelete {
		return &pb.LookCardReply{
			Status: "用户已删除",
		}, nil
	}

	var (
		res             bool
		addressFromSign string
	)
	if 10 >= len(req.SendBody.Sign) {
		return &pb.LookCardReply{
			Status: "签名错误",
		}, nil
	}
	res, addressFromSign = verifySig(req.SendBody.Sign, []byte(user.Address))
	if !res || addressFromSign != user.Address {
		return &pb.LookCardReply{
			Status: "签名错误",
		}, nil
	}

	return u.uuc.LookCard(ctx, req, userId)
}

func addressCheck(addressParam string) (bool, error) {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	if !re.MatchString(addressParam) {
		return false, nil
	}

	client, err := ethclient.Dial("https://bsc-dataseed4.binance.org/")
	if err != nil {
		return false, err
	}

	// a random user account address
	address := common.HexToAddress(addressParam)
	bytecode, err := client.CodeAt(context.Background(), address, nil) // nil is latest block
	if err != nil {
		return false, err
	}

	if len(bytecode) > 0 {
		return false, nil
	}

	return true, nil
}

func verifySig(sigHex string, msg []byte) (bool, string) {
	sig := hexutil.MustDecode(sigHex)

	msg = accounts.TextHash(msg)
	if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
		sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1
	}

	recovered, err := crypto.SigToPub(msg, sig)
	if err != nil {
		return false, ""
	}

	recoveredAddr := crypto.PubkeyToAddress(*recovered)

	sigPublicKeyBytes := crypto.FromECDSAPub(recovered)
	signatureNoRecoverID := sig[:len(sig)-1] // remove recovery id
	verified := crypto.VerifySignature(sigPublicKeyBytes, msg, signatureNoRecoverID)
	return verified, recoveredAddr.Hex()
}
