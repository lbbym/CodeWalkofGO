package main

import (
	"time"
	"net/http"
	"log"
)

const (
	numPollers		= 2
	pollInterval	= 60 * time.Second
	statusInterval	= 10 * time.Second
	errTimeout		= 10 * time.Second
)

var urls = []string{
	"http://www.google.com/",
	"http://golang.org/",
	"http://blog.golang.org",
	"https://www.baidu.com/",
}

//State 表示一个 URL 最后的已知状态
type State struct {
	url		string
	status	string
}

//StateMonitor 维护一个映射，它存储了 URL 被轮询的状态， 并每隔 updateInterval
//纳秒打印出其当前的状态。它向资源状态的接收者返回一个 chan State。
func StateMonitor(updateInterval time.Duration) chan<-State {
	updates := make(chan State)
	urlStatus := make(map[string]string)
	ticker := time.NewTicker(updateInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				logState(urlStatus)
			case s := <-updates:
				urlStatus[s.url] = s.status
			}
		}
	}()
	return updates
}

//logState 打印出一个状态映射。
func logState(s map[string]string)  {
	log.Println("Current state: ")
	for k, v := range s {
		log.Println("%s %s", k, v)
	}
}

//Resource 表示一个被此程序轮询的 HTTP URL
type Resource struct {
	url		 string
	errCount int
}

//Poll 为 url 执行一个 HTTP HEAD 请求，并返回 HTTP 的状态字符串或一个错误字符串
func (r *Resource) Poll() string {
	resp, err := http.Head(r.url)
	if err != nil {
		log.Println("ERROR", r.url, err)
		r.errCount++
		return err.Error()
	}
	r.errCount = 0
	return resp.Status
}

//Sleep 在将 Resource 发送到 done 之前休眠一段适当时间
func (r *Resource) Sleep(done chan <- *Resource)  {
	time.Sleep(pollInterval + errTimeout*time.Duration(r.errCount))
	done <- r
}

func Poller(in <-chan *Resource, out chan<- *Resource, status chan<- State)  {
	for r := range in {
		s := r.Poll()
		status <- State{r.url, s}
		out <- r
	}
}

func main()  {
	//创建我们的输入输出信道。
	pending, complete := make(chan *Resource), make(chan *Resource)

	//启动 StateMonitor
	status := StateMonitor(statusInterval)

	//启动一些 Poller Go 程
	for i := 0; i < numPollers; i++ {
		go Poller(pending, complete, status)
	}

	//将一些 Resource 发送至 Pending 序列
	go func() {
			for _, url := range urls{
				pending <- &Resource{url,0}
			}
	}()

	for r := range complete{
		go r.Sleep(pending)
	}
}