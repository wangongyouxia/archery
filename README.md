# Archery

一个golang压测框架，通过代码自定义测试场景

## 用户指引
(1) 下载工程源码.

(2) 修改work.go, 实现里面的Work()函数, 函数内执行你要的操作, 并返回(bool,int64)类型的两个数值, 第一个数值标记成功(true)与失败(false), 第二个数值为消耗的时间, 单位为毫秒.

(3) 执行` go build`如果你没有go，请先安装go.

(4) 执行` ./archery`启动工具(如果需要指定访问端口, 通过-port 指定).

(5) 如果你要监控目标服务器, 需要把archery可执行文件复制到被监控服务器, 并执行` ./archery -mode monitor -master_addr [启动工具所在的机器地址] `启动对目标服务器的监控.

(6) 访问http://[启动工具所在的机器IP]:6767(或你通过-port指定的端口), 管理你的测试任务.

## 分布式部署指引
(1) 修改work.go, 实现里面的Work()函数, 函数内执行你要的操作, 并返回(bool,int64)类型的两个数值, 第一个数值标记成功(true)与失败(false), 第二个数值为消耗的时间, 单位为毫秒.

(2) 执行` go build`编译项目, 把可执行文件archery复制到你的控制机(master只负责任务控制，不执行任务)和执行机(slave).

(3) 在控制机(master)执行` ./archery -mode master`启动控制机(master), 如果需要指定访问端口, 通过-port 指定

(4) 在执行机(slave)执行` ./archery -mode slave -master_addr [控制机的机器地址]` 启动执行机(slave)

(5) 如果你要监控目标服务器, 需要把可执行文件archery复制到目标服务器, 然后执行`./archery -mode monitor -master_addr [控制机的机器地址]`启动对目标服务器的监控.

(6) 访问http://[控制机的机器IP]:6767(或你通过-port指定的端口), 管理你的测试任务.

## 注意事项
分布式执行时, 控制机(master)和执行机(slave)网络应能相互访问, 如果你要监控服务器资源, 被监控的机器和控制机(master)之间也应能相互访问.
