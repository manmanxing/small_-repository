package myCache

/**

客户端功能实现

使用一致性哈希选择节点        是                                    是
    |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                    |  否                                    ↓  否
                    |----------------------------> 回退到本地节点处理。

*/


//通过 key 获取相应的节点 PeerGetter
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//从对应的group中查询缓存
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}