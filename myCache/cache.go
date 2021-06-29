package myCache

import (
	"github.com/manmanxing/small_repository/myCache/lru"
	"sync"
)

/**
实现缓存的并发控制
*/

type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64 //指定 lru cache maxBytes 的值
}

func (c *cache)Add(key string,value ByteView)  {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		//延迟初始化
		c.lru = lru.New(c.cacheBytes,nil)
	}
	c.lru.Add(key,value)
}

func (c *cache)Get(key string)(value ByteView,ok bool)  {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v,ok := c.lru.Get(key);ok{
		return v.(ByteView),ok
	}

	return
}