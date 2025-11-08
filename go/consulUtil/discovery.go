package consulUtil

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"log/slog"

	"github.com/hashicorp/consul/api"
)

// 服务缓存结构
type serviceCache struct {
	services []*api.ServiceEntry
	expireAt time.Time
	mu       sync.RWMutex
}

var (
	cache = make(map[string]*serviceCache)
	// 缓存过期时间（默认20秒，允许外部配置）
	cacheTTL = 20 * time.Second
	// 新增：保护cacheTTL的互斥锁（确保并发读写安全）
	cacheTTLMu sync.RWMutex

	//轮循
	roundRobinIndexes = make(map[string]int) // 轮询索引：按服务名隔离（key:服务名，value:该服务的当前索引）
	indexMu           sync.Mutex             // 保护轮询索引map的并发安全
	//随机
	randomGenerator *rand.Rand
	randomOnce      sync.Once
)

// GetHealthyServices 从缓存或Consul获取健康服务实例
// serviceName: 服务名称
// useCache: 是否使用缓存
func GetHealthyServices(serviceName string, useCache bool) ([]*api.ServiceEntry, error) {
	if ConsulClient == nil {
		return nil, errors.New("consul client not initialized")
	}

	// 尝试从缓存获取
	if useCache {
		cacheItem, ok := getCache(serviceName)
		if ok && !cacheItem.expireAt.Before(time.Now()) {
			slog.Info("get %s from cache, count: %d", serviceName, len(cacheItem.services))
			return cacheItem.services, nil
		}
	}

	// 缓存未命中或过期，从Consul查询
	services, _, err := ConsulClient.Health().Service(
		serviceName,
		"",
		true,
		&api.QueryOptions{},
	)
	if err != nil {
		return nil, errors.New("query healthy services failed: " + err.Error())
	}
	if len(services) == 0 {
		return nil, errors.New("no healthy service found for: " + serviceName)
	}

	// 更新缓存
	updateCache(serviceName, services)
	return services, nil
}

// 负载均衡策略：随机
func SelectRandom(services []*api.ServiceEntry) (*api.ServiceEntry, error) {
	if len(services) == 0 {
		return nil, errors.New("no services available to select")
	}
	// 确保随机数生成器只初始化一次
	randomOnce.Do(func() {
		randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	})
	index := randomGenerator.Intn(len(services))
	return services[index], nil
}

// 负载均衡策略：轮询（按服务名隔离索引）
func SelectRoundRobin(services []*api.ServiceEntry) *api.ServiceEntry {
	if len(services) == 0 {
		return nil
	}
	// 获取服务名（所有实例属于同一个服务，取第一个实例的服务名即可）
	serviceName := services[0].Service.Service

	// 加锁保证并发安全（避免多协程同时修改同一服务的索引）
	indexMu.Lock()
	defer indexMu.Unlock()

	// 获取当前服务的轮询索引（若未初始化则默认为0）
	currentIndex := roundRobinIndexes[serviceName]
	// 计算当前应选择的实例索引（取模确保在有效范围内）
	idx := currentIndex % len(services)
	// 更新索引（为下一次轮询做准备）
	roundRobinIndexes[serviceName] = currentIndex + 1

	return services[idx]
}

// 从缓存获取服务
func getCache(serviceName string) (*serviceCache, bool) {
	cacheItem, ok := cache[serviceName]
	if ok {
		cacheItem.mu.RLock()
		defer cacheItem.mu.RUnlock()
	}
	return cacheItem, ok
}

// 更新缓存时读取最新的cacheTTL（加读锁）
func updateCache(serviceName string, services []*api.ServiceEntry) {
	// 读取当前缓存过期时间
	cacheTTLMu.RLock()
	ttl := cacheTTL
	cacheTTLMu.RUnlock()

	cacheItem, ok := cache[serviceName]
	if !ok {
		cache[serviceName] = &serviceCache{
			services: services,
			expireAt: time.Now().Add(ttl), // 使用读取到的ttl
			mu:       sync.RWMutex{},
		}
		return
	}

	cacheItem.mu.Lock()
	defer cacheItem.mu.Unlock()
	cacheItem.services = services
	cacheItem.expireAt = time.Now().Add(ttl) // 使用读取到的ttl
}

// 新增：设置缓存过期时间的函数（允许外部配置）
// ttl: 缓存过期时间（必须大于0，否则返回错误）
func SetCacheTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("cache TTL must be greater than 0")
	}
	cacheTTLMu.Lock()
	defer cacheTTLMu.Unlock()
	cacheTTL = ttl
	slog.Info("cache TTL updated", "new_ttl", ttl.String())
	return nil
}
