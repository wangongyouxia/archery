package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

type StartFlag struct {
	port        int
	mode        string
	master_addr string
}

func NewArcheryHttpServer() ArcheryHttpServer {
	var ahs ArcheryHttpServer
	ahs.Archeries = make(map[string](*Archery))
	work_list := ahs.Task.LoadWorkList()
	for _,work := range work_list {
		var archery Archery
		archery.work = work.WorkFunc
		archery.ratio = work.Ratio
		ahs.Archeries[work.Title] = &archery
	}
	return ahs
}

func SlaveExitHandler(master_addr string, json_str string) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, exit slave...")
		req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/slave_report_exit", master_addr), strings.NewReader(json_str))
		client := &http.Client{}
		client.Do(req)
		os.Exit(0)
	}()
}

func MasterExitHandler(ahs *ArcheryHttpServer){
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, Exit Slaves and Server Monitor...")
		for idx := range (*ahs).Slaves{
			ahs.Slaves[idx].Exit()
		}
		if ahs.MonitorServer{
			ahs.TargetServer.ExitTargetServer()
		}
		os.Exit(0)
	}()
}


func MonitorExitHandler(master_addr string, json_str string) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, exit server monitor...")
		req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/target_server_report_exit", master_addr), strings.NewReader(json_str))
		client := &http.Client{}
		client.Do(req)
		os.Exit(0)
	}()
}

func SingleExitHandler(ahs *ArcheryHttpServer){
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, exit server monitor...")
		if ahs.MonitorServer{
			ahs.TargetServer.ExitTargetServer()
		}
		os.Exit(0)
	}()
}

//单机部署函数，绑定函数，注册信号处理函数
func StartSingle(port int) {
	ahs := NewArcheryHttpServer()
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/get_second_data", ahs.getSecondData)
	http.HandleFunc("/slave_report", ahs.SlaveReport)
	http.HandleFunc("/target_server_report", ahs.TargetServerReport)
	http.HandleFunc("/target_server_report_exit", ahs.TargetServerReportExit)
	http.HandleFunc("/start", ahs.StartTestHandler)
	http.HandleFunc("/stop", ahs.StopTestHandler)
	http.Handle("/static/", http.FileServer(http.Dir("./")))
	SingleExitHandler(&ahs)
	http.ListenAndServe(fmt.Sprintf(":%d",port), nil)
}

//启动服务器资源监控
func StartMonitor(master_addr string) {
	listener, _ := net.Listen("tcp", ":0")
	port := listener.Addr().(*net.TCPAddr).Port
	now := int64(time.Now().UnixNano() / 1e6)
	var target_server TargetServer
	target_server.Addr = fmt.Sprintf(":%d",port)
	target_server.TimeStampInMs = now
	json_str, err := json.Marshal(target_server)
	if err != nil {
		fmt.Errorf("Marshal Error %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/target_server_report", master_addr), strings.NewReader(string(json_str)))
	if err != nil {
		log.Println(err)
		return
	}
	client := &http.Client{}
	var resp *http.Response
	var err_http error
	for {
		resp, err_http = client.Do(req)
		if err_http != nil {
			log.Println("Try connecting master error: ", err_http)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	resp.Body.Close()
	var ahs ArcheryHttpServer
	var httpd http.Server
	http.HandleFunc("/get_server_second_data", ahs.GetTargetServerData)
	http.HandleFunc("/exit", ahs.ExitHandler)
	MonitorExitHandler(master_addr, string(json_str))
	httpd.Serve(listener)
}

//启动分布式部署的master
func StartMaster(port int) {
	ahs := NewArcheryHttpServer()
	ahs.Distribute = true
	http.HandleFunc("/", IndexHandler) //返回前端页面
	http.HandleFunc("/get_second_data", ahs.getSecondData) //获取每秒数据，会同步请求每个slave，并汇总返回
	http.HandleFunc("/slave_report", ahs.SlaveReport) //slave启动上报，需要记录该slave信息
	http.HandleFunc("/slave_report_exit", ahs.SlaveReportExit) //slave退出上报，需要删除该slave信息
	http.HandleFunc("/target_server_report", ahs.TargetServerReport)
	http.HandleFunc("/target_server_report_exit", ahs.TargetServerReportExit)
	http.HandleFunc("/start", ahs.StartTestHandler) //开始压测处理函数
	http.HandleFunc("/stop", ahs.StopTestHandler) //停止压测处理函数
	http.HandleFunc("/get_server_second_data", ahs.GetTargetServerData)
	http.Handle("/static/", http.FileServer(http.Dir("./")))
	MasterExitHandler(&ahs)
	http.ListenAndServe(fmt.Sprintf(":%d",port), nil)
}

//启动分布式部署的slave
func StartSlave(master_addr string) {
	listener, _ := net.Listen("tcp", ":0")
	port := listener.Addr().(*net.TCPAddr).Port
	//log.Printf("%d",port)
	now := int64(time.Now().UnixNano() / 1e6)
	var report_data SlaveReportData
	report_data.Port = port
	report_data.TimeStampInMs = now
	json_str, err := json.Marshal(report_data)
	if err != nil {
		fmt.Errorf("Marshal Error %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/slave_report", master_addr), strings.NewReader(string(json_str)))
	if err != nil {
		log.Println(err)
		return
	}
	client := &http.Client{}
	var resp *http.Response
	var err_http error
	for {
		resp, err_http = client.Do(req)
		if err_http != nil {
			log.Println("Try connecting master error: ", err_http)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	resp.Body.Close()
	ahs := NewArcheryHttpServer()
	ahs.Mode = 2
	var httpd http.Server
	http.HandleFunc("/get_second_data", ahs.getSecondData) //获取每秒数据函数
	http.HandleFunc("/start", ahs.StartTestHandler) //开始压测处理函数
	http.HandleFunc("/stop", ahs.StopTestHandler) //结束压测处理函数
	http.HandleFunc("/exit", ahs.ExitHandler) //退出进程函数
	SlaveExitHandler(master_addr, string(json_str))
	httpd.Serve(listener)
}


//主函数，根据flag，选择不同的模式启动
func main() {
	config := StartFlag{}
	flag.StringVar(&config.mode, "mode", "single", "specify start mode, valid value: [single,slave,master,monitor]")
	flag.IntVar(&config.port, "port", 8018, "specify listen port(for master or single mode)")
	flag.StringVar(&config.master_addr, "master_addr", "127.0.0.1:8018", "specify the master address(ip or hostname)")
	flag.Parse()
	if (config.mode == "single") {
		StartSingle(config.port)
	} else if (config.mode == "master") {
		StartMaster(config.port)
	} else if (config.mode == "slave") {
		StartSlave(config.master_addr)
	} else if (config.mode == "monitor") {
		StartMonitor(config.master_addr)
	} else {
		fmt.Println("mode not valid, run -h for help")
	}
}
