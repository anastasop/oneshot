namespace java jhug.anastasop.memcached.protocol

struct Item {
	1:string key,
	2:binary value
	3:i32 flags = 0,
	4:i32 expiration = 0,
	5:i64 cas_id
}

enum MemcachedError {
	ERR_CACHE_MISS = 1,
	ERR_CAS_CONFLICT = 2,
	ERR_NOT_STORED = 3,
	ERR_SERVER_ERROR = 4,
	ERR_NO_STATS = 5,
	ERR_MALFORMED_KEY = 6,
	ERR_NO_SERVERS = 7
}

exception MemcachedException {
	1:MemcachedError error,
	2:string explanation
}


service Memcached {
	void set_item(1:Item item),
	Item get_item(1:string key),
	void delete_item(1:string key)
}
