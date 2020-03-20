package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"seckill.fronted/rabbitmq"
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

//设置GetOne 秒杀接口请求地址 数量控制接口服务器内网ip地址  或者getOne的SLB内网IP地址
var GetOneIp = "127.0.0.1"

var GetOnePort = "8084"

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
		return a.GetDataFromMap(uid.Value)
	} else {
		//不是本机则充当代理访问数据返回结果
		return a.GetDataFromOtherMap(hostRequest, req)
	}
}

//获取本机map 并且处理业务逻辑 返回的结果类型为bool类型
func (a *AccessControl) GetDataFromMap(uid string) (isOk bool) {
	uidInt, err := strconv.Atoi(uid)
	if err != nil {
		return false
	}
	data := a.GetNewRecord(uidInt)

	//执行逻辑判断 这里简略设置一下执行逻辑
	if data != nil {
		return true
	}

	return
}

//获取其他节点处理结果 充当服务器代理的角色
func (a *AccessControl) GetDataFromOtherMap(host string, request *http.Request) bool {
	hostUrl := "http://" + host + ":" + port + "/check"
	response, body, err := GetCurl(hostUrl, request)
	if err != nil {
		return false
	}

	//判断状态
	if response.StatusCode == 200 {
		if string(body) == "true" {
			return true
		} else {
			return false
		}
	}

	return false
}

//模拟http请求
func GetCurl(hostUrl string, request *http.Request) (response *http.Response, body []byte, err error) {
	//获取uid
	uidPre, err := request.Cookie("uid")
	if err != nil {
		return
	}

	//获取sign
	uidSign, err := request.Cookie("sign")
	if err != nil {
		return
	}

	//模拟接口访问
	client := &http.Client{}
	req, err := http.ReadRequest("GET", hostUrl, nil)
	if err != nil {
		return
	}

	//手动指定，排查多余cookies
	cookieUid := &http.Cookie{Name: "uid", Value: uidPre.Value, Path: "/"}
	cookieSign := &http.Cookie{Name: "sign", Value: uidSign.Value, Path: "/"}

	//添加cookie到模拟请求中
	req.AddCookie(cookieUid)
	req.AddCookie(cookieSign)

	//获取返回结果
	response, err = client.Do(req)
	defer response.Body.Close()
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}

	return
}

//执行check正常业务逻辑
func Check(rw http.ResponseWriter, req *http.Request) {
	//执行正常的业务逻辑
	fmt.Println("执行check!")

	queryForm, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil && len(queryForm["productID"] <= 0 && len(queryForm["productID"][0]) <= 0) {
		rw.Write([]byte()"false")
		return
	}

	productString := queryForm["productID"][0]
	fmt.Println(productString)

	//获取用户cookie 用于获取用户id来使用
	userCookie, err := r.Cookie("uid")
	if err != nil {
		rw.Write([]byte("False"))
		return
	}

	//1.分布式权限验证
	right := accessControl.GetDistributedRight(req)
	if right == false {
		rw.Write([]byte("false"))
		return
	}

	//2.获取数量控制权限 防止秒杀出现超卖现象
	hostUrl := "http://" + GetOneIp + ":" + GetOnePort + "/getOne"
	responseValidate, validateBody, err := GetCurl(hostUrl, req)
	if err != nil {
		rw.Write([]byte("false"))
		return
	}

	//判断数量控制接口请求状态
	if responseValidate.StatusCode == 200 {
		if string(validateBody) == "true" {
			//整合下单
			//获取商品ID
			productID, err := strconv.ParseInt(productString, 10, 64)
			if err != nil {
				rw.Write([]byte("false"))
				return
			}

			//获取用户ID
			userID, err := strconv.ParseInt(userCookie.Value, 10, 64)
			if err != nil {
				rw.Write([]byte("false"))
				return
			}

			//创建消息体
			message := NewMessage(userID, productID)
			//消息体类型转化
			byteMessage, err := json.Marshal(message)
			if err != nil {
				rw.Write([]byte("false"))
				return
			}

			//生产消息
			err = rabbitMqValidate.PublishSimple(string(byteMessage))
			if err != nil {
				rw.Write([]byte("false"))
				return
			}

			rw.Write([]byte("true"))
			return
		}
	}
}

//统一验证拦截器 每个接口都需要提前验证
func Auth(rw http.ResponseWriter, req *http.Request) error {
	fmt.Println("执行验证!")
	//添加基于Cookie的权限验证
	if err := CheckUserInfo(req); err != nil {
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

	//获取本机ip地址
	localIp, err := common.GetIntranceIp()
	if err != nil {
		fmt.Println(err)
	}

	localHost = localIp
	fmt.Println("本机服务器ip地址: ", localhost)

	//使用rabbitmq 发送消息校验服务器
	rabbitMqValidate := rabbitmq.NewRabbitMQSimple("imoocProduct")
	defer rabbitMqValidate.Destory()

	//1.过滤器
	filter := common.NewFilter()
	//2.注册拦截器
	filter.RegisterFilterUri("/check", Auth)
	//3.启动服务
	http.HandleFunc("/check", filter.Handle(Check))

	http.ListenAndServe(":8083", nil)
}
