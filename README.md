## docker2influx
Push internal state of docker deamon, process information and golang scheduler to influxdb.

### Installation

```sh
go get github.com/ppalucki/docker2influx
```


### How to use


Start docker daemon with:
```sh
GODEBUG=schedtrace=1000 sudo -E ./dockerb daemon --host :8080 --log-level error --exec-root=`pwd`/tmp/run/docker --graph=`pwd`/tmp/lib/docker 2>&1 |tee docker.log
```

Start collector like this:
```sh
DOCKER_LOG="/home/ppalucki/work/go15/src/github.com/docker/docker.log" DOCKER_HOST="tcp://127.0.0.1:8080" TAGS=",version=v193,net=none" DOCKER_PID=`sudo lsof -t -sTCP:LISTEN -i :8080` docker2influx
```

### Collected data

- containers
- goroutines
- threads
- rss
- vmsize
- gomaxprocs
- idleprocs
- idlethreads
- runqueue
- spinningthreads






