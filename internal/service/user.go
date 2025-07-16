package service

import (
	pb "cardbinance/api/user/v1"
	"cardbinance/internal/biz"
	"cardbinance/internal/conf"
	"cardbinance/internal/pkg/middleware/auth"
	"context"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-kratos/kratos/v2/log"
	jwt2 "github.com/golang-jwt/jwt/v5"
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

	//res, err = addressCheck(userAddress)
	//if nil != err {
	//	return &v1.EthAuthorizeReply{
	//		Token:  "",
	//		Status: "地址验证失败",
	//	}, nil
	//}
	//if !res {
	//	return &v1.EthAuthorizeReply{
	//		Token:  "",
	//		Status: "地址格式错误",
	//	}, nil
	//}

	var (
		addressFromSign string
		content         = []byte(userAddress) // todo 签名内容修改
	)

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
	if err != nil {
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

func (u *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	var address string

	return u.uuc.GetUserByAddress(ctx, address)
}

//func addressCheck(addressParam string) (bool, error) {
//	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
//	if !re.MatchString(addressParam) {
//		return false, nil
//	}
//
//	client, err := ethclient.Dial("https://bsc-dataseed4.binance.org/")
//	if err != nil {
//		return false, err
//	}
//
//	// a random user account address
//	address := common.HexToAddress(addressParam)
//	bytecode, err := client.CodeAt(context.Background(), address, nil) // nil is latest block
//	if err != nil {
//		return false, err
//	}
//
//	if len(bytecode) > 0 {
//		return false, nil
//	}
//
//	return true, nil
//}

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
