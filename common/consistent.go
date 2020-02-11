package common

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

//一致性hash算法实现服务器和资源的均匀分布 用于权限验证的横向扩展

//声明新的切片类型 长度是2的32次方-1
type uints []uint32

//返回切片的长度
func (u *uints) len() int {
	return len(u)
}

//比对两个数的大小
func (u *uints) less(i, j int) bool {
	return u[i] < u[j]
}

//切片中两个值的交换
func (u *uints) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

//当hash环上没有数据时，提示错误
var errorEmpty = errors.New("Hash 环无数据")

//创建结构体保存一致性hash信息
type Consistent struct {
	//hash环, key为hash值, 值存放节点的信息
	circle map[uint32]string

	//已经排序的节点hash切片
	sortedHashes uints

	//虚拟节点个数，用来增加hash的平衡性
	VirtualNode int

	//map 读写锁
	sync.RWMutex
}

//创建一致性hash算法结构体, 设置默认虚拟节点个数
func NewConsistent() *Consistent {
	return &Consistent{
		//初始化变量
		circle: make(map[uint32]string),
		//设置虚拟节点个数
		VirtualNode: 20,
	}
}

//自动生成key值
func (c *Consistent) generateKey(element string, index int) string {
	//副本key生成逻辑
	return element + strconv.Itoa(index)
}

//获取hash位置
func (c *Consistent) hashKey(key string) uint32 {
	if len(key) < 64 {
		//声明一个数组长度为64
		var srcatch [64]byte
		//拷贝数据到数组中
		copy(srcatch[:], key)
		//使用IEEE 多项式返回数据的CRC-32校验和
		return crc32.ChecksumIEEE(srcatch[:len(key)])
	} else {
		return crc32.ChecksumIEEE([]byte(key))
	}
}

func (c *Consistent) updateSortedHashes() {
	hashes := c.sortedHashes[:0]
	//判断切片的容量 是否过大 如果过大 则重置
	if cap(c.sortedHashes)/(c.VirtualNode*4) > len(c.circle) {
		hashes = nil
	}

	//添加hashes
	for k := range c.circle {
		hashes = append(hashes, k)
	}

	//对所有节点hash值进行排序,
	//方便二分查找数据存放在哪一个节点上面
	sort.Sort(hashes)

	//排序完之后 进行重新赋值
	c.sortedHashes = hashes
}

//向hash环中添加节点
func (c *Consistent) Add(element string) {
	//加锁
	c.Lock()
	//解锁
	defer c.UnLock()

	c.add(element)
}

//添加节点
func (c *Consistent) add(element string) {
	//循环虚拟节点，设置副本
	for i := 0; i < c.VirtualNode; i++ {
		//根据生成的节点添加到hash环中
		c.circle[c.hashkey(c.generateKey(element, i))] = element
	}
	//更新排序
	c.updateSortedHashes()
}

//移除节点
func (c *Consistent) remove(element string) {
	//删除相关的副本节点信息
	for i := 0; i < VirtualNode; i++ {
		delete(c.circle, c.hashKey(c.generate(element, i)))
	}
	//更新排序
	c.updateSortedHashes()
}

//从hash环中删除节点
func (c *Consistent) Remove(element string) {
	c.Lock()
	defer c.UnLock()
	c.remove(element)
}

//顺时针查找最近的节点
func search(key uint32) int {
	//查找算法
	f := func(x int) bool {
		return c.sortedHashes[x] > key
	}

	//使用二分法查找节点数据 搜索指定切片满足条件的最小值的切片值的下标
	i := sort.Search(len(c.sortedHashes), f)

	//如果超出范围 则设置i=0
	if i >= len(c.sortedHashes) {
		i = 0
	}

	return i
}

//根据数据标示获取最近的服务器节点信息
func (c *Consistent) Get(name string) (string, error) {
	//加锁
	c.RLock()
	//解锁
	defer c.UnRLock()
	//如果为零则返回错误
	if len(c.circle) == 0 {
		return "", errorEmpty
	}

	//计算hash值
	key := c.hashKey(name)
	i := c.search(key)

	return c.circle[c.sortedHashes[i]], nil
}
