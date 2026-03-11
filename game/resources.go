package game

import (
	"sync"
	"time"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceGold   ResourceType = "gold"
	ResourceWood   ResourceType = "wood"
	ResourceFood   ResourceType = "food"
	ResourceStone  ResourceType = "stone"
	ResourceIron   ResourceType = "iron"
	ResourceMana   ResourceType = "mana"
)

// Resource 资源结构
type Resource struct {
	Type        ResourceType `json:"type" bson:"type"`
	Amount      int64        `json:"amount" bson:"amount"`
	Capacity    int64        `json:"capacity" bson:"capacity"`
	Production  int64        `json:"production" bson:"production"` // 每秒产量
	LastUpdated time.Time    `json:"last_updated" bson:"last_updated"`
}

// ResourcesManager 资源管理器
type ResourcesManager struct {
	resources map[ResourceType]*Resource
	mutex     sync.RWMutex
}

// NewResourcesManager 创建新的资源管理器
func NewResourcesManager() *ResourcesManager {
	rm := &ResourcesManager{
		resources: make(map[ResourceType]*Resource),
	}
	
	// 初始化默认资源
	rm.resources[ResourceGold] = &Resource{
		Type:        ResourceGold,
		Amount:      1000,
		Capacity:    10000,
		Production:  10,
		LastUpdated: time.Now(),
	}
	rm.resources[ResourceWood] = &Resource{
		Type:        ResourceWood,
		Amount:      1000,
		Capacity:    10000,
		Production:  8,
		LastUpdated: time.Now(),
	}
	rm.resources[ResourceFood] = &Resource{
		Type:        ResourceFood,
		Amount:      1000,
		Capacity:    10000,
		Production:  12,
		LastUpdated: time.Now(),
	}
	rm.resources[ResourceStone] = &Resource{
		Type:        ResourceStone,
		Amount:      500,
		Capacity:    5000,
		Production:  5,
		LastUpdated: time.Now(),
	}
	rm.resources[ResourceIron] = &Resource{
		Type:        ResourceIron,
		Amount:      300,
		Capacity:    3000,
		Production:  3,
		LastUpdated: time.Now(),
	}
	rm.resources[ResourceMana] = &Resource{
		Type:        ResourceMana,
		Amount:      200,
		Capacity:    2000,
		Production:  2,
		LastUpdated: time.Now(),
	}
	
	return rm
}

// GetResource 获取指定类型的资源
func (rm *ResourcesManager) GetResource(resourceType ResourceType) *Resource {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if resource, exists := rm.resources[resourceType]; exists {
		return resource
	}
	return nil
}

// GetAmount 获取资源数量
func (rm *ResourcesManager) GetAmount(resourceType ResourceType) int64 {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if resource, exists := rm.resources[resourceType]; exists {
		// 计算当前实际数量（考虑生产）
		now := time.Now()
		elapsedSeconds := now.Sub(resource.LastUpdated).Seconds()
		actualAmount := resource.Amount + int64(elapsedSeconds*float64(resource.Production))
		
		// 不超过容量
		if actualAmount > resource.Capacity {
			actualAmount = resource.Capacity
		}
		
		return actualAmount
	}
	return 0
}

// AddResource 增加资源
func (rm *ResourcesManager) AddResource(resourceType ResourceType, amount int64) bool {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	if resource, exists := rm.resources[resourceType]; exists {
		// 先更新到最新状态
		now := time.Now()
		elapsedSeconds := now.Sub(resource.LastUpdated).Seconds()
		resource.Amount += int64(elapsedSeconds * float64(resource.Production))
		if resource.Amount > resource.Capacity {
			resource.Amount = resource.Capacity
		}
		resource.LastUpdated = now
		
		// 增加资源
		newAmount := resource.Amount + amount
		if newAmount < 0 {
			return false // 不足
		}
		if newAmount > resource.Capacity {
			newAmount = resource.Capacity
		}
		resource.Amount = newAmount
		return true
	}
	return false
}

// RemoveResource 减少资源
func (rm *ResourcesManager) RemoveResource(resourceType ResourceType, amount int64) bool {
	return rm.AddResource(resourceType, -amount)
}

// CanAfford 检查是否能负担指定的资源消耗
func (rm *ResourcesManager) CanAfford(costs map[ResourceType]int64) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	for resourceType, cost := range costs {
		if resource, exists := rm.resources[resourceType]; exists {
			// 计算当前实际数量
			now := time.Now()
			elapsedSeconds := now.Sub(resource.LastUpdated).Seconds()
			actualAmount := resource.Amount + int64(elapsedSeconds*float64(resource.Production))
			if actualAmount > resource.Capacity {
				actualAmount = resource.Capacity
			}
			
			if actualAmount < cost {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// UpdateProduction 更新资源生产率
func (rm *ResourcesManager) UpdateProduction(resourceType ResourceType, production int64) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	if resource, exists := rm.resources[resourceType]; exists {
		// 先更新到最新状态
		now := time.Now()
		elapsedSeconds := now.Sub(resource.LastUpdated).Seconds()
		resource.Amount += int64(elapsedSeconds * float64(resource.Production))
		if resource.Amount > resource.Capacity {
			resource.Amount = resource.Capacity
		}
		resource.LastUpdated = now
		
		// 更新生产率
		resource.Production = production
	}
}

// GetAllResources 获取所有资源的当前状态
func (rm *ResourcesManager) GetAllResources() map[string]int32 {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	resources := make(map[string]int32)
	now := time.Now()
	
	for resourceType, resource := range rm.resources {
		elapsedSeconds := now.Sub(resource.LastUpdated).Seconds()
		actualAmount := resource.Amount + int64(elapsedSeconds*float64(resource.Production))
		if actualAmount > resource.Capacity {
			actualAmount = resource.Capacity
		}
		resources[string(resourceType)] = int32(actualAmount)
	}
	
	return resources
}