package jwtutil

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// JWTMiddleware 生成Gin中间件（验证token并自动刷新）
// 注意：依赖gin框架，若其他项目不使用gin可删除此文件
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"code": 401, "msg": "未提供token"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(401, gin.H{"code": 401, "msg": "token格式错误"})
			c.Abort()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			c.JSON(401, gin.H{"code": 401, "msg": "无效的token"})
			c.Abort()
			return
		}

		// 自动刷新token（剩余时间<30分钟时）
		if claims.ExpiresAt != nil && claims.ExpiresAt.Unix()-time.Now().Unix() < 30*60 {
			newToken, err := GenerateToken(claims.UserClaimsInfo)
			if err == nil {
				c.Header("new-token", newToken)
			}
		}

		c.Set("claims", claims)
		c.Next()
	}
}

// GetUserInfo 从Gin上下文获取用户信息（依赖gin）
func GetUserInfo(c *gin.Context) *UserClaims {
	claims, exists := c.Get("claims")
	if !exists {
		return nil
	}
	return claims.(*UserClaims)
}
