package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type TestDataSum struct {
	TotalReqNum    int64 `json:"total_request_num"`
	TotalRespNum   int64 `json:"total_succ_response_num"`
	TotalSuccRespTime   int64 `json:"total_succ_resp_time"`
	TotalFailedNum int64 `json:"total_failed_num"`
}

type TargetServerOneSecondData struct {
	TimeStampInSec     int `json:"time_stamp"`
	CpuRate10000       int `json:"cpu_rate"`
	MemoryUsage10000   int `json:"memory_usage"`
	LastSecondIdleCpu  int `json:"last_second_idle_cpu"`
	LastSecondTotalCpu int `json:"last_second_total_cpu"`
}

type TargetServer struct {
	Addr               string `json:"addr"`
	TimeStampInMs      int64  `json:"time_stamp"`
	LastSecondIdleCpu  int    `json:"last_second_idle_cpu"`
	LastSecondTotalCpu int    `json:"last_second_total_cpu"`
}

func (ts *TargetServer) GetTargetServerData() (TargetServerOneSecondData, bool) {
	var target_server_data TargetServerOneSecondData
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/get_server_second_data", ts.Addr), strings.NewReader(""))
	if err != nil {
		log.Println(err)
		return target_server_data, false
	}
	resp, http_err := client.Do(req)
	if http_err != nil {
		log.Println(http_err)
		return target_server_data, false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	err_json := json.Unmarshal(body, &target_server_data)
	if err_json != nil {
		log.Println(err_json)
		return target_server_data, false
	}
	if ts.LastSecondTotalCpu != target_server_data.LastSecondTotalCpu {
		delta_idle_cpu := target_server_data.LastSecondIdleCpu - ts.LastSecondIdleCpu
		delta_total_cpu := target_server_data.LastSecondTotalCpu - ts.LastSecondTotalCpu
		ts.LastSecondIdleCpu = target_server_data.LastSecondIdleCpu
		ts.LastSecondTotalCpu = target_server_data.LastSecondTotalCpu
		target_server_data.CpuRate10000 = 10000 - (10000 * delta_idle_cpu / delta_total_cpu)
	}
	defer resp.Body.Close()
	return target_server_data, true
}

func (ts *TargetServer) ExitTargetServer() bool {
	client := &http.Client{}
	log.Printf("Try Exit Target Server Monitor %s\n", ts.Addr)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/exit", ts.Addr), strings.NewReader("exit"))
	if err != nil {
		log.Println(err)
		return false
	}
	resp, http_err := client.Do(req)
	if http_err != nil {
		log.Println("Exit Target Server Failed: ", http_err)
		return false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "success" {
		return false
	}
	defer resp.Body.Close()
	return true
}

type ServerStatus struct {
	OneSecondData    OneSecondData             `json:"one_second_data"`
	Status           int                       `json:"server_status"`
	SlaveNum         int                       `json:"slave_num"`
	TargetServerData TargetServerOneSecondData `json:"target_server_data"`
	TestDataSum      TestDataSum               `json:"test_data_sum"`
}

type TestData struct {
	TargetQps         float64 `json:"target-qps"`
	IncreasePerSecond float64 `json:"increase-per-second"`
	Args		  []string `json:"args"`
}

type SlaveReportData struct {
	Port          int    `json:"port"`
	TimeStampInMs int64  `json:"time_stamp"`
}

type Slave struct {
	Status         int // 0:Creadted 1:Ready 2:Running 3:Lost 4:Unkown(No report between 1 - 5s)
	TimeGapInMs    int //master time - slave time
	LastSecondData OneSecondData
	Addr         string
}

func (s *Slave) StartTest(target_qps float64, increase_step float64,args []string) bool {
	var test_data TestData
	test_data.TargetQps = target_qps
	test_data.IncreasePerSecond = increase_step
	json_str, json_err := json.Marshal(test_data)
	if json_err != nil {
		log.Println(json_err)
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/start", s.Addr), strings.NewReader(string(json_str)))
	if err != nil {
		log.Println(err)
		return false
	}
	resp, err_req_slave_start := client.Do(req)
	if err_req_slave_start != nil {
		log.Println("request slave failed")
		log.Println(err_req_slave_start)
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "success" {
		return false
	}
	return true
}

func (s *Slave) GetSlaveData() (ServerStatus, bool) {
	var slave_data ServerStatus
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/get_second_data", s.Addr), strings.NewReader(""))
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return slave_data, false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	err_json := json.Unmarshal(body, &slave_data)
	if err_json != nil {
		log.Println(err)
		return slave_data, false
	}
	defer resp.Body.Close()
	return slave_data, true
}

func (s *Slave) StopTest() bool {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/stop", s.Addr), strings.NewReader("stop"))
	if err != nil {
		log.Println(err)
		return false
	}
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "success" {
		return false
	}
	defer resp.Body.Close()
	return true
}

func (s *Slave) Exit() bool {
	client := &http.Client{}
	log.Printf("Try Exit slave %s\n", s.Addr)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/exit", s.Addr), strings.NewReader("exit"))
	if err != nil {
		log.Println(err)
		return false
	}
	resp, http_err := client.Do(req)
	if http_err != nil {
		log.Println("Exit Slave Failed: ", http_err)
		return false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "success" {
		return false
	}
	defer resp.Body.Close()
	return true
}

type ArcheryHttpServer struct {
	Slaves           []Slave
	HttpServerStatus int //0:Stop 1:Running 2:Distribute
	Distribute       bool
	Mode           int //0:single 1:master 2:slave
	Archery             Archery
	MonitorServer    bool
	TargetServer     TargetServer
}

func (ahs *ArcheryHttpServer) VerifySlave(slave Slave) {
	_, is_ok := slave.GetSlaveData()
	if is_ok {
		ahs.Slaves = append(ahs.Slaves, slave)
		log.Printf("Add slave %s\n", slave.Addr)
	}
}

func (ahs *ArcheryHttpServer) VerifyTargetServer(target_server TargetServer) {
	_, is_ok := target_server.GetTargetServerData()
	if is_ok {
		ahs.TargetServer = target_server
		log.Printf("Add target server %s\n", ahs.TargetServer.Addr)
	}
}

func (ahs *ArcheryHttpServer) SlaveReport(w http.ResponseWriter, r *http.Request) {
	s, _ := ioutil.ReadAll(r.Body)
	log.Println(string(s))
	var slave_report SlaveReportData
	err := json.Unmarshal(s, &slave_report)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}
	var slave Slave
	slave.Addr = fmt.Sprintf("%s:%d", strings.Split(r.RemoteAddr, ":")[0], slave_report.Port)
	w.WriteHeader(200)
	w.Write([]byte("success"))
	go func() {
		time.Sleep(time.Second)
		ahs.VerifySlave(slave)
	}()
}

func (ahs *ArcheryHttpServer) SlaveReportExit(w http.ResponseWriter, r *http.Request) {
	s, _ := ioutil.ReadAll(r.Body)
	log.Println(string(s))
	var slave_report SlaveReportData
	err := json.Unmarshal(s, &slave_report)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}
	for idx := range ahs.Slaves {
		if ahs.Slaves[idx].Addr == fmt.Sprintf("%s:%d", strings.Split(r.RemoteAddr, ":")[0], slave_report.Port) {
			ahs.Slaves = append(ahs.Slaves[:idx], ahs.Slaves[idx+1:]...)
		}
	}
	w.WriteHeader(200)
	w.Write([]byte("success"))
	log.Println("remove slave: ", fmt.Sprintf("%s:%s", strings.Split(r.RemoteAddr, ":")[0], slave_report.Port))
}

func (ahs *ArcheryHttpServer) TargetServerReportExit(w http.ResponseWriter, r *http.Request) {
	s, _ := ioutil.ReadAll(r.Body)
	log.Println(string(s))
	var ts TargetServer
	err := json.Unmarshal(s, &ts)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}
	if ahs.TargetServer.Addr == fmt.Sprintf("%s:%s", strings.Split(r.RemoteAddr, ":")[0], strings.Split(ts.Addr, ":")[1]) {
		ahs.MonitorServer = false
		ahs.TargetServer = TargetServer{}
	}
	w.WriteHeader(200)
	w.Write([]byte("success"))
	log.Println("remove target server: ", fmt.Sprintf("%s:%s", strings.Split(r.RemoteAddr, ":")[0], strings.Split(ts.Addr, ":")[1]))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if contents, err := ioutil.ReadFile("static/archery.html"); err == nil {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(200)
		w.Write(contents)
	} else {
		w.WriteHeader(500)
	}
}
func (ahs *ArcheryHttpServer) ExitHandler(w http.ResponseWriter, r *http.Request) {
	s, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
	}
	if string(s) != "exit" {
		w.WriteHeader(500)
	} else {
		w.WriteHeader(200)
		w.Write([]byte("success"))
		log.Println("Catch Exit Req, Exit Now...")
		go func() {
			time.Sleep(time.Second)
			os.Exit(0)
		}()
	}
}

func (ahs *ArcheryHttpServer) StopTestHandler(w http.ResponseWriter, r *http.Request) {
	s, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
	}
	if string(s) != "stop" {
		w.WriteHeader(500)
	} else if ahs.Distribute {
		for slave := range ahs.Slaves {
			ahs.Slaves[slave].StopTest()
		}
		ahs.HttpServerStatus = 0
		w.WriteHeader(200)
		w.Write([]byte("success"))
	} else {
		ahs.Archery.StopLoadTest()
		ahs.HttpServerStatus = 0
		w.WriteHeader(200)
		w.Write([]byte("success"))
	}
}

