package lru

import (
	"container/list"
)

//非并发安全
type Cache struct {
	maxBytes  int64                         //允许使用的最大内存
	useBytes  int64                         //目前已经使用的内存，必须<= maxBytes
	list      *list.List                    //使用标准库实现的双向链表list.List
	cache     map[string]*list.Element      //实现查询复杂度为O(1),key就是 entry 中的 key
	onEvicted func(key string, value Value) //某条记录被移除时的回调函数，可以为nil
}

//双向链表节点的数据 kv 类型，会保存在 list.Element 的 Value 中
type entry struct {
	key   string
	value Value //允许值是实现了 Value 接口的任意类型
}

type Value interface {
	Len() int //记录value所使用的bytes
}

//实例化 Cache
func New(_maxBytes int64, _oEvicted func(key string, value Value)) (cache *Cache) {
	if _maxBytes <= 0 {
		panic("max bytes cant <= 0")
	}

	return &Cache{
		maxBytes:  _maxBytes,
		useBytes:  0,
		list:      list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: _oEvicted,
	}
}

//第一步是从字典中找到对应的双向链表的节点
//第二步，将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.list.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//删除实际上就是lru的缓存淘汰：即移除最近最少访问的节点
func (c *Cache) Delete() {
	ele := c.list.Back()
	if ele != nil {
		//在双向链表中移除
		c.list.Remove(ele)
		//在map中移除
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		//计算 Cache 已使用的内存
		c.useBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//如果需要删除回调
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

//新增与修改
//注意已使用内存的计算与上限
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		//如果键存在，则更新对应节点的值，并将该节点移到前面。
		c.list.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.useBytes = c.useBytes + int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		//不存在则是新增场景，首先添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系
		ele := c.list.PushFront(&entry{
			key:   key,
			value: value,
		})
		c.cache[key] = ele
		c.useBytes = c.useBytes + int64(len(key)) + int64(value.Len())
	}

	//这里需要判断已使用内存是否超过了最大内存，如果超了就执行删除
	//如果第一次插入时， maxBytes < useBytes，那么将永远获取不到缓存
	for c.maxBytes < c.useBytes {
		c.Delete()
	}
}

//用于计算目前双向链表中已存储的数据条目
func (c *Cache) Len() int {
	return c.list.Len()
}