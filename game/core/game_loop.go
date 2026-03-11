package core

import (
	"log"
	"sync"
	"time"

	"slg-game/database"
)

// GameLoop 游戏主循环（不包含 World）
type GameLoop struct {
	db            database.DB
	tickInterval  time.Duration
	tickCount     uint64
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewGameLoop 创建游戏主循环
func NewGameLoop(db database.DB, tickInterval time.Duration) *GameLoop {
	return &GameLoop{
		db:           db,
		tickInterval: tickInterval,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动游戏主循环
func (gl *GameLoop) Start() {
	gl.wg.Add(1)
	go func() {
		defer gl.wg.Done()
		ticker := time.NewTicker(gl.tickInterval)
		defer ticker.Stop()

		log.Printf("[GameLoop] Game loop started with tick interval: %v", gl.tickInterval)

		for {
			select {
			case <-ticker.C:
				gl.tick()
			case <-gl.stopChan:
				log.Println("[GameLoop] Game loop stopping...")
				return
			}
		}
	}()
}

// Stop 停止游戏主循环
func (gl *GameLoop) Stop() {
	close(gl.stopChan)
	gl.wg.Wait()
	log.Println("[GameLoop] Game loop stopped")
}

// tick 执行一个游戏 tick
func (gl *GameLoop) tick() {
	gl.tickCount++

	// 每 10 个 tick 记录一次状态
	if gl.tickCount%10 == 0 {
		log.Printf("[GameLoop] Tick: %d", gl.tickCount)
	}

	// 处理建筑建造完成
	gl.processBuildingCompletion()

	// 处理科技研究完成
	gl.processTechnologyCompletion()

	// 处理军队移动
	gl.processArmyMovement()

	// 注意：World 模块现在有自己独立的循环，不在这里调用
}

// processBuildingCompletion 处理建筑建造完成
func (gl *GameLoop) processBuildingCompletion() {
	// TODO: 检查建造队列，完成到期的建筑
}

// processTechnologyCompletion 处理科技研究完成
func (gl *GameLoop) processTechnologyCompletion() {
	// TODO: 检查研究队列，完成到期的科技
}

// processArmyMovement 处理军队移动
func (gl *GameLoop) processArmyMovement() {
	// TODO: 处理军队移动和战斗
}
