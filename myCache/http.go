package myCache

import (
	"fmt"
	"github.com/manmanxing/small_repository/myCache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

/**
分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
*/

const (
	defaultBasePath = "/mycache/"
	defaultReplicas = 50
)

//HTTP服务端代码实现
type HTTPPool struct {
	self     string //记录自己的地址
	basePath string //路径前缀
	mux      sync.Mutex
	peers    *consistenthash.Map //用来根据具体的 key 选择节点。
	//映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
	//key:远程节点的url
	httpGetter map[string]*httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

//实现 http.Handler.ServeHttp 方法
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log(r.Method + "-" + r.URL.Path)

	//访问路径示例：/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group:"+groupName, http.StatusNotFound)
		return
	}

	value, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(value.Copy())
}

func (p *HTTPPool) Log(msg string) {
	log.Printf("[server %s] %s", p.self, msg)
}

//HTTP客户端实现
type httpGetter struct {
	baseURL string
}

//从节点查询缓存的具体实现
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))

	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body:%v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

//实例化了一致性哈希算法，并且添加了传入的节点。
//并为每一个节点创建了一个 HTTP 客户端 httpGetter。
func (p *HTTPPool) Set(peers ...string) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetter = make(map[string]*httpGetter, len(peers))

	for _, peer := range peers {
		p.httpGetter[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer " + peer)
		return p.httpGetter[peer], true
	}

	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)