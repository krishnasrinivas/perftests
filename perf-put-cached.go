package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/minio/minio-go"
)

func usage() {
	fmt.Println(`
perf-put-cached <object-size-in-MB> <parallel-upload-count>

This test measures the upload bandwidth when the uploaded object is cached on the server side. i.e the test does not hit the disk.

Ex. perf-put-cached 1024 10

This test spawns 10 threads and each thread uploads an object of size 1024 MB.

We need to make sure that the server side RAM is larger than 1024*10 (say 32G) so that the uploaded objects gets cached in the RAM and
the tests do not hit the disks during the duration of the tests.
`)
	os.Exit(0)
}

func performanceTest(client *minio.Core, f *os.File, bucket, objectPrefix string, objSize int64, threadCount int) (bandwidth float64, objsPerSec float64, delta float64) {
	ch := make(chan struct{})
	var wg = &sync.WaitGroup{}
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Start all the goroutines at the same time
			<-ch
			_, err := client.PutObject(bucket, fmt.Sprintf("%s.%d", objectPrefix, i), objSize, io.NewSectionReader(f, 0, objSize), nil, nil, nil)
			if err != nil {
				fmt.Println(err)
			}
		}(i)
	}
	t1 := time.Now()
	close(ch)
	wg.Wait() // Wait till all go routines finish
	delta = time.Since(t1).Seconds()
	bandwidth = float64(objSize*int64(threadCount)) / delta / 1024 / 1024 // in MBps
	objsPerSec = float64(threadCount) / delta
	return bandwidth, objsPerSec, delta
}

func removeObjects(bucket string) {
	client, err := minio.New(os.Getenv("MINIO_ENDPOINT"), os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), false)
	if err != nil {
		log.Fatal(err)
	}
	listCh := client.ListObjects(bucket, "", true, nil)
	for entry := range listCh {
		if entry.Err != nil {
			log.Fatal(err)
		}
		err = client.RemoveObject(bucket, entry.Key)
		if entry.Err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	bucket := "testbucket"
	objectPrefix := "testobject"
	if len(os.Args) != 3 {
		usage()
	}

	objSizeStrs := strings.Split(os.Args[1], ",")
	var objSizes []int64
	for _, objSizeStr := range objSizeStrs {
		objSize, err := strconv.Atoi(objSizeStr)
		if err != nil {
			log.Print(err)
			usage()
		}
		objSizes = append(objSizes, int64(objSize*1024*1024))
	}

	threadCount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Print(err)
		usage()
	}
	f, err := os.Open("bigfile")
	if err != nil {
		log.Fatal(err)
	}
	client, err := minio.NewCore(os.Getenv("MINIO_ENDPOINT"), os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), false)
	if err != nil {
		log.Fatal(err)
	}

	client.MakeBucket(bucket, "") // Ignore "bucket-exists" error

	for _, objSize := range objSizes {
		removeObjects(bucket)
		bandwidth, objsPerSec, delta := performanceTest(client, f, bucket, objectPrefix, objSize, threadCount)
		t := struct {
			ObjSize     int64
			ThreadCount int
			Delta       float64
			Bandwidth   float64
			ObjsPerSec  float64
		}{
			objSize, threadCount, delta, bandwidth, objsPerSec,
		}
		b, err := json.Marshal(t)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	}
}
