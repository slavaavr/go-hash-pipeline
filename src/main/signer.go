package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

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
			close(out)
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
		wg          = &sync.WaitGroup{}
		dataCounter int

		md5Proc = func(index int, data string, quotaMd5Ch chan struct{}, md5Ch chan string) {
			quotaMd5Ch <- struct{}{}
			md5Ch <- DataSignerMd5(data)
			<-quotaMd5Ch
		}
		crc32Proc = func(index int, data string, crc32Ch chan string) {
			crc32Ch <- DataSignerCrc32(data)
		}
		responseBuilder = func(index int, crc32Ch1, crc32Ch2 chan string) {
			defer wg.Done()
			out <- <-crc32Ch1 + "~" + <-crc32Ch2
		}
	)
	for dataRaw := range in {
		data := strconv.Itoa(dataRaw.(int))
		fmt.Printf("->%d SingleHash data %s\n", dataCounter, data)
		wg.Add(1)
		go md5Proc(dataCounter, data, quotaMd5Ch, md5Ch)
		go crc32Proc(dataCounter, data, crc32Ch1)
		go crc32Proc(dataCounter, <-md5Ch, crc32Ch2)
		go responseBuilder(dataCounter, crc32Ch1, crc32Ch2)
		dataCounter++
		md5Ch = make(chan string)
		crc32Ch1 = make(chan string)
		crc32Ch2 = make(chan string)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	const countHash = 6
	var (
		chs [countHash]chan string
		wg  = &sync.WaitGroup{}

		initChs = func(chs *[countHash]chan string) {
			for i := range chs {
				chs[i] = make(chan string)
			}
		}
		crc32Proc = func(thRaw int, data string, crc32Ch chan string) {
			th := strconv.Itoa(thRaw)
			crc32Ch <- DataSignerCrc32(th + data)
		}
		responseBuilder = func(chs [countHash]chan string) {
			defer wg.Done()
			var result string
			for i := range chs {
				result += <-chs[i]
			}
			out <- result
		}
	)
	initChs(&chs)
	for dataRaw := range in {
		data := dataRaw.(string)
		for i := 0; i < countHash; i++ {
			go crc32Proc(i, data, chs[i])
		}
		wg.Add(1)
		go responseBuilder(chs)
		initChs(&chs)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var hashes []string
	for dataRaw := range in {
		data := dataRaw.(string)
		hashes = append(hashes, data)
	}
	sort.Strings(hashes)
	result := strings.Join(hashes, "_")
	fmt.Printf("CombineResults: %s", result)
	out <- result
}
