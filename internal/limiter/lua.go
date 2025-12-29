package limiter

// Token Bucket Script
// keys: [tokens_key, timestamp_key]
// args: [rate, capacity, now, requested]
const resultTokenBucket = `
local tokens_key = KEYS[1]
local timestamp_key = KEYS[2]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local fill_time = capacity / rate
local ttl = math.floor(fill_time * 2)

local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
  last_tokens = capacity
end

local last_refreshed = tonumber(redis.call("get", timestamp_key))
if last_refreshed == nil then
  last_refreshed = 0
end

local delta = math.max(0, now - last_refreshed)
local filled_tokens = math.min(capacity, last_tokens + (delta * rate))
local allowed = filled_tokens >= requested
local new_tokens = filled_tokens
local allowed_num = 0

if allowed then
  new_tokens = filled_tokens - requested
  allowed_num = 1
end

redis.call("setex", tokens_key, ttl, new_tokens)
redis.call("setex", timestamp_key, ttl, now)

return { allowed_num, new_tokens }
`

// Sliding Window Script
// keys: [window_key]
// args: [window_size_ms, limit, now_ms]
const resultSlidingWindow = `
local key = KEYS[1]
local window_size = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove items outside the window
redis.call("ZREMRANGEBYSCORE", key, 0, now - window_size)

-- Check count
local count = redis.call("ZCARD", key)

if count < limit then
  -- Add new request
  redis.call("ZADD", key, now, now)
  -- Set expiry for cleanup (window size + buffer)
  redis.call("PEXPIRE", key, window_size)
  return 1
end

return 0
`
