# perftests

## perf-put-cached

perf-put-cached <object-size-in-MB> <parallel-upload-count>

This test measures the upload bandwidth when the uploaded object is cached on the server side. i.e the test does not hit the disk.

Ex. perf-put-cached 1024 10

This test spawns 10 threads and each thread uploads an object of size 1024 MB.

We need to make sure that the server side RAM is larger than 1024*10 (say 32G) so that the uploaded objects gets cached in the RAM and
the tests do not hit the disks during the duration of the tests.

## perf-put-uncached

perf-put-uncached <comma-seperated-object-sizes-in-MB> <thread-count> <time-in-secs>

This test is same as perf-put-cached but it is run for a longer duration (duration is specified as an argument to the test)
i.e since the uploads are done for a longer duration the objects get written to the disk during the test.

Ex. perf-put-cached 1024 10 60

This test spawns 10 threads and each thread uploads an object of size 1024 MB. The test runs for 60 seconds.
The tests fill up the server side backend FS memory cache and soon start hitting the disks.

## perf-get-cached

perf-get-cached <object-size-in-MB> <thread-count> <time-in-secs>

This test measures download bandwidth when the object is cached on the server side backend FS. i.e the test does not hit the backend disk.

Ex. perf-get-cached 1024 20 60

This example creates a bucket "testbucket" and uploads "testobject" of size 1024MB.
The test then spawns 20 threads, each thread downloads "testobject" in a loop. The test runs for 60 seconds.
Note that each thread downloads the object in full.

The test is called "cached" test as the "testobject" is cached on the server side backend FS and hence the test
does not hit the disk on the server side.

## perf-get-uncached

perf-get-uncached <object-size-in-MB> <parallel-download-count>

This test measures download bandwidth when the objects are uncached on the server side backend FS. i.e the test hits the backend disk.

Ex. perf-get-uncached 1024 20

This test spawns 20 threads. Let's call the threads thread.0 thread.1 ... thread.19
Now thread-0 downloads testobject.0, thread-1 downloads testobject.1 ... thread.19 downloads testobject.19

Note that before running the test we need to create testobject.0, testobject.1 ... testobject.19 and run:
"echo 3 > /proc/sys/vm/drop_caches" so that the server-side backend FS does not have any cached data.
