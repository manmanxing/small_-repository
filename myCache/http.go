package myCache

import (
	"log"
	"net/http"
	"strings"
)

/**
分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
*/

const defaultBasePath = "/mycache/"

type HTTPPool struct {
	self     string //记录自己的地址
	basePath string //路径前缀
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