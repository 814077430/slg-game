package tech

import (
	"fmt"

	"slg-game/database"
)

// TechnologyType 定义科技类型
type TechnologyType string

const (
	TechResourceProduction TechnologyType = "resource_production"
	TechBuildingSpeed      TechnologyType = "building_speed"
	TechArmyTraining       TechnologyType = "army_training"
)

// Technology 科技
type Technology struct {
	Type         TechnologyType `json:"type"`
	CurrentLevel int32          `json:"current_level"`
	MaxLevel     int32          `json:"max_level"`
}

// TechnologyManager 科技管理器（简化版）
type TechnologyManager struct {
	db database.DB
}

// NewTechnologyManager 创建科技管理器
func NewTechnologyManager(db database.DB) *TechnologyManager {
	return &TechnologyManager{db: db}
}

// ResearchTechnology 研究科技
func (tm *TechnologyManager) ResearchTechnology(playerID uint64, techType TechnologyType) error {
	return fmt.Errorf("not implemented")
}

// GetTechnology 获取科技信息
func (tm *TechnologyManager) GetTechnology(playerID uint64, techType TechnologyType) (*Technology, error) {
	return &Technology{
		Type:         techType,
		CurrentLevel: 0,
		MaxLevel:     10,
	}, nil
}

// CompleteResearch 完成科技研究
func (tm *TechnologyManager) CompleteResearch(playerID uint64, techType TechnologyType, level int32) error {
	return fmt.Errorf("not implemented")
}
