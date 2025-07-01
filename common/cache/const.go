package cache

import "time"

const (
	MaxClientIDKey  = "max_client_id_{%d}_%d"
	LastMsgKey      = "last_msg_{%d}_%d"
	LoginSlotSetKey = "login_slot_set_{%d}" // though hash tag guarantees that in cluster mode the key is on the same shard
	TTL7D           = 7 * 24 * time.Hour
)