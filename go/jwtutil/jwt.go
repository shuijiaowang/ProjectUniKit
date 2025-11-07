package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret     []byte        // JWT签名密钥
	defaultExpiry time.Duration // 默认过期时间
)

func Init(secret string, expiry time.Duration) {
	jwtSecret = []byte(secret)
	defaultExpiry = expiry
}

// UserTokenInfo 用于生成JWT的用户核心信息
type UserClaimsInfo struct {
	ID       int    // 用户ID
	Username string // 用户名
	UUID     string // 用户UUID
}

// CustomClaims 自定义JWT载荷
type UserClaims struct {
	UserClaimsInfo // 嵌入用户信息
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT令牌
// 允许自定义过期时间（传0则使用默认值）
// GenerateToken 生成JWT令牌
// info：用户核心信息
// expiry：可选过期时间（传0则使用默认值）
func GenerateToken(info UserClaimsInfo, expiry ...time.Duration) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt未初始化，请先调用Init方法")
	}

	// 处理过期时间
	expirationTime := time.Now()
	if len(expiry) > 0 && expiry[0] > 0 {
		expirationTime = expirationTime.Add(expiry[0])
	} else {
		expirationTime = expirationTime.Add(defaultExpiry)
	}

	claims := &UserClaims{
		UserClaimsInfo: info, // 直接嵌入用户信息
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "JWTUtil",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*UserClaims, error) {
	if len(jwtSecret) == 0 {
		return nil, errors.New("jwt未初始化，请先调用Init方法")
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
