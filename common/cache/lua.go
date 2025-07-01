package cache

import (
	"context"
	"fmt"
)

const (
	LuaCompareAndIncrClientID = "LuaCompareAndIncrClientID"

	LuaCleanupConnection = "LuaCleanupConnection"
)

type luaPart struct {
	LuaScript string
	Sha       string
}

var luaScriptTable map[string]*luaPart = map[string]*luaPart{
	LuaCompareAndIncrClientID: {
		LuaScript: "if redis.call('exists', KEYS[1]) == 0 then redis.call('set', KEYS[1], 0) end;if redis.call('get', KEYS[1]) == ARGV[1] then redis.call('incr', KEYS[1]);redis.call('expire', KEYS[1], ARGV[2]); return 1 else return -1 end",
	},
	LuaCleanupConnection: {
		// This script cleans up all distributed states for a connection atomically.
		// ARGV[1]: connID
		// ARGV[2]: deviceID
		// ARGV[3]: slot_size
		LuaScript: `
            -- Get arguments
            local conn_id_str = ARGV[1]
            local conn_id_num = tonumber(conn_id_str)
            local device_id = ARGV[2]
            local slot_size = tonumber(ARGV[3])

            -- TODO: For multi-gateway deployment, a unique gateway_id is required as an argument.
            -- All keys below should be prefixed with this gateway_id to prevent connID collisions.
            -- e.g., local router_key = "gateway_rotuer:" .. gateway_id .. ":" .. device_id
            
            -- 1. Clean up Login Slot
            local slot = conn_id_num % slot_size
            local login_slot_key = "login_slot_set_{" .. slot .."}"
            -- The meta value format is "deviceID|connID" in the current implementation.
            local login_slot_meta = device_id .. "|" .. conn_id_str
            redis.call("SREM", login_slot_key, login_slot_meta)

            -- 2. Clean up Router information
            local router_key = "gateway_rotuer_" .. device_id
            redis.call("DEL", router_key)

            -- 3. Clean up the "last message" for downlink ACK
            local last_msg_key = "last_msg_{" .. slot .. "}_" .. conn_id_str
            redis.call("DEL", last_msg_key)
            
            -- 4. Clean up uplink idempotency keys (max_client_id)
            -- This is the most complex part without sessionIDs.
            -- The current implementation uses a wildcard key.
            -- We will use SCAN to find and delete all related keys. This can be slow.
            -- TODO: A better approach is to have the application pass all associated sessionIDs
            -- to this script for precise deletion.
            local pattern = "max_client_id_{" .. slot .. "}_" .. conn_id_str .. "_*"
            local cursor = "0"
            repeat
                local result = redis.call("SCAN", cursor, "MATCH", pattern, "COUNT", 100)
                cursor = result[1]
                local keys = result[2]
                if #keys > 0 then
                    redis.call("DEL", unpack(keys))
                end
            until cursor == "0"

            return 1
        `,
	},
}

// init lua script
func initLuaScript(ctx context.Context) {
	for name, part := range luaScriptTable {
		cmd := rdb.ScriptLoad(ctx, part.LuaScript)
		if cmd == nil {
			panic(fmt.Sprintf("lua init failed lua=%s", name))
		}
		if cmd.Err() != nil {
			panic(cmd.Err())
		}
		part.Sha = cmd.Val()
	}
}