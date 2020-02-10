package main

import (
	"fmt"
	"net/http"

	"seckill.validate/common"
)

//执行check正常业务逻辑
func Check(rw http.ResponseWriter, req *http.Request) {
	//执行正常的业务逻辑
	fmt.Println("执行check!")
}

//统一验证拦截器 每个接口都需要提前验证
func Auth(rw http.ResponseWriter, req *http.Request) error {
	return nil
}

func main() {
	//1.过滤器
	filter := common.NewFilter()
	//2.注册拦截器
	filter.RegisterFilterUri("/check", Auth)
	//3.启动服务
	http.HandleFunc("/check", filter.Handle(Check))

	http.ListenAndServe(":8083", nil)
}