func (ahs *ArcheryHttpServer) StartTestHandler(w http.ResponseWriter, r *http.Request) {
	if ahs.HttpServerStatus == 1 {
		return
	}
	s, _ := ioutil.ReadAll(r.Body)
	var test_data TestData
	err := json.Unmarshal(s, &test_data)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}
	if ahs.Distribute {
		//分布式处理流程
		slave_num := int64(len(ahs.Slaves))
		//		params := make([][2]int64, slave_num)
		target := test_data.TargetQps / float64(slave_num)
		step := test_data.IncreasePerSecond / float64(slave_num)
		//		for idx := range params {
		//			if idx >= int(target_yushu) {
		//				params[idx][0] = int64(test_data.TargetQps / slave_num)
		//			} else {
		//				params[idx][0] = int64(test_data.TargetQps/slave_num + 1)
		//			}
		//			if idx >= int(step_yushu) {
		//				params[idx][1] = int64(test_data.IncreasePerSecond / slave_num)
		//			} else {
		//				params[idx][1] = int64(test_data.IncreasePerSecond/slave_num + 1)
		//			}
		//		}
		for slave := range ahs.Slaves {
			ahs.Slaves[slave].StartTest(target, step,test_data.Args)
		}
		ahs.HttpServerStatus = 1
		w.WriteHeader(200)
		w.Write([]byte("success"))
	} else {
		//单机部署处理流程
		go ahs.Archery.StartLoadTest(0, test_data.TargetQps, test_data.IncreasePerSecond, -1,test_data.Args)
		ahs.HttpServerStatus = 1
		w.WriteHeader(200)
		w.Write([]byte("success"))
	}
}

