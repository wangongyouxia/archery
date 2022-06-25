# Archery

一个golang压测框架，通过代码自定义测试场景，通过web ui指定tps，支持分布式部署。

## 效果图展示
![image](https://github.com/wangongyouxia/archery/raw/master/static/result.png)

## 用户指引
(1) 下载工程源码.

(2) 修改work.go, 实现里面的Work()函数, 函数内执行单个压测操作, 并返回(bool,int)类型的两个数值, 第一个数值标记成功(true)与失败(false), 第二个数值为消耗的时间, 单位为毫秒, 当有多个任务需要按比例同时测试时, 可按照如下格式修改LoadWorkList函数.
```
func (task *Task) Work1() (bool, int) {
	...
}

func (task *Task) Work2() (bool, int) {
	...
}

func (task *Task) LoadWorkList() ([]WorkInfo,*Task) {
	var res []WorkInfo
	res = append(res,WorkInfo{task.Work1,2,"title-1"}) // 2表示每个事务发2次请求
	res = append(res,WorkInfo{task.Work2,5,"title-2"}) // 5表示每个事务发5次请求
	return res,task
}
```

(3) 执行`go build`如果你没有go，请先安装go.

(4) 执行`./archery`启动工具(如果需要指定访问端口, 通过-port 指定).

(5) 如果你要监控目标服务器资源（cpu与内存占用）, 需要把archery可执行文件复制到被监控服务器, 并执行`./archery -mode monitor -master_addr [启动工具所在的机器地址] `启动对目标服务器的监控.

(6) 访问http://[启动工具所在的机器IP]:8018(或你通过-port指定的端口), 管理你的测试任务.

## 分布式部署指引
(1) 修改work.go, 实现里面的Work()函数, 函数内执行你要的操作, 并返回(bool,int)类型的两个数值, 第一个数值标记成功(true)与失败(false), 第二个数值为消耗的时间, 单位为毫秒.

(2) 执行`go build`编译项目, 把可执行文件archery复制到你的控制机(master只负责任务控制，不执行任务)和执行机(slave).

(3) 在控制机(master)执行`./archery -mode master`启动控制机(master), 如果需要指定访问端口, 通过-port 指定

(4) 在执行机(slave)执行`./archery -mode slave -master_addr [控制机的机器地址]` 启动执行机(slave)

(5) 如果你要监控目标服务器资源（cpu与内存占用）, 需要把可执行文件archery复制到目标服务器, 然后执行`./archery -mode monitor -master_addr [控制机的机器地址]`启动对目标服务器的监控.

(6) 访问http://[控制机的机器IP]:8018(或你通过-port指定的端口), 管理你的测试任务.

## 注意事项
分布式执行时, 控制机(master)和执行机(slave)网络应能相互访问, 如果你要监控服务器资源, 被监控的机器和控制机(master)之间也应能相互访问.

## 工具对比
| -        | ab                 | jmeter                        | locust                                           | archery                                              | 云压测                           |
| :------- | :----------------- | :---------------------------- | :----------------------------------------------- | :--------------------------------------------------- | :------------------------------- |
| 实现语言 | C                  | Jave                          | Python                                           | Go                                                   | -                                |
| UI界面   | 无                 | 有                            | 有                                               | 有                                                   | 有                               |
| 优点     | 使用简单，上手简单 | 功能丰富，支持生成HTML报告    | 支持分布式、压测数据支持导出，支持自定义测试场景 | 支持分布式，对go用户友好，性能强，支持自定义测试场景 | 没有环境部署、找测试机资源的麻烦 |
| 缺点     | 只支持简单测试场景 | 难以直接控制qps，需要反复调整 | 需要写python代码，性能较差                       | 需要写golang代码                                     | 功能丰富程度取决于测试平台       |


## 压测常见问题
**time_wait积累，可用端口耗尽无法建立新连接，报错*Cannot assign
requested address***

出现原因：主动关闭tcp连接的一方，系统默认需要等待2MSL时间，才能释放连接资源，这段时间内该端口对应的四元组不可用，当产生速度比释放速度快，就会导致time_wait状态连接积累，常见于短链接测试，通常会出现报错Cannot
assign requested address，可通过

`netstat -an|grep TIME_WAIT|wc -l`

确认此问题，出现问题时，这个数值通常是万数量级的，问题解决方法：

编辑内核文件/etc/sysctl.conf，加入以下内容

```
net.ipv4.tcp_syncookies = 1 #表示开启SYN Cookies。当出现SYN等待队列溢出时，启用cookies来处理，可防范少量SYN攻击，默认为0，表示关闭；
net.ipv4.tcp_tw_reuse = 1 #表示开启重用。允许将TIME-WAIT sockets重新用于新的TCP连接，默认为0，表示关闭；
net.ipv4.tcp_tw_recycle = 1 #表示开启TCP连接中TIME-WAIT sockets的快速回收，默认为0，表示关闭，此配置在4+内核版本已经废弃。
net.ipv4.tcp_fin_timeout = 30 #修改系默认的 TIMEOUT 时间
net.ipv4.ip_local_port_range = 1024 65535 #修改可用端口范围
```

然后执行 /sbin/sysctl -p 让参数生效

使用长连接，这是最好的解决办法，但有时候压测就是需要测短链接场景，则无法使用此方法

**打开文件数过多，无法建立连接**

因为在linux中，连接也算打开的文件，所以也受最大打开文件数限制，可以通过ulimit
-Hn查看限制当前shell打开的文件数，这个问题通常报错Too many open
files，可通过ls -1 /proc/\${PID}/fd \| wc
-l命令查看当前打开的文件确认该问题，通过ulimit -n 65535解决

**qps到一定数值就无法继续压上去**

通常需要检查客户端CPU使用率、网络带宽限制，因为这两个限制通常不报错，但是会导致qps压不上去。
