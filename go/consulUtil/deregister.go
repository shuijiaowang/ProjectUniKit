package consulUtil

import (
	"errors"
	"log/slog"
)

// DeregisterService 从Consul注销服务
// serviceID: 服务唯一ID（与注册时的ID一致）
func DeregisterService(serviceID string) error {
	if ConsulClient == nil {
		return errors.New("consul client not initialized")
	}
	if err := ConsulClient.Agent().ServiceDeregister(serviceID); err != nil {
		return errors.New("service deregister failed: " + err.Error())
	}
	slog.Info("服务注销", "[serviceID]", serviceID)
	return nil
}
