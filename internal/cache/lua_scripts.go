package cache

import "github.com/redis/go-redis/v9"

// LuaScripts 管理所有 Redis Lua 脚本
type LuaScripts struct {
	// 原子更新 Redis 并检测状态变化
	UpdateAndDetectChange *redis.Script

	// 原子入队 + 去重
	EnqueueWithDedup *redis.Script

	// 批量出队
	DequeueBatch *redis.Script

	// 原子释放锁（验证 owner）
	ReleaseLock *redis.Script
}

// NewLuaScripts 创建 Lua 脚本管理器
func NewLuaScripts() *LuaScripts {
	return &LuaScripts{
		UpdateAndDetectChange: redis.NewScript(luaUpdateAndDetectChange),
		EnqueueWithDedup:      redis.NewScript(luaEnqueueWithDedup),
		DequeueBatch:          redis.NewScript(luaDequeueBatch),
		ReleaseLock:           redis.NewScript(luaReleaseLock),
	}
}

// luaUpdateAndDetectChange 原子更新 Redis 并检测状态变化
// KEYS[1]: agent:status:{agentID}
// ARGV[1]: new_status
// ARGV[2]: new_last_seen (Unix timestamp)
// ARGV[3]: TTL (seconds)
// 返回: 0=状态未变化, 1=状态变化, 2=首次注册
const luaUpdateAndDetectChange = `
local key = KEYS[1]
local new_status = ARGV[1]
local new_last_seen = ARGV[2]
local ttl = tonumber(ARGV[3])

-- 读取旧状态
local old_status = redis.call('HGET', key, 'status')

-- 写入新数据
redis.call('HMSET', key,
    'status', new_status,
    'last_seen', new_last_seen)

-- 设置过期时间
redis.call('EXPIRE', key, ttl)

-- 返回状态是否变化
if old_status == false then
    return 2  -- 首次注册
elseif old_status ~= new_status then
    return 1  -- 状态变化
else
    return 0  -- 状态未变化
end
`

// luaEnqueueWithDedup 原子入队 + 去重
// KEYS[1]: agent:batch:queue (List)
// KEYS[2]: agent:batch:pending (Set)
// ARGV[1]: agentID
// 返回: 当前队列长度，如果已存在返回 -1
const luaEnqueueWithDedup = `
local queue_key = KEYS[1]
local pending_key = KEYS[2]
local agent_id = ARGV[1]

-- 检查是否已在队列中
local exists = redis.call('SISMEMBER', pending_key, agent_id)

if exists == 1 then
    return -1  -- 已存在，跳过
end

-- 加入队列和去重集合
redis.call('LPUSH', queue_key, agent_id)
redis.call('SADD', pending_key, agent_id)

-- 返回当前队列长度
return redis.call('LLEN', queue_key)
`

// luaDequeueBatch 批量出队
// KEYS[1]: agent:batch:queue (List)
// KEYS[2]: agent:batch:pending (Set)
// ARGV[1]: batch_size
// 返回: agentID 数组
const luaDequeueBatch = `
local queue_key = KEYS[1]
local pending_key = KEYS[2]
local batch_size = tonumber(ARGV[1])

local agent_ids = {}

for i = 1, batch_size do
    local agent_id = redis.call('RPOP', queue_key)

    if agent_id == false then
        break  -- 队列为空
    end

    -- 从去重集合中移除
    redis.call('SREM', pending_key, agent_id)

    table.insert(agent_ids, agent_id)
end

return agent_ids
`

// luaReleaseLock 原子释放锁（验证 owner）
// KEYS[1]: agent:batch:lock
// ARGV[1]: instanceID (owner)
// 返回: 1=成功释放, 0=不是锁的持有者
const luaReleaseLock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`
