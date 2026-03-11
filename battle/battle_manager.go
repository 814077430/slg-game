package battle

import (
	"log"
	"sync"
	"time"

	"slg-game/database"
)

// BattleManager 战斗管理器（独立线程）
type BattleManager struct {
	db           database.DB
	tickInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	currentTick  uint64
	mutex        sync.RWMutex
}

// BattleQueueItem 战斗队列项
type BattleQueueItem struct {
	AttackerID  uint64
	DefenderID  uint64
	AttackerTroops []interface{}
	ScheduledTime time.Time
	Status      string // pending/processing/completed
}

// NewBattleManager 创建战斗管理器
func NewBattleManager(db database.DB) *BattleManager {
	return &BattleManager{
		db:           db,
		tickInterval: 1000 * time.Millisecond, // 1 秒
		stopChan:     make(chan struct{}),
	}
}

// StartLoop 启动战斗独立循环
func (bm *BattleManager) StartLoop() {
	bm.wg.Add(1)
	go func() {
		defer bm.wg.Done()
		log.Printf("[Battle] Battle loop started with tick interval: %v", bm.tickInterval)
		
		ticker := time.NewTicker(bm.tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				bm.tick()
			case <-bm.stopChan:
				log.Println("[Battle] Battle loop stopping...")
				return
			}
		}
	}()
}

// StopLoop 停止战斗循环
func (bm *BattleManager) StopLoop() {
	close(bm.stopChan)
	bm.wg.Wait()
	log.Println("[Battle] Battle loop stopped")
}

// Stop 停止战斗管理器（别名）
func (bm *BattleManager) Stop() {
	bm.StopLoop()
}

// tick 执行一个战斗 tick
func (bm *BattleManager) tick() {
	bm.mutex.Lock()
	bm.currentTick++
	tick := bm.currentTick
	bm.mutex.Unlock()

	// 每 10 个 tick 记录一次状态
	if tick%10 == 0 {
		log.Printf("[Battle] Tick: %d", tick)
	}

	// 处理行军队列
	bm.processMarchQueue()

	// 处理战斗结果
	bm.processBattleResults()

	// 清理过期数据
	bm.processCleanup()
}

// processMarchQueue 处理行军队列
func (bm *BattleManager) processMarchQueue() {
	// TODO: 检查行军队列，处理到达的军队
	// collection := bm.db.GetCollection("march_queue")
	// 查询已到达的行军
	// 触发战斗或返回
}

// processBattleResults 处理战斗结果
func (bm *BattleManager) processBattleResults() {
	// TODO: 处理战斗结果，更新资源、兵力等
}

// processCleanup 清理过期数据
func (bm *BattleManager) processCleanup() {
	// TODO: 清理过期的战斗记录
}

// RequestBattle 请求战斗（添加到战斗队列）
func (bm *BattleManager) RequestBattle(attackerID, defenderID uint64, attackerTroops []interface{}) error {
	collection := bm.db.GetCollection("battle_queue")
	
	battle := map[string]interface{}{
		"attacker_id":    attackerID,
		"defender_id":    defenderID,
		"attacker_troops": attackerTroops,
		"scheduled_time": time.Now().Add(5 * time.Second), // 5 秒后战斗
		"status":         "pending",
		"created_at":     time.Now(),
	}
	
	return collection.InsertOne(battle)
}

// GetBattleQueue 获取战斗队列
func (bm *BattleManager) GetBattleQueue() ([]map[string]interface{}, error) {
	collection := bm.db.GetCollection("battle_queue")
	// 查询待处理的战斗
	return collection.GetAll(), nil
}

// GetTick 获取当前 tick 数
func (bm *BattleManager) GetTick() uint64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	return bm.currentTick
}
