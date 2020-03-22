package common

import "net/http"

//声明一个新的数据类型（函数类型)
type FilterHandle func(rw http.ResponseWriter, req *http.Request) error

//拦截器结构体
type Filter struct {
	//用来存储需要拦截的url
	filterMap map[string]FilterHandle
}

//Filter初始化函数
func NewFilter() *Filter {
	return &Filter{filterMap: make(map[string]FilterHandle)}
}

//注册拦截器
func (f *FilterHandle) RegisterFilterUri(uri string, handler FilterHandle) {
	f.filterMap[uri] = handler
}

//根据Uri获取对应的handle
func (f *FilterHandle) GetFilterHandl(uri string) FilterHandle {
	return f.filterMap[uri]
}

//声明新的函数类型
type WebHandle func(rw http.ResponseWriter, req *http.Request)

//执行拦截器
func (f *FilterHandle) Handle(webHandle WebHandle) func(re http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		for path, handle := range f.filterMap {
			if strings.Contains(r.RequestURI, path) {
				//执行当前拦截业务逻辑
				er := handle(rw, req)
				if err != nil {
					rw.Write([]bytr(err.Error()))
					return
				}
				//跳出循环
				break
			}
		}
		//执行正常的业务逻辑
		webHandle(rw, req)
	}
}
