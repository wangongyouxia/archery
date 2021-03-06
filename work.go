package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"strings"
	"time"
)

type WorkInfo struct {
	WorkFunc   func() (bool,int)
	Ratio  float64
	Title      string
}

type Task struct {
	/*任务结构体，因部分测试场景需要共享数据，故使用此结构体避免全局变量的使用，如本demo的场景使用了同一个client实例
	Task struct, which is used for sharing data within different testcases. In this example, all testcases use the same instance
	*/
	client	http.Client
}

func (task *Task)Init() {
	/*此函数，每次开始测试只会执行一次(Init执行一次、Work执行多次)，用于初始化
	*/
	task.client = http.Client{ Timeout : time.Second }
}

func (task *Task) Work() (bool, int) {
	/*此函数为压测具体事项，如下为GET github首页的示例
	this is an example for requesting github webpage with GET method
	*/
	req, _ := http.NewRequest("GET", "https://github.com/", strings.NewReader(""))
	req.Header.Add("User-Agent","curl/7.64.1")
	var start_time, cost_time int
	start_time = int(time.Now().UnixNano() / 1e6)
	resp, err := task.client.Do(req) //此处使用了上面初始化的client
	if err != nil {
		fmt.Println(err)
		return false, 0
	}
	ioutil.ReadAll(resp.Body)
	end_time := int(time.Now().UnixNano() / 1e6)
	cost_time = end_time - start_time
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		fmt.Println(resp.Header)
		fmt.Printf("Expect 200, %d GET!", resp.StatusCode)
		return false, 0
	}
	return true, cost_time
	//fmt.Println(resp)
}

func (task *Task)Uninit() {
	/*此函数，每一次压测只会执行一次(同Init)，用于释放初始化的资源，这个示例没有资源需要释放
	*/
}

func (task *Task)LoadWorkList() ([]WorkInfo,*Task) {
	var res []WorkInfo
	res = append(res,WorkInfo{task.Work,1,"github"})
	return res,task
}
