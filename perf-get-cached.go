package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	minio "github.com/minio/minio-go"
)

type transferUnit struct {
	s int64
	t time.Duration
}

func usage() {
	fmt.Println(`
perf-get-cached <object-size-in-MB> <thread-count> <time-in-secs>

This test measures download bandwidth when the object is cached on the server side backend FS. i.e the test does not hit the backend disk.

Ex. perf-get-cached 1024 20 60

This example creates a bucket "testbucket" and uploads "testobject" of size 1024MB.
The test then spawns 20 threads, each thread downloads "testobject" in a loop. The test runs for 60 seconds.
Note that each thread downloads the object in full.

The test is called "cached" test as the "testobject" is cached on the server side backend FS and hence the test
does not hit the disk on the server side.
`)
	os.Exit(0)
}

func downloadInLoop(client *minio.Core, f *os.File, size int64, bucket, objectPrefix string, threadNum int, ch chan<- transferUnit) {
	for i := 0; ; i++ {
		t := time.Now()
		r, _, err := client.GetObject(bucket, "testobject", minio.RequestHeaders{})
		if err != nil {
			fmt.Println(err)
		}
		io.Copy(ioutil.Discard, r)
		r.Close()
		ch <- transferUnit{size, time.Since(t)}
	}
}

func collectStats(endAfter time.Duration, ch <-chan transferUnit) {
	endCh := time.After(endAfter)
	var totalSize int64
	for {
		select {
		case entry := <-ch:
			totalSize += entry.s
		case <-endCh:
			fmt.Println("bandwidth", float64(totalSize)/endAfter.Seconds()/1024/1024, "MBps")
			return
		}
	}
}

func main() {
	bucket := "testbucket"
	objectPrefix := "testobject"
	if len(os.Args) != 4 {
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
	_, err = client.PutObject(bucket, objectPrefix, int64(objSize), io.NewSectionReader(f, 0, int64(objSize)), nil, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan transferUnit)

	for i := 0; i < threadCount; i++ {
		go downloadInLoop(client, f, int64(objSize), bucket, objectPrefix, i, ch)
	}

	collectStats(time.Duration(int64(timeToRun)*int64(time.Second)), ch)
}
