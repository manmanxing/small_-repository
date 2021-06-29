package myCache

import (
	"errors"
	"fmt"
	"log"
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

//回调函数，如果缓存不存在，用户自己实现的用于获取源数据
type GetterFunc func(key string) ([]byte, error)

func (g GetterFunc) Get(key string) ([]byte, error) {
	return g(key)
}

//对应缓存的命名空间，实际上是将不同缓存区分开
type Group struct {
	name      string     //group 的名字
	getter    Getter     //如果获取不到缓存，需要用户提供兜底的缓存获取函数
	mainCache cache      //该 group 中的缓存
	peers     PeerPicker //如果该 group 中不存在，需要查询其他节点
}

var (
	//定义一个全局的互斥锁，保证 Group 的并发访问
	groupMux sync.RWMutex
	//保存所有的缓存命名空间
	groups = make(map[string]*Group)
)

//name：该缓存空间的名称
//cacheBytes：该缓存空间的缓存大小
//getter：回调函数，如果缓存不存在，用户自己实现的用于获取源数据
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

//根据 key 获取缓存
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is empty")
	}
	//如果能从缓存中获取，就返回
	if v, ok := g.mainCache.Get(key); ok {
		fmt.Println(fmt.Sprintf("%s cache hit", key))
		return v, nil
	}
	//如果从缓存中没有获取到，那么从另外地方获取
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	//使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。
	//若是本机节点或失败，则回退到 getLocally()
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

//使用用户指定的兜底函数获取缓存
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

func (g *Group) RegisterPeers(peers PeerPicker) {
	if peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
