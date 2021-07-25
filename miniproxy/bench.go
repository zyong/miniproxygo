package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// 开200，500，1000并发
func main() {
	times := flag.Int("times", 100, "looop run times")

	urlList := []string{
		"https://www.jd.com",
		"https://www.taobao.com",
		"https://www.163.com",
		"https://www.sina.com.cn",
		"https://www.baidu.com",
		"https://news.baidu.com",
	}

	proxyUrl, _ := url.Parse("http://192.168.3.13:8080")
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	var wg sync.WaitGroup
	var count int32
	for i := 0; i < *times; i++ {
		wg.Add(1)
		go func(i int) {
			for {
				if atomic.LoadInt32(&count) == int32(i) {
					atomic.AddInt32(&count, 1)
					break
				} else {
					time.Sleep(100 * time.Microsecond)
				}
			}
			fmt.Printf("%d exec start\n", i)
			_, err := client.Get(urlList[i%len(urlList)])

			if err != nil {
				fmt.Errorf("%v\n", err)
				return
			}
			// fmt.Printf("code %d\n", res.StatusCode)
			wg.Done()

		}(i)
	}
	wg.Wait()

}
