package myCache


//对缓存的只读数据结构
type ByteView struct {
	//用于存储真实的缓存值
	//选择 byte 是为了能支持任意的数据类型的存储，比如字符串，图片等
	b []byte
}

//实现 Value 接口
func (v ByteView)Len()int  {
	return len(v.b)
}

func (v ByteView)String()string  {
	return string(v.b)
}

//针对 b []byte 进行复制，防止外部程序修改
func (v ByteView)Copy()[]byte  {
	return cloneBytes(v.b)
}

func cloneBytes(b []byte)[]byte  {
	c := make([]byte,len(b))
	copy(c,b)
	return c
}