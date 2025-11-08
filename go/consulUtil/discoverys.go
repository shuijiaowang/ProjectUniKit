package consulUtil

import (
	"errors"
	"log/slog"
	"strconv"
	"sync"

	"github.com/hashicorp/consul/api"
)

//我现在需要一个map集合的内存状态，主键为ServiceName，value=ServiceResult
//然后在初始化中调用把数据加载到map集合中，我在程序组就不调用这个方法了，直接从数组中取值，ok？

// ServiceResult 服务选择结果结构体
type ServiceResult struct {
	ServiceName  string            //服务名
	ServiceEntry *api.ServiceEntry // 选中的服务实例
	BaseURL      string            // 服务基础URL
	Success      bool              // 是否成功选中服务
}

// LoadBalanceStrategy 负载均衡策略类型
type LoadBalanceStrategy string

// 定义支持的负载均衡策略
const (
	RandomStrategy     LoadBalanceStrategy = "random"      // 随机策略
	RoundRobinStrategy LoadBalanceStrategy = "round_robin" // 轮询策略
)

// 全局服务状态map（键：ServiceName，值：ServiceResult）
var (
	serviceStatusMap = make(map[string]ServiceResult)
	statusMapMu      sync.RWMutex // 保证map并发安全
)

// SelectServiceFromNames 从多个服务名中选择服务实例
// serviceNames: 服务名切片
// strategy: 负载均衡策略
// useCache: 是否使用缓存
func SelectServiceFromNames(serviceNames []string, strategy LoadBalanceStrategy, useCache bool) error {
	// 检查Consul客户端是否初始化
	if ConsulClient == nil {
		return errors.New("consul client not initialized")
	}

	// 检查服务名切片是否为空
	if len(serviceNames) == 0 {
		return errors.New("service names slice cannot be empty")
	}

	// 使用写锁，因为我们要修改map
	statusMapMu.Lock()
	defer statusMapMu.Unlock()

	// 为每个服务名单独选择实例
	for _, serviceName := range serviceNames {
		// 初始化当前服务的结果
		result := ServiceResult{
			ServiceName: serviceName,
			Success:     false,
		}

		// 获取当前服务的健康实例
		services, err := GetHealthyServices(serviceName, useCache)
		if err != nil {
			slog.Warn("获取健康服务失败", "service", serviceName, "err", err)
			serviceStatusMap[serviceName] = result // 将失败结果存入map
			continue
		}

		// 检查是否有可用实例
		if len(services) == 0 {
			slog.Warn("无健康服务实例", "service", serviceName)
			serviceStatusMap[serviceName] = result // 将失败结果存入map
			continue
		}

		// 根据负载策略选择实例
		var selected *api.ServiceEntry
		switch strategy {
		case RandomStrategy:
			selected, err = SelectRandom(services)
		case RoundRobinStrategy:
			selected = SelectRoundRobin(services)
		default:
			slog.Warn("不支持的负载策略", "strategy", strategy, "service", serviceName)
			serviceStatusMap[serviceName] = result // 将失败结果存入map
			continue
		}

		if err != nil || selected == nil {
			slog.Warn("选择服务实例失败", "service", serviceName, "err", err)
			serviceStatusMap[serviceName] = result // 将失败结果存入map
			continue
		}

		// 构建基础URL并更新结果
		baseURL := "http://" + selected.Service.Address + ":" + strconv.Itoa(selected.Service.Port) //这里不改成https？
		result.ServiceEntry = selected
		result.BaseURL = baseURL
		result.Success = true

		// 将最终结果存入全局map
		serviceStatusMap[serviceName] = result
	}
	return nil
}

// GetServiceResult 从全局map中获取指定服务的结果
// 这是一个辅助函数，供程序其他部分安全地读取map
func GetServiceResult(serviceName string) (ServiceResult, bool) {
	statusMapMu.RLock()
	defer statusMapMu.RUnlock()
	result, exists := serviceStatusMap[serviceName]
	return result, exists
}

// GetAllServiceResults 获取map中的所有服务结果
func GetAllServiceResults() map[string]ServiceResult {
	statusMapMu.RLock()
	defer statusMapMu.RUnlock()
	// 返回一个拷贝，防止外部直接修改内部map
	copiedMap := make(map[string]ServiceResult, len(serviceStatusMap))
	for k, v := range serviceStatusMap {
		copiedMap[k] = v
	}
	return copiedMap
}
