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
	work_list,task := ahs.Task.LoadWorkList()
	for _, work := range work_list {
		var archery Archery
		archery.work = work.WorkFunc
		archery.ratio = work.Ratio
		archery.task = task
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

func MasterExitHandler(ahs *ArcheryHttpServer) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, Exit Slaves and Server Monitor...")
		for idx := range (*ahs).Slaves {
			ahs.Slaves[idx].Exit()
		}
		if ahs.MonitorServer {
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

func SingleExitHandler(ahs *ArcheryHttpServer) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		log.Println("Ctrl+C pressed in Terminal, exit server monitor...")
		if ahs.MonitorServer {
			ahs.TargetServer.ExitTargetServer()
		}
		os.Exit(0)
	}()
}

//????????????????????????????????????????????????????????????
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
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

//???????????????????????????
func StartMonitor(master_addr string) {
	listener, _ := net.Listen("tcp", ":0")
	port := listener.Addr().(*net.TCPAddr).Port
	now := int64(time.Now().UnixNano() / 1e6)
	var target_server TargetServer
	target_server.Addr = fmt.Sprintf(":%d", port)
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
	err = httpd.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}

//????????????????????????master
func StartMaster(port int) {
	ahs := NewArcheryHttpServer()
	ahs.Distribute = true
	http.HandleFunc("/", IndexHandler)                         //??????????????????
	http.HandleFunc("/get_second_data", ahs.getSecondData)     //??????????????????????????????????????????slave??????????????????
	http.HandleFunc("/slave_report", ahs.SlaveReport)          //slave??????????????????????????????slave??????
	http.HandleFunc("/slave_report_exit", ahs.SlaveReportExit) //slave??????????????????????????????slave??????
	http.HandleFunc("/target_server_report", ahs.TargetServerReport)
	http.HandleFunc("/target_server_report_exit", ahs.TargetServerReportExit)
	http.HandleFunc("/start", ahs.StartTestHandler) //????????????????????????
	http.HandleFunc("/stop", ahs.StopTestHandler)   //????????????????????????
	http.HandleFunc("/get_server_second_data", ahs.GetTargetServerData)
	http.Handle("/static/", http.FileServer(http.Dir("./")))
	MasterExitHandler(&ahs)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

//????????????????????????slave
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
	http.HandleFunc("/get_second_data", ahs.getSecondData) //????????????????????????
	http.HandleFunc("/start", ahs.StartTestHandler)        //????????????????????????
	http.HandleFunc("/stop", ahs.StopTestHandler)          //????????????????????????
	http.HandleFunc("/exit", ahs.ExitHandler)              //??????????????????
	SlaveExitHandler(master_addr, string(json_str))
	err = httpd.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}

//??????????????????flag??????????????????????????????
func main() {
	config := StartFlag{}
	flag.StringVar(&config.mode, "mode", "single", "specify start mode, valid value: [single,slave,master,monitor]")
	flag.IntVar(&config.port, "port", 8018, "specify listen port(for master or single mode)")
	flag.StringVar(&config.master_addr, "master_addr", "127.0.0.1:8018", "specify the master address(ip or hostname)")
	flag.Parse()
	if config.mode == "single" {
		StartSingle(config.port)
	} else if config.mode == "master" {
		StartMaster(config.port)
	} else if config.mode == "slave" {
		StartSlave(config.master_addr)
	} else if config.mode == "monitor" {
		StartMonitor(config.master_addr)
	} else {
		fmt.Println("mode not valid, run -h for help")
	}
}
