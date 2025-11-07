package jwtutil

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// 从环境变量初始化（便捷方式）
func InitFromEnv() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return errors.New("环境变量JWT_SECRET未设置")
	}

	expiryStr := os.Getenv("JWT_EXPIRY_HOURS")
	expiryHours := 24 // 默认24小时
	if expiryStr != "" {
		hours, err := strconv.Atoi(expiryStr)
		if err != nil {
			return errors.New("JWT_EXPIRY_HOURS格式错误")
		}
		expiryHours = hours
	}

	Init(secret, time.Duration(expiryHours)*time.Hour)
	return nil
}
