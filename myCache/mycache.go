package myCache

import (
	"errors"
	"fmt"
	"sync"
)

/**
负责与用户的交互，并且控制缓存值存储和获取的流程

流程如下：
                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

type Getter interface {
	Get(key string) ([]byte, error)
}

//回调函数，如果缓存不存在，提供给用户用于获取源数据
type GetterFunc func(key string) ([]byte, error)

func (g GetterFunc) Get(key string) ([]byte, error) {
	return g(key)
}

//对应缓存的命名空间，实际上是将不同缓存区分开
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	//定义一个全局的互斥锁，保证 Group 的并发访问
	groupMux sync.RWMutex
	//保存所有的缓存命名空间
	groups = make(map[string]*Group)
)

//name：该缓存空间的名称
//cacheBytes：该缓存空间的缓存大小
//getter：获取源数据，用户自己实现
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("getter is nil")
	}

	groupMux.Lock()
	defer groupMux.Unlock()

	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
	}

	groups[name] = g

	return g
}

//获取缓存命名空间
func GetGroup(name string) *Group {
	groupMux.RLock()
	defer groupMux.RUnlock()
	return groups[name]
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is empty")
	}

	if v, ok := g.mainCache.Get(key); ok {
		fmt.Println(fmt.Sprintf("%s cache hit", key))
		return v, nil
	}
	//如果从缓存中没有获取到，那么从回调函数中获取
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.AddCache(key, value)
	return value, nil
}

func (g *Group) AddCache(key string, value ByteView) {
	g.mainCache.Add(key, value)
}