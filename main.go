package main

import (
	"fmt"
	"net/http"

	"seckill.validate/common"
)

//统一验证拦截器 每个接口都需要提前验证
func Auth(rw http.ResponseWriter, req *http.Request) error {
	fmt.Println("验证成功")
	return nil
}

func main() {
	//1.过滤器
	filter := common.NewFilter()
	//2.注册拦截器
	filter.RegisterFilterUri("/check", Auth)
	//3.启动服务
	http.HandleFunc("/check", filter.Handle(check))
}
