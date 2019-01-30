package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// сюда писать код
func ExecutePipeline(jobs ...job) {
	var (
		in  = make(chan interface{})
		out = make(chan interface{})
		wg  = &sync.WaitGroup{}
	)
	for _, someJob := range jobs {
		wg.Add(1)
		go func(in, out chan interface{}, someJob job) {
			defer wg.Done()
			someJob(in, out)
			//close(out)
		}(in, out, someJob)
		in = out
		out = make(chan interface{})
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var (
		quotaMd5Ch  = make(chan struct{}, 1)
		md5Ch       = make(chan string)
		crc32Ch1    = make(chan string)
		crc32Ch2    = make(chan string)
		dataCounter uint32

		md5Proc = func(data string, quotaMd5Ch chan struct{}, md5Ch chan string) {
			quotaMd5Ch <- struct{}{}
			signMd5 := DataSignerMd5(data)
			//fmt.Printf("%d SingleHash signMd5(data) %s\n", dataCounter, signMd5)
			md5Ch <- signMd5
			<-quotaMd5Ch
		}
		crc32Proc = func(data string, crc32Ch chan string) {
			signCrc32 := DataSignerCrc32(data)
			//fmt.Printf("%d SingleHash crc32(data) %s\n", dataCounter, signCrc32)
			crc32Ch <- signCrc32
		}
		responseBuilder = func(crc32Ch1, crc32Ch2 chan string) {
			result := <-crc32Ch1 + "~" + <-crc32Ch2
			//fmt.Printf("%d SingleHash result %s\n", dataCounter, result)
			out <- result
		}
	)
	for dataRaw := range in {
		data := strconv.Itoa(dataRaw.(int))
		fmt.Printf("->%d SingleHash data %s\n", dataCounter, data)
		go md5Proc(data, quotaMd5Ch, md5Ch)
		go crc32Proc(data, crc32Ch1)
		go crc32Proc(<-md5Ch, crc32Ch2)
		go responseBuilder(crc32Ch1, crc32Ch2)
		dataCounter++
		md5Ch = make(chan string)
		crc32Ch1 = make(chan string)
		crc32Ch2 = make(chan string)
	}
}

func MultiHash(in, out chan interface{}) {
	const countHash = 6
	var (
		chs [countHash]chan string

		initChs = func(chs [countHash]chan string) {
			for i := range chs {
				chs[i] = make(chan string)
			}
		}
		crc32Proc = func(thRaw int, data string, crc32Ch chan string) {
			th := strconv.Itoa(thRaw)
			signCrc32 := DataSignerCrc32(th + data)
			//fmt.Printf("%s MultiHash: crc32(th+step1)) %d %s\n", data, thRaw, signCrc32)
			crc32Ch <- signCrc32
		}
		responseBuilder = func(chs [countHash]chan string) {
			var result string
			for i := range chs {
				result += <-chs[i]
			}
			//fmt.Printf("MultiHash result: %s\n\n", result)
			out <- result
		}
	)
	initChs(chs)
	for dataRaw := range in {
		data := dataRaw.(string)
		for i := 0; i < countHash; i++ {
			go crc32Proc(i, data, chs[i])
		}
		go responseBuilder(chs)
		initChs(chs)
	}
}

func CombineResults(in, out chan interface{}) {
	var hashes []string
	for dataRaw := range in {
		data := dataRaw.(string)
		hashes = append(hashes, data)
	}
	sort.Strings(hashes)
	result := strings.Join(hashes, "_")
	//fmt.Printf("CombineResults: %s", result)
	out <- result
}
