// filePath: consulUtil/refresh.go
package consulUtil

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

var refreshCancel context.CancelFunc

// StartServiceStatusRefresh 启动定时任务自动刷新服务状态map
// serviceNames: 需要监控的服务名列表
// strategy: 负载均衡策略
// interval: 刷新间隔
// 返回用于停止定时任务的context.CancelFunc
func StartServiceStatusRefresh(serviceNames []string, strategy LoadBalanceStrategy, interval time.Duration) error {
	if len(serviceNames) == 0 {
		return errors.New("service names cannot be empty")
	}
	if interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	if ConsulClient == nil {
		return errors.New("consul client not initialized")
	}

	ctx, cancel := context.WithCancel(context.Background())
	refreshCancel = cancel
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		// 立即执行一次初始化刷新
		refresh(serviceNames, strategy)

		for {
			select {
			case <-ctx.Done():
				slog.Info("服务刷新停止")
				return
			case <-ticker.C:
				refresh(serviceNames, strategy)
			}
		}
	}()

	return nil
}

// RefreshServiceStatus 手动强制刷新服务状态map（不使用缓存）
func RefreshServiceStatus(serviceNames []string, strategy LoadBalanceStrategy) error {
	return SelectServiceFromNames(serviceNames, strategy, false)
}

// 内部刷新函数
func refresh(serviceNames []string, strategy LoadBalanceStrategy) {
	slog.Info("开始刷新服务:", "services", serviceNames)
	if err := SelectServiceFromNames(serviceNames, strategy, false); err != nil {
		slog.Warn("刷新失败", "error", err)
	} else {
		slog.Info("刷新成功", "services", serviceNames)
	}
}
