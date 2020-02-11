package main

import (
	"errors"
	"fmt"
	"net/http"

	"seckill.validate/encrypt"

	"seckill.validate/common"
)

//设置集群地址 最好内网ip 有助于提高访问的速度
var hostArray = []string{
	"127.0.0.1",
	"127.0.0.1",
}

//设置本台服务器ip地址
var localHost = "127.0.0.1"

//设置本机服务器端口
var port = "8081"

//设置一致性hash服务器对象
var hashConsistent *common.Consistent

//用来存放控制信息
type AccessControl struct {
	//用来存放用户想要存放的信息
	sourcesArray map[int]interface{}

	//加锁解决map并发读写安全
	*sync.RWMutex
}

//创建全局变量
var accessControl = &AccessControl{sourceArray: make(map[int]interface{})}

//获取用户存储中指定的数据
func (a *AccessControl) GetNewRecord(uid int) interface{} {
	a.RWMutex.RLock()
	data := a.sourceArray[uid]
	defer a.RWMutex.RUnlock()
	return data
}

//设置数据
func (a *AccessControl) SetNewRecord(uid int, data interface{}) {
	a.RWMutex.Lock()
	defer a.RWMutex.Unlock()
	a.sourceArray[uid] = "用户设置数据"
}

//获取用户分布式权限
func (a *AccessControl) GetDistributedRight(req *http.Request) bool {
	//获取用户UID 从用户的请求里面获取
	uid, err := req.Cookie("name")
	if err != nil {
		return false
	}

	//采用一致性算法 根据用户id 判断获取具体机器
	hostRequest, err := hashConsistent.Get(uid.Value)
	if err != nil {
		return false
	}

	//判断是否是本机
	if hostRequest == localHost {
		//执行本机数据读取和校验
	} else {
		//不是本机则充当代理访问数据返回结果
	}
}

//执行check正常业务逻辑
func Check(rw http.ResponseWriter, req *http.Request) {
	//执行正常的业务逻辑
	fmt.Println("执行check!")
}

//统一验证拦截器 每个接口都需要提前验证
func Auth(rw http.ResponseWriter, req *http.Request) error {
	fmt.Println("执行验证!")
	//添加基于Cookie的权限验证
	if err := CheckUserInfo(req) {
		return err
	}
	return nil
}

//执行校验用户登录的拦截器 基于Cookie的权限校验 用户的身份校验
func CheckUser(req *http.Request) error {
	//获取Uid, cookie
	uidCookie, err := req.Cookie("uid")
	if err != nil {
		return errors.New("从Cookie中获取用户Uid失败")
	}

	//获取用户加密串
	signCookie, err := req.Cookie("sign")
	if err != nil {
		return errors.New("用户加密串 Cookie 获取失败! ")
	}

	//解密加密串
	signByte, err := encrypt.DePwdCode(signCookie.Value)
	if err != nil {
		return errors.New("加密串已被篡改过!")
	}

	fmt.Println("结果比对")
	fmt.Println("用户Id: " + uidCookie.Value)
	fmt.Println("解密后的用户ID: " + string(signByte))
	if checkInfo(uidCookie.Value, string(signByte)) {
		return nil
	}

	return errors.New("用户身份校验失败")
}

//自定义逻辑判断
func checkInfo(checkStr, signStr string) bool {
	if checkStr == signStr {
		return true
	}

	return false
}

func main() {
	//负载均衡器设置
	//采用一致性哈希算法
	hashConsistent = common.NewConsistent()
	//采用一致性hash算法,添加节点到hash环上
	for _, v := range hostArray {
		hashConsistent.Add(v)
	}

	//1.过滤器
	filter := common.NewFilter()
	//2.注册拦截器
	filter.RegisterFilterUri("/check", Auth)
	//3.启动服务
	http.HandleFunc("/check", filter.Handle(Check))

	http.ListenAndServe(":8083", nil)
}
