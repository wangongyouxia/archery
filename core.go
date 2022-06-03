package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"sort"
)

//用于返回给前端的结构体，执行过程中，每秒更新一次，每收到一个前端的请求，就返回最新的数据
type OneSecondData struct {
	Req               int64 `json:"request_num"`
	SuccResp         int64 `json:"success_response_num"`
	AverageCostTime int64 `json:"average_cost_time"`
	FailedNum        int64 `json:"failed_num"`
	Timestamp        int64 `json:"time_stamp"`
	MinCostTime int  `json:"min_cost_time"`
	FiftyPercentCostTime int `json:"fifty_percent_cost_time"`
	NintyPercentCostTime int `json:"ninty_percent_cost_time"`
	NintyNinePercentCostTime int `json:"ninty_nine_percent_cost_time"`
	MaxCostTime int `json:"max_cost_time"`
	RawData []int `json:"raw_data"`
	TotalReqNum    int64 `json:"total_request_num"`
	TotalRespNum   int64 `json:"total_succ_response_num"`
	TotalSuccRespTime   int64 `json:"total_succ_resp_time"`
	TotalFailedNum int64 `json:"total_failed_num"`
}


//Archery结构体，用于存储压测执行过程中需要的各个数据
type Archery struct {
	request_num_in_one_second       int64
	succ_response_num_in_one_second int64
	total_request_num               int64
	total_succ_response_num         int64
	total_response_time             int64
	total_failed_num                int64
	total_resp_time_in_one_second   int64
	failed_num_in_one_second        int64
	sleep_time_in_microsecond       int64
	Status                          int64 //0:stop, 1:start
	last_second_data                OneSecondData
	Last_second_whole_test_data     []int
	slice_lock                      sync.Mutex
	cpu_num                         int
	work                            func()(bool,int)
	description                     string
	ratio                           float64
	task                            *Task
}



//停止执行压测函数
func (archery *Archery) StopLoadTest() {
	archery.Status = 0
	atomic.StoreInt64(&(archery.request_num_in_one_second), 0)
	atomic.StoreInt64(&(archery.succ_response_num_in_one_second), 0)
	atomic.StoreInt64(&(archery.total_resp_time_in_one_second), 0)
	atomic.StoreInt64(&(archery.failed_num_in_one_second), 0)
	tmp := OneSecondData{}
	tmp.TotalReqNum   = archery.last_second_data.TotalReqNum //  int64 `json:"total_request_num"`
    tmp.TotalRespNum = archery.last_second_data.TotalRespNum //  int64 `json:"total_succ_response_num"`
    tmp.TotalSuccRespTime = archery.last_second_data.TotalSuccRespTime //  int64 `json:"total_succ_resp_time"`
    tmp.TotalFailedNum = archery.last_second_data.TotalFailedNum
	archery.last_second_data = tmp

}

//执行单个压测动作函数，并把执行结果统计到当前秒的数据中
func (archery *Archery) RunSingleJob(args []string) {
	succ, cost_time := archery.work()
	if archery.Status != 1 {
		return
	}
	if succ {
		archery.slice_lock.Lock()
		archery.Last_second_whole_test_data = append(archery.Last_second_whole_test_data,int(cost_time))
		archery.slice_lock.Unlock()
		atomic.AddInt64(&(archery.succ_response_num_in_one_second), 1)
		atomic.AddInt64(&(archery.total_succ_response_num), 1)
		atomic.AddInt64(&(archery.total_response_time), int64(cost_time))
		atomic.AddInt64(&(archery.total_resp_time_in_one_second), int64(cost_time))
	} else {
		atomic.AddInt64(&(archery.failed_num_in_one_second), 1)
		atomic.AddInt64(&(archery.total_failed_num), 1)
	}
}

//根据计算得出的时间间隔，循环等待延时并调用压测动作函数
func (archery *Archery) RunJobs(qps float64, wg *sync.WaitGroup,args []string) {
	for archery.Status == 1 {
		time.Sleep(time.Duration(archery.sleep_time_in_microsecond) * time.Microsecond)
		atomic.AddInt64(&(archery.request_num_in_one_second), 1)
		atomic.AddInt64(&(archery.total_request_num), 1)
		go archery.RunSingleJob(args)
	}
	wg.Done()
}

//动态调整时间间隔函数，如果实际qps比目标qps小了，就减小时间间隔，大了就增加
func (archery *Archery) DelayTimeAdjust(qps float64) {
	time.Sleep(time.Second)
	for archery.Status == 1 {
		time.Sleep(time.Second)
		if archery.last_second_data.Req > int64(qps) {
			archery.sleep_time_in_microsecond += int64(0.01 * float64(archery.sleep_time_in_microsecond))
		} else if archery.last_second_data.Req < int64(qps) {
			archery.sleep_time_in_microsecond -= int64(0.01 * float64(archery.sleep_time_in_microsecond))
		}
	}
}

