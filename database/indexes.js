// SLG Game Database Indexes
// MongoDB 索引创建脚本
// 使用方法：mongosh mongodb://localhost:27017/slg_game < indexes.js

print("Creating indexes for SLG Game Database...\n");

// ============================================================
// Players Collection
// ============================================================
print("📁 Creating indexes for 'players' collection...");

db.players.createIndex({ "player_id": 1 }, { unique: true });
db.players.createIndex({ "username": 1 }, { unique: true });
db.players.createIndex({ "x": 1, "y": 1 });
db.players.createIndex({ "alliance_id": 1 });
db.players.createIndex({ "level": -1 });
db.players.createIndex({ "alliance_id": 1, "level": -1 });
db.players.createIndex({ "last_login": -1 });

print("  ✓ Players indexes created\n");

// ============================================================
// Alliances Collection
// ============================================================
print("📁 Creating indexes for 'alliances' collection...");

db.alliances.createIndex({ "alliance_id": 1 }, { unique: true });
db.alliances.createIndex({ "name": 1 }, { unique: true });
db.alliances.createIndex({ "creator_id": 1 });

print("  ✓ Alliances indexes created\n");

// ============================================================
// World Tiles Collection
// ============================================================
print("📁 Creating indexes for 'world_tiles' collection...");

db.world_tiles.createIndex({ "coord.x": 1, "coord.y": 1 }, { unique: true });
db.world_tiles.createIndex({ "owner_id": 1 });
db.world_tiles.createIndex({ "tile_type": 1 });

print("  ✓ World tiles indexes created\n");

// ============================================================
// Battle Logs Collection
// ============================================================
print("📁 Creating indexes for 'battle_logs' collection...");

db.battle_logs.createIndex({ "battle_time": -1 });
db.battle_logs.createIndex({ "attacker_id": 1, "battle_time": -1 });
db.battle_logs.createIndex({ "defender_id": 1, "battle_time": -1 });

print("  ✓ Battle logs indexes created\n");

// ============================================================
// Research Queue Collection
// ============================================================
print("📁 Creating indexes for 'research_queue' collection...");

db.research_queue.createIndex({ "player_id": 1 });
db.research_queue.createIndex({ "end_time": 1 });
db.research_queue.createIndex({ "player_id": 1, "end_time": 1 });

print("  ✓ Research queue indexes created\n");

// ============================================================
// Summary
// ============================================================
print("═══════════════════════════════════════════════════════");
print("All indexes created successfully!");
print("═══════════════════════════════════════════════════════\n");

// Show index summary
print("Index Summary:");
print("  - players: " + db.players.getIndexes().length + " indexes");
print("  - alliances: " + db.alliances.getIndexes().length + " indexes");
print("  - world_tiles: " + db.world_tiles.getIndexes().length + " indexes");
print("  - battle_logs: " + db.battle_logs.getIndexes().length + " indexes");
print("  - research_queue: " + db.research_queue.getIndexes().length + " indexes");
