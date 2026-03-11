# SLG Game Database Setup Scripts
# MongoDB 数据库初始化和维护脚本

# ============================================================
# 1. 创建数据库和索引
# ============================================================

# 连接到 MongoDB
# mongosh mongodb://localhost:27017/slg_game

# 创建数据库
use slg_game

# ============================================================
# 2. 创建集合和索引
# ============================================================

# --- players 玩家表 ---
db.createCollection("players")

# 唯一索引
db.players.createIndex({ "player_id": 1 }, { unique: true })
db.players.createIndex({ "username": 1 }, { unique: true })

# 查询索引
db.players.createIndex({ "x": 1, "y": 1 })
db.players.createIndex({ "alliance_id": 1 })
db.players.createIndex({ "level": -1 })

# 复合索引
db.players.createIndex({ "alliance_id": 1, "level": -1 })

# --- alliances 联盟表 ---
db.createCollection("alliances")

db.alliances.createIndex({ "alliance_id": 1 }, { unique: true })
db.alliances.createIndex({ "name": 1 }, { unique: true })
db.alliances.createIndex({ "creator_id": 1 })

# --- world_tiles 世界地图表 ---
db.createCollection("world_tiles")

db.world_tiles.createIndex({ "coord.x": 1, "coord.y": 1 }, { unique: true })
db.world_tiles.createIndex({ "owner_id": 1 })
db.world_tiles.createIndex({ "tile_type": 1 })

# --- battle_logs 战斗记录表 ---
db.createCollection("battle_logs")

db.battle_logs.createIndex({ "battle_time": -1 })
db.battle_logs.createIndex({ "attacker_id": 1, "battle_time": -1 })
db.battle_logs.createIndex({ "defender_id": 1, "battle_time": -1 })

# --- research_queue 研究队列 ---
db.createCollection("research_queue")

db.research_queue.createIndex({ "player_id": 1 })
db.research_queue.createIndex({ "end_time": 1 })
db.research_queue.createIndex({ "player_id": 1, "end_time": 1 })

# ============================================================
# 3. 插入初始数据（可选）
# ============================================================

# 插入初始世界地图数据（示例）
# db.world_tiles.insertMany([
#   {
#     coord: { x: 0, y: 0 },
#     tile_type: "grass",
#     owner_id: 0,
#     building_id: "",
#     resource: { gold: 5, wood: 5, food: 5, stone: 5 }
#   }
# ])

# ============================================================
# 4. 数据验证规则（MongoDB 4.0+）
# ============================================================

# 玩家数据验证
db.runCommand({
  collMod: "players",
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["player_id", "username", "password_hash"],
      properties: {
        player_id: { bsonType: "long", description: "Player ID is required" },
        username: { 
          bsonType: "string",
          minLength: 3,
          maxLength: 32,
          description: "Username must be 3-32 characters"
        },
        password_hash: { 
          bsonType: "string",
          minLength: 64,
          maxLength: 64,
          description: "Password hash must be SHA256"
        },
        gold: { bsonType: "long", minimum: 0 },
        wood: { bsonType: "long", minimum: 0 },
        food: { bsonType: "long", minimum: 0 }
      }
    }
  }
})

# ============================================================
# 5. 查看集合信息
# ============================================================

# 查看所有集合
db.getCollectionNames()

# 查看集合统计
db.players.stats()
db.alliances.stats()
db.world_tiles.stats()

# 查看索引
db.players.getIndexes()
db.alliances.getIndexes()
db.world_tiles.getIndexes()

# ============================================================
# 6. 数据备份和恢复
# ============================================================

# 备份数据库
# mongodump --uri="mongodb://localhost:27017" --db=slg_game --out=./backup

# 恢复数据库
# mongorestore --uri="mongodb://localhost:27017" --db=slg_game ./backup/slg_game

# 导出 JSON
# mongoexport --uri="mongodb://localhost:27017" --db=slg_game --collection=players --out=players.json

# 导入 JSON
# mongoimport --uri="mongodb://localhost:27017" --db=slg_game --collection=players --file=players.json

# ============================================================
# 7. 性能优化
# ============================================================

# 分析查询性能
# db.players.find({ player_id: 10001 }).explain("executionStats")

# 修复碎片
# db.players.reIndex()

# 压缩集合
# db.runCommand({ compact: "players" })

# ============================================================
# 8. 清理数据（谨慎使用）
# ============================================================

# 删除所有玩家数据
# db.players.deleteMany({})

# 删除所有联盟数据
# db.alliances.deleteMany({})

# 删除所有战斗记录（保留最近 1000 条）
# db.battle_logs.deleteMany({ 
#   battle_time: { $lt: db.battle_logs.find().sort({ battle_time: -1 }).limit(1000).toArray()[999].battle_time }
# })

# 删除过期研究队列
# db.research_queue.deleteMany({ end_time: { $lt: new Date() } })

# ============================================================
# 9. 监控查询
# ============================================================

# 查看慢查询
# db.setProfilingLevel(2, 100)  # 记录所有超过 100ms 的查询

# 查看当前操作
# db.currentOp()

# 查看数据库大小
# db.stats()

# 查看集合大小
# db.players.dataSize()
# db.players.storageSize()