func (ahs *ArcheryHttpServer) getSecondData(w http.ResponseWriter, r *http.Request) {
	var target_server_data TargetServerOneSecondData
	var test_data_sum TestDataSum
	if ahs.MonitorServer {
		target_server_data, _ = ahs.TargetServer.GetTargetServerData()
	}
	if !ahs.Distribute {
		one_second_data := ahs.Archery.GetSecondData(ahs.Mode == 2)
		test_data_sum := ahs.Archery.GetTestDataSum()
		result_struct := ServerStatus{one_second_data, ahs.HttpServerStatus, 0, target_server_data, test_data_sum}
		json_str, err := json.Marshal(result_struct)
		if err != nil {
			fmt.Errorf("Marshal Error %v", err)
		}
		//log.Println(string(json_str))
		w.Write([]byte(string(json_str)))
	} else {
		var req_num_sum, succ_resp_sum, failed_num_sum, resp_time_sum int64
		var raw_data []int
		for slave := range ahs.Slaves {
			slave_data, _ := ahs.Slaves[slave].GetSlaveData()
			req_num_sum += slave_data.OneSecondData.Req
			succ_resp_sum += slave_data.OneSecondData.SuccResp
			failed_num_sum += slave_data.OneSecondData.FailedNum
			raw_data = append(raw_data,slave_data.OneSecondData.RawData...)
			resp_time_sum += slave_data.OneSecondData.AverageCostTime * slave_data.OneSecondData.SuccResp
			test_data_sum.TotalReqNum += slave_data.TestDataSum.TotalReqNum
			test_data_sum.TotalRespNum += slave_data.TestDataSum.TotalRespNum
			test_data_sum.TotalSuccRespTime += slave_data.TestDataSum.TotalSuccRespTime
			test_data_sum.TotalFailedNum += slave_data.TestDataSum.TotalFailedNum
		}
		var result OneSecondData
		if succ_resp_sum != 0 {
			var ninty,ninty_nine,fifty int
			snapshot_len := len(raw_data)
			sort.Ints(raw_data)
			if snapshot_len > 0 {
				ninty = raw_data[int(float64(snapshot_len) * 0.9)]//[int(float64(snapshot_len) * 0.9)]
				ninty_nine = raw_data[int(float64(snapshot_len) * 0.99)]
				fifty = raw_data[snapshot_len/2]
			}
			result = OneSecondData{req_num_sum, succ_resp_sum, resp_time_sum / succ_resp_sum, failed_num_sum, int64(time.Now().Unix()), 0, fifty, ninty, ninty_nine,  0,[]int{}}
		} else {
			result = OneSecondData{req_num_sum, succ_resp_sum, 0, failed_num_sum, int64(time.Now().Unix()), 0, 0, 0, 0, 0,[]int{}}
		}
		result_struct := ServerStatus{result, ahs.HttpServerStatus, len(ahs.Slaves), target_server_data, test_data_sum}
		json_str, err := json.Marshal(result_struct)
		if err != nil {
			fmt.Errorf("Marshal Error %v", err)
		}
		//log.Println(string(json_str))
		w.Write(json_str)
	}
}

