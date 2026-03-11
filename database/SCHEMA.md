# SLG Game Database Schema
# MongoDB 数据模型定义

# ============================================================
# 1. PLAYERS - 玩家表
# ============================================================
# Collection: players
# Description: 存储玩家所有游戏数据

{
  _id: ObjectId,              # MongoDB 主键
  player_id: NumberLong,      # 玩家唯一 ID (从 10001 开始)
  username: String,           # 用户名 (3-32 字符，唯一)
  password_hash: String,      # 密码 SHA256 哈希 (64 字符)
  email: String,              # 邮箱地址
  
  # 玩家属性
  level: NumberInt,           # 等级 (默认：1)
  experience: NumberLong,     # 经验值 (默认：0)
  gold: NumberLong,           # 金币 (默认：1000, >=0)
  wood: NumberLong,           # 木材 (默认：1000, >=0)
  food: NumberLong,           # 粮食 (默认：1000, >=0)
  population: NumberInt,      # 当前人口 (默认：0)
  max_population: NumberInt,  # 最大人口 (默认：100)
  
  # 世界坐标
  x: NumberInt,               # X 坐标 (默认：0)
  y: NumberInt,               # Y 坐标 (默认：0)
  
  # 建筑列表 (嵌入文档)
  buildings: [
    {
      _id: ObjectId,
      type: String,           # 建筑类型：town_hall/barracks/farm/lumber_mill/mine/wall/archery_range/stable/academy/watch_tower
      level: NumberInt,       # 建筑等级 (0=未建造，1+=已建造)
      x: NumberInt,           # 建筑 X 坐标
      y: NumberInt,           # 建筑 Y 坐标
      build_time: Date,       # 建造开始时间
      finish_time: Date,      # 建造完成时间
      is_completed: Boolean   # 是否已完成建造
    }
  ],
  
  # 军队列表 (嵌入文档)
  troops: [
    {
      _id: ObjectId,
      type: String,           # 兵种类型：infantry/archer/cavalry/siege
      count: NumberLong,      # 士兵数量
      training_time: Date,    # 训练开始时间
      finish_time: Date,      # 训练完成时间
      is_completed: Boolean   # 是否已完成训练
    }
  ],
  
  # 科技研究 (键值对)
  research: {
    tech_type_1: NumberInt,   # 科技类型 -> 等级
    tech_type_2: NumberInt
  },
  
  # 联盟信息
  alliance_id: NumberLong,    # 联盟 ID (0=无联盟)
  alliance_role: String,      # 联盟角色：leader/officer/member
  
  # VIP 系统
  vip_level: NumberInt,       # VIP 等级 (默认：0)
  vip_expire: Date,           # VIP 过期时间
  
  # 游戏设置
  settings: {
    language: String,         # 语言：zh-CN/en-US (默认：zh-CN)
    notifications: Boolean,   # 通知开关 (默认：true)
    sound_enabled: Boolean,   # 音效开关 (默认：true)
    music_enabled: Boolean    # 音乐开关 (默认：true)
  },
  
  # 时间戳
  created_at: Date,           # 创建时间
  last_login: Date            # 最后登录时间
}

# 索引:
# - player_id (unique)
# - username (unique)
# - x, y (复合索引)
# - alliance_id
# - level (降序)

# ============================================================
# 2. ALLIANCES - 联盟表
# ============================================================
# Collection: alliances
# Description: 存储联盟信息

{
  _id: ObjectId,
  alliance_id: NumberLong,    # 联盟唯一 ID (从 1001 开始)
  name: String,               # 联盟名称 (唯一)
  description: String,        # 联盟描述
  creator_id: NumberLong,     # 创建者玩家 ID
  created_at: Date,           # 创建时间
  member_count: NumberInt,    # 当前成员数
  max_members: NumberInt,     # 最大成员数 (默认：50)
  level: NumberInt,           # 联盟等级 (默认：1)
  
  # 成员列表 (嵌入文档)
  members: [
    {
      player_id: NumberLong,  # 玩家 ID
      username: String,       # 玩家用户名
      role: String,           # 角色：leader/officer/member
      joined_at: Date         # 加入时间
    }
  ]
}

# 索引:
# - alliance_id (unique)
# - name (unique)
# - creator_id

# ============================================================
# 3. WORLD_TILES - 世界地图表
# ============================================================
# Collection: world_tiles
# Description: 存储世界地图每个地块的信息

{
  _id: ObjectId,
  coord: {
    x: NumberInt,             # X 坐标
    y: NumberInt              # Y 坐标
  },
  tile_type: String,          # 地形类型：grass/water/mountain/forest/desert
  owner_id: NumberLong,       # 所有者玩家 ID (0=无主)
  building_id: String,        # 建筑 ID (空=无建筑)
  resource: {
    gold: NumberInt,          # 金矿储量
    wood: NumberInt,          # 木材储量
    food: NumberInt,          # 粮食储量
    stone: NumberInt          # 石矿储量
  }
}

# 索引:
# - coord.x, coord.y (unique 复合索引)
# - owner_id
# - tile_type

# ============================================================
# 4. BATTLE_LOGS - 战斗记录表
# ============================================================
# Collection: battle_logs
# Description: 存储所有战斗记录

{
  _id: ObjectId,
  attacker_id: NumberLong,    # 攻击者玩家 ID
  defender_id: NumberLong,    # 防御者玩家 ID
  battle_time: Date,          # 战斗发生时间
  result: String,             # 战斗结果：attacker_win/defender_win/draw
  attacker_losses: NumberLong,# 攻击方损失兵力
  defender_losses: NumberLong,# 防御方损失兵力
  looted_resources: {
    gold: NumberLong,         # 掠夺金币
    wood: NumberLong,         # 掠夺木材
    food: NumberLong          # 掠夺粮食
  }
}

# 索引:
# - battle_time (降序)
# - attacker_id, battle_time (复合)
# - defender_id, battle_time (复合)

# ============================================================
# 5. RESEARCH_QUEUE - 研究队列
# ============================================================
# Collection: research_queue
# Description: 存储玩家正在进行的研究

{
  _id: ObjectId,
  player_id: NumberLong,      # 玩家 ID
  technology_type: String,    # 科技类型
  target_level: NumberInt,    # 目标等级
  start_time: Date,           # 开始研究时间
  end_time: Date              # 预计完成时间
}

# 索引:
# - player_id
# - end_time
# - player_id, end_time (复合)

# ============================================================
# 数据类型说明
# ============================================================
# NumberInt: 32 位整数 (最大 21 亿)
# NumberLong: 64 位整数 (最大 900 亿亿)
# String: 字符串
# Date: 日期时间
# Boolean: 布尔值 (true/false)
# ObjectId: MongoDB 自动生成 24 字符 ID
# Array: 数组/列表
# Object: 嵌套对象

# ============================================================
# 默认值说明
# ============================================================
# 新玩家创建时:
# - level: 1
# - experience: 0
# - gold/wood/food: 1000
# - population: 0
# - max_population: 100
# - x/y: 0
# - buildings: []
# - troops: []
# - research: {}
# - alliance_id: 0
# - alliance_role: ""
# - vip_level: 0
# - settings.language: "zh-CN"