func (archery *Archery) getLastSecondPercentData() (int,int,int,int,int) {
	var min_value,fifty,ninty,ninty_nine,max_value int
	snapshot_list := archery.Last_second_whole_test_data
	snapshot_len := len(archery.Last_second_whole_test_data)
	sort.Ints(snapshot_list)
	if snapshot_len > 0 {
		//min_value = archery.Last_second_whole_test_data[0]
		//max_value = archery.Last_second_whole_test_data[snapshot_len-1]
		ninty = snapshot_list[int(float64(snapshot_len) * 0.9)]//[int(float64(snapshot_len) * 0.9)]
		ninty_nine = snapshot_list[int(float64(snapshot_len) * 0.99)]
		fifty = snapshot_list[snapshot_len/2]
	}
	return min_value,fifty,ninty,ninty_nine,max_value
}

//统计上一秒数据放到OneSecondData中等待前端来取，并清零数据开始下一秒统计
func (archery *Archery) Controller() {
	for archery.Status == 1 {
		time.Sleep(time.Second)
		var average_resp_time int64
		if atomic.LoadInt64(&(archery.succ_response_num_in_one_second)) == 0 {
			average_resp_time = 0
		} else {
			average_resp_time = int64(atomic.LoadInt64(&(archery.total_resp_time_in_one_second)) / int64(atomic.LoadInt64(&(archery.succ_response_num_in_one_second))))
		}
		//fmt.Printf("total req:%d, total resp:%d, req/s:%d, resp/s:%d, average_resp_time:%d\n", archery.total_request_num, archery.total_succ_response_num, archery.request_num_in_one_second, archery.succ_response_num_in_one_second, average_resp_time)
		now := int64(time.Now().Unix())
		min_value,fifty,ninty,ninty_nine,max_value := archery.getLastSecondPercentData()
		archery.last_second_data = OneSecondData{archery.request_num_in_one_second, archery.succ_response_num_in_one_second, average_resp_time, archery.failed_num_in_one_second, now, min_value,fifty,ninty,ninty_nine,max_value,archery.Last_second_whole_test_data,archery.total_request_num,archery.total_succ_response_num,archery.total_response_time,archery.total_failed_num}
		archery.slice_lock.Lock()
		archery.Last_second_whole_test_data = nil
		archery.slice_lock.Unlock()
		atomic.StoreInt64(&(archery.request_num_in_one_second), 0)
		atomic.StoreInt64(&(archery.succ_response_num_in_one_second), 0)
		atomic.StoreInt64(&(archery.total_resp_time_in_one_second), 0)
		atomic.StoreInt64(&(archery.failed_num_in_one_second), 0)
	}
}


//返回上一秒数据
func (archery *Archery) GetSecondData(need_raw bool) OneSecondData {
	res := archery.last_second_data
	if !need_raw {
		res.RawData = nil
	}
	return res
}

//定时停止测试，暂时没用
func (archery *Archery) StopInTime(duration_time int64) {
	time.Sleep(time.Duration(duration_time) * time.Second)
	archery.Status = 0
}

//控制qps的函数，根据qps每秒增加数，计算出当前秒目标qps，并根据这个目标qps，算出需要等待的延时时间
func (archery *Archery) QpsController(start_qps float64, end_qps float64, qps_step float64, qps *float64,wg *sync.WaitGroup,args []string) {
	for archery.Status == 1 && *qps < end_qps {
		time.Sleep(time.Duration(1) * time.Second)
		*qps = *qps + qps_step
		//当预期qps达到逻辑cpu数50倍的时候，启动多个协程（数量和逻辑cpu数一致），同时起执行job，以最大化利用cpu性能。
		if runtime.NumCPU() * 50 < int(*qps) && archery.cpu_num == 1{
			archery.cpu_num = runtime.NumCPU()
			for t := 1; t < archery.cpu_num; t++ {
				wg.Add(1)
				go archery.RunJobs(*qps,wg,args)
			}
		}
		archery.sleep_time_in_microsecond = int64(float64(1000000) * float64(archery.cpu_num) / *qps)
	}
	archery.DelayTimeAdjust(*qps)
}

//开始测试函数，args参数为传入Work函数的参数，暂时没有使用
func (archery *Archery) StartLoadTest(start_qps float64, end_qps float64, qps_step float64, duration_time int64,args []string) {
	archery.last_second_data = OneSecondData{}
	archery.succ_response_num_in_one_second = 0
	archery.total_failed_num = 0
	archery.total_succ_response_num = 0
	archery.total_response_time = 0
	archery.total_request_num = 0
	archery.Status = 1
	//压测定时关闭，暂时没有使用这个功能
	if duration_time > 0 {
		go archery.StopInTime(duration_time)
	}
	//go archery.DelayTimeAdjust(qps)
	if qps_step <= 0 {
		qps_step = end_qps
	}
	qps := start_qps
	if qps == 0 {
		qps = qps_step
	}
	var wg sync.WaitGroup
	archery.cpu_num = 1 // cpu数初始值设置为1，在QpsController中修改
	archery.sleep_time_in_microsecond = int64(float64(1000000) / qps)
	go archery.QpsController(start_qps, end_qps, qps_step, &qps, &wg, args)
	go archery.Controller()
	wg.Add(1)
	go archery.RunJobs(qps, &wg,args)
	wg.Wait()
	return
}
