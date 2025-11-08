package consulUtil

import (
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

// ConsulClient Consul客户端实例
var ConsulClient *api.Client

func InitConsul(consulAddr string, r *gin.Engine, registration *api.AgentServiceRegistration) error {

	// 验证必要参数
	if consulAddr == "" {
		return errors.New("consul address cannot be empty")
	}
	if registration == nil {
		return errors.New("service registration cannot be nil")
	}
	if registration.Name == "" || registration.Address == "" || registration.Port == 0 {
		return errors.New("registration missing required fields (Name/Address/Port)")
	}

	config := api.DefaultConfig()
	config.Address = consulAddr // Consul地址（如"localhost:8500"）
	client, err := api.NewClient(config)
	if err != nil {
		return err
	}
	ConsulClient = client
	// 注册服务（返回错误而非直接Fatal）
	if err := registerToConsul(r, registration); err != nil {
		return err
	}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		if err := DeregisterService(registration.ID); err != nil {
			slog.Error("deregister service failed", "err", err)
		}
		refreshCancel()
		slog.Info("定时刷新服务停止")
		os.Exit(0)
	}()
	return nil
}