func (ahs *ArcheryHttpServer) TargetServerReport(w http.ResponseWriter, r *http.Request) {

	ahs.MonitorServer = true
	s, _ := ioutil.ReadAll(r.Body)
	var target_server_report TargetServer
	err := json.Unmarshal(s, &target_server_report)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}
	var ts TargetServer
	ts.Addr = fmt.Sprintf("%s:%s", strings.Split(r.RemoteAddr, ":")[0], strings.Split(target_server_report.Addr, ":")[1])
	w.WriteHeader(200)
	w.Write([]byte("success"))
	go func() {
		time.Sleep(time.Second)
		go ahs.VerifyTargetServer(ts)
	}()
}

//读取本机/proc/stat以及/proc/meminfo，返回上一秒的cpu使用率和内存使用量
func (ahs *ArcheryHttpServer) GetTargetServerData(w http.ResponseWriter, r *http.Request) {
	cpu_data, err_cpu := ioutil.ReadFile("/proc/stat")
	if err_cpu != nil {
		w.WriteHeader(500)
		log.Println(err_cpu)
		return
	}
	mem_data, err_mem := ioutil.ReadFile("/proc/meminfo")
	if err_mem != nil {
		w.WriteHeader(500)
		log.Println(err_mem)
		return
	}
	cpu_info := strings.Fields(string(cpu_data))
	//log.Println(cpu_info)
	var total_cpu int
	for i := 1; i < 10; i++ {
		detail, _ := strconv.Atoi(cpu_info[i])
		total_cpu += detail
	}
	idle_cpu, _ := strconv.Atoi(cpu_info[4])
	mem_info := strings.Split(string(mem_data), "\n")
	mem_line := make(map[string]int)
	for _,line := range mem_info {
		if line == "" {
			break
		}
		line_data := strings.Fields(line)
		mem_line[line_data[0]],_ = strconv.Atoi(line_data[1])
	}
	total_mem := mem_line["MemTotal:"]
	var avai_mem int
	avai_mem = mem_line["MemFree:"] + mem_line["Buffers:"] + mem_line["Cached:"]
	var server_data TargetServerOneSecondData
	server_data.LastSecondIdleCpu = idle_cpu
	server_data.LastSecondTotalCpu = total_cpu
	server_data.MemoryUsage10000 = 10000 - (10000 * avai_mem / total_mem)
	server_data.TimeStampInSec = int(time.Now().Unix())
	json_str, err := json.Marshal(server_data)
	if err != nil {
		fmt.Errorf("Marshal Error %v", err)
	}
	//log.Println(string(json_str))
	w.Write([]byte(string(json_str)))
}
