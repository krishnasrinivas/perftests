package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"

	"time"

	"github.com/minio/minio-go"
)

func usage() {
	fmt.Println("perf <object-size-in-MB> <parallel-upload-count>")
	fmt.Println(`Run "echo 3 > /proc/sys/vm/drop_caches" before running the test so that there is no cache-effect.`)
	os.Exit(0)
}

func performanceTest(client *minio.Core, bucket, objectPrefix string, objSize int64, threadCount int) (bandwidth float64, objsPerSec float64, delta float64) {
	var wg = &sync.WaitGroup{}
	t1 := time.Now()
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Start all the goroutines at the same time
			o, _, err := client.GetObject(bucket, fmt.Sprintf("%s.%d", objectPrefix, i), minio.RequestHeaders{})
			if err != nil {
				fmt.Println(err)
			}
			_, err = io.CopyN(ioutil.Discard, o, objSize)
			if err != nil {
				fmt.Println(err)
			}
		}(i)
	}
	wg.Wait() // Wait till all go routines finish
	delta = time.Since(t1).Seconds()
	bandwidth = float64(objSize*int64(threadCount)) / delta / 1024 / 1024 // in MBps
	objsPerSec = float64(threadCount) / delta
	return bandwidth, objsPerSec, delta
}

func main() {
	bucket := "testbucket"
	objectPrefix := "testobject"
	if len(os.Args) != 3 {
		usage()
	}

	objSize, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Print(err)
		usage()
	}
	objSize = objSize * 1024 * 1024
	threadCount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Print(err)
		usage()
	}
	client, err := minio.NewCore(os.Getenv("MINIO_ENDPOINT"), os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), false)
	if err != nil {
		log.Fatal(err)
	}

	client.MakeBucket(bucket, "") // Ignore "bucket-exists" error

	bandwidth, objsPerSec, delta := performanceTest(client, bucket, objectPrefix, int64(objSize), threadCount)
	t := struct {
		ObjSize     int64
		ThreadCount int
		Delta       float64
		Bandwidth   float64
		ObjsPerSec  float64
	}{
		int64(objSize), threadCount, delta, bandwidth, objsPerSec,
	}
	b, err := json.Marshal(t)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
