package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	minio "github.com/minio/minio-go"
)

type transferUnit struct {
	s int64
	t time.Duration
}

func usage() {
	fmt.Println(`
perf-put-uncached <comma-seperated-object-sizes-in-MB> <thread-count> <time-in-secs>

This test is same as perf-put-cached but it is run for a longer duration (duration is specified as an argument to the test)
i.e since the uploads are done for a longer duration the objects get written to the disk during the test.

Ex. perf-put-cached 1024 10 60

This test spawns 10 threads and each thread uploads an object of size 1024 MB. The test runs for 60 seconds.
The tests fill up the server side backend FS memory cache and soon start hitting the disks.

`)
	fmt.Println("perf 2,8,32,128 20 180 ---> runs 4 tests, each test with different object size, with 20 threads for 180 seconds")
	os.Exit(0)
}

func uploadInLoop(client *minio.Core, f *os.File, size int64, bucket, objectPrefix string, threadNum int, ch chan<- transferUnit) {
	for i := 0; ; i++ {
		t := time.Now()
		_, err := client.PutObject(bucket, fmt.Sprintf("%s.%d.%d", objectPrefix, threadNum, i), size, io.NewSectionReader(f, 0, size), nil, nil, nil)
		if err != nil {
			fmt.Println(err)
		}
		ch <- transferUnit{size, time.Since(t)}
	}
}

func performanceTest(client *minio.Core, f *os.File, bucket, objectPrefix string, objSize int64, threadCount int, timeToRun time.Duration) (bandwidth float64, objsPerSec float64) {
	ch := make(chan transferUnit)

	for i := 0; i < threadCount; i++ {
		go uploadInLoop(client, f, int64(objSize), bucket, objectPrefix, i, ch)
	}

	endCh := time.After(time.Duration(timeToRun))
	var totalSize int64
	objCount := 0
	for {
		select {
		case entry := <-ch:
			totalSize += entry.s
			objCount++
		case <-endCh:
			bandwidth = float64(totalSize) / timeToRun.Seconds() / 1024 / 1024
			return bandwidth, float64(objCount) / timeToRun.Seconds()
		}
	}
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
	if len(os.Args) != 4 {
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

	timeToRun, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Print(err)
		usage()
	}

	client, err := minio.NewCore(os.Getenv("MINIO_ENDPOINT"), os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), false)
	if err != nil {
		log.Fatal(err)
	}

	client.MakeBucket(bucket, "") // Ignore "bucket-exists" error

	f, err := os.Open("bigfile")
	if err != nil {
		log.Fatal(err)
	}
	for _, objSize := range objSizes {
		removeObjects(bucket)
		bandwidth, objsPerSec := performanceTest(client, f, bucket, objectPrefix, objSize, threadCount, time.Duration(int64(timeToRun)*int64(time.Second)))
		t := struct {
			ObjSize     int64
			ThreadCount int
			Duration    int
			Bandwidth   float64
			ObjsPerSec  float64
		}{
			objSize, threadCount, timeToRun, bandwidth, objsPerSec,
		}
		b, err := json.Marshal(t)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	}
}
