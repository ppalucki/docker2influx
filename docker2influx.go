/*
DOCKER_LOG="/home/ppalucki/work/go15/src/github.com/docker/docker.log" DOCKER_HOST="tcp://127.0.0.1:8080" TAGS=",version=v193,net=none" DOCKER_PID=`sudo lsof -t -sTCP:LISTEN -i :8080`  ./docker2influx
*/
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/hpcloud/tail"
	"github.com/shirou/gopsutil/process"
)

const INFLUXDB_UDP = "127.0.0.1:4444"

// dockerData gather date from docker daemon directly (using DOCKER_HOST)
// and from proc/DOCKER_PID/status and publish those to conn
func dockerData(tags string, conn net.Conn) {
	if os.Getenv("DOCKER_HOST") == "" {
		log.Fatal(`please provide eg. DOCKER_HOST="tcp://127.0.0.1:8080"`)
	}

	// pid
	pidS := os.Getenv("DOCKER_PID")
	if pidS == "" {
		log.Fatal("cannot find docker deamon - please provide DOCKER_PID - try DOCKER_PID=`sudo lsof -t -sTCP:LISTEN -i :8080`")
	}
	pid, err := strconv.Atoi(pidS)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("docker pid=%d", pid)

	// docker client
	dockerClient, _ := docker.NewClientFromEnv()
	for {
		p, err := process.NewProcess(int32(pid))
		// threads
		threads, err := p.NumThreads()
		if err != nil {
			log.Fatal(err)
		}
		mi, err := p.MemoryInfo()
		if err != nil {
			log.Fatal(err)
		}
		rss := mi.RSS
		vms := mi.VMS

		// docker info
		info, err := dockerClient.Info()
		// log.Printf("info=%#v", info)

		output := fmt.Sprintf("docker,driver=%s%s containers=%di,goroutines=%di,images=%di,threads=%di,vmsize=%di,rss=%di",
			info.Get("Driver"), tags, info.GetInt("Containers"), info.GetInt("NGoroutines"), info.GetInt("Images"), threads, vms, rss)

		n, err := conn.Write([]byte(output))
		if err != nil {
			log.Fatal(err)
		}
		log.Println(n, output)
		time.Sleep(1 * time.Second)
	}
}

/*
SCHED 0ms: gomaxprocs=1 idleprocs=0 threads=2 spinningthreads=0 idlethreads=0 runqueue=0 [1]

provide
GODEBUG=schedtrace=1000
*/
func schedData(tags string, dockerLog string, conn net.Conn) {

	t, err := tail.TailFile(dockerLog, tail.Config{Follow: true, Location: &tail.SeekInfo{Offset: 0, Whence: 2}})
	if err != nil {
		log.Fatal(err)
	}
	for line := range t.Lines {
		// when = line.Time
		text := line.Text
		var output string
		if strings.Contains(text, "SCHED") {
			data := strings.Split(text, " ")

			//SCHED 0ms: 2.gomaxprocs=1 idleprocs=0 threads=2 spinningthreads=0 idlethreads=0 runqueue=0 [1]
			gomaxprocs, err := strconv.Atoi(strings.Split(data[2], "=")[1])
			if err != nil {
				log.Fatal(err)
			}
			idleproces, err := strconv.Atoi(strings.Split(data[3], "=")[1])
			if err != nil {
				log.Fatal(err)
			}
			threads, err := strconv.Atoi(strings.Split(data[4], "=")[1])
			if err != nil {
				log.Fatal(err)
			}
			spinningthreads, err := strconv.Atoi(strings.Split(data[5], "=")[1])
			if err != nil {
				log.Fatal(err)
			}
			idlethreads, err := strconv.Atoi(strings.Split(data[6], "=")[1])
			if err != nil {
				log.Fatal(err)
			}
			runqueue, err := strconv.Atoi(strings.Split(data[7], "=")[1])
			if err != nil {
				log.Fatal(err)
			}

			output = fmt.Sprintf("sched%s gomaxprocs=%di,idleprocs=%di,threads=%di,spinningthreads=%di,idlethreads=%di,runqueue=%di",
				tags, gomaxprocs, idleproces, threads, spinningthreads, idlethreads, runqueue)

		}

		if strings.Contains("ERRO[", text) {
			fmt.Println("ERROR:", text)
			output = fmt.Sprintf(`events,kind=error%s message="%q",count=1,tags=error`, tags, text)
		}

		if strings.Contains("WARN[", text) {
			fmt.Println("WARNING:", text)
			output = fmt.Sprintf(`events,kind=warn%s message="%q",count=1,tags=warn`, tags, text)
		}

		n, err := conn.Write([]byte(output))
		if err != nil {
			log.Fatal(err)
		}
		log.Println(n, output)
	}
}

func main() {

	// tags
	tags := os.Getenv("TAGS")

	// ------ influx connection
	// TODO: checkout replacing with officiial client
	// client "github.com/influxdb/influxdb/client/v2"
	// influxUrl, _ := url.Parse("http://localhost:8086")
	// influx := client.NewClient(client.Config{
	// 	URL:      influxUrl,
	// 	Username: "root",
	// 	Password: "root",
	// })
	// udp socket (or use infludb client)
	log.Println("connecting influxdb:", INFLUXDB_UDP)
	conn, err := net.Dial("udp", INFLUXDB_UDP)
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// ------ docker daemon exposed data (proc/status and info)
	go dockerData(tags, conn)

	// sched data
	// docker_log
	dockerLog := os.Getenv("DOCKER_LOG")
	if dockerLog != "" {
		log.Println("DOCKER_LOG at", dockerLog)
		go schedData(tags, dockerLog, conn)
	}

	// block forever
	select {}

}
