package consulUtil

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

// 注册服务到Consul
func registerToConsul(r *gin.Engine, registration *api.AgentServiceRegistration) error {

	// 自动添加健康检查（如果未配置）
	if registration.Check == nil {
		// 默认HTTP健康检查（5秒间隔，1秒超时）
		registration.Check = &api.AgentServiceCheck{
			HTTP:                           "http://" + registration.Address + ":" + strconv.Itoa(registration.Port) + "/health", //默认，可初始化是传入https
			Interval:                       "5s",
			Timeout:                        "1s",
			DeregisterCriticalServiceAfter: "30s", // 健康检查失败30秒后自动注销
		}
	}
	// 注册健康检查接口（如果是HTTP类型检查）
	if registration.Check != nil && registration.Check.HTTP != "" {
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})
		slog.Info("注册健康检查: /health")
	}

	if err := ConsulClient.Agent().ServiceRegister(registration); err != nil {
		return errors.New("服务注册失败: " + err.Error())
	}
	slog.Info("服务注册成功", "[serviceName]", registration.Name)
	return nil
}
