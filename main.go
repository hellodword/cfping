package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/hellodword/cfping/ping"
	"github.com/panjf2000/ants/v2"
)

func main() {

	cidrPath := flag.String("cidr", "cidr.txt", "path to cidr file")
	outputPath := flag.String("output", "", "output file path, default stdout")

	every := flag.Int("every", 5, "how many requests for each ip, at least 5")
	sample := flag.Int64("sample", 0xff, "rand range for picking samples")

	head := flag.Int("head", 16, "max ip number of output, 0 for all")
	showDelay := flag.Bool("show_delay", false, "show_delay")

	workers := flag.Int("workers", runtime.NumCPU()*10, "default cpu*10")

	url := flag.String("url", "https://www.cloudflare.com/cdn-cgi/trace", "your url")
	status := flag.Int("status", http.StatusOK, "status code of your url")

	timeout := flag.Int("timeout", 1000, "milliseconds")
	iFace := flag.String("interface", "", "use specific interface")

	verbose := flag.Bool("verbose", false, "show verbose output")
	insecure := flag.Bool("insecure", false, "tls skip verify")
	http2 := flag.Bool("http2", false, "force attempt http2")
	minTls := flag.Int("tls", 3, "0=tls1.0, 1=tls1.1, 2=tls1.2, 3=tls1.3")
	proxyStr := flag.String("proxy", "", "http://127.0.0.1:1081 socks5://127.0.0.1:1080 socks5h://127.0.0.1:1080")

	flag.Parse()

	if *every < 5 {
		fmt.Fprintf(os.Stderr, "every %d\n", *every)
		os.Exit(1)
	}

	var err error

	writer := os.Stdout
	if *outputPath != "" {
		writer, err = os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Create output %s %s\n", *outputPath, err)
			os.Exit(1)
		}
		defer writer.Close()
	}

	file, err := os.Open(*cidrPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open cidr file %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	rand.Seed(time.Now().Unix())
	defer ants.Release()

	var wg sync.WaitGroup

	var failed uint32
	var exited uint32

	defer func() {
		if atomic.LoadUint32(&exited) == 1 {
			os.Exit(1)
		}
	}()

	bar := pb.ProgressBarTemplate(`[{{string . "failed"}}] {{with string . "prefix"}}{{.}} {{end}}{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{rtime . "ETA %s"}}{{with string . "suffix"}} {{.}}{{end}}`).
		Start64(0).Add(0).Set("failed", atomic.LoadUint32(&failed))

	var allData ping.SortedData
	var allLock sync.Mutex

	pool, err := ants.NewPoolWithFunc(*workers, func(ip interface{}) {
		defer wg.Done()
		defer bar.Add(1)

		if atomic.LoadUint32(&exited) == 1 {
			return
		}

		var datas ping.SortedData
		for i := 0; i < *every; i++ {
			data, err := ping.Cloudflare(*url, ip.(string), *iFace, *status, *timeout, *http2, *minTls, *insecure, *proxyStr)
			if err != nil {
				atomic.AddUint32(&failed, 1)
				bar.Set("failed", atomic.LoadUint32(&failed))
				if *verbose {
					fmt.Println(ip, err)
				}
				return
			}
			datas = append(datas, data)
		}
		sort.Sort(datas)
		// 去最高和最低
		datas = datas.Remove(len(datas) - 1)
		datas = datas.Remove(0)

		var j int64
		for _, data := range datas {
			j += data.Delay
		}

		allLock.Lock()
		defer allLock.Unlock()
		allData = append(allData, &ping.Data{
			IP:    ip.(string),
			Delay: j / int64(len(datas)),
		})

	})

	barMax := 0

	addTask := func(rip string) {
		wg.Add(1)
		barMax++
		bar.SetTotal(int64(barMax))
		go pool.Invoke(rip)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var ips []string

	for scanner.Scan() {
		cidr := scanner.Text()
		if cidr == "" {
			continue
		}
		ip := net.ParseIP(cidr)
		if ip != nil {
			ips = append(ips, ip.String())
			continue
		}

		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Not CIDR %s %s\n", cidr, err)
			continue
		}

		ones, bits := ipnet.Mask.Size()
		if bits != net.IPv4len*8 {
			fmt.Fprintf(os.Stderr, "Not IPV4 %s\n", cidr)
			continue
		}

		ipCount := int64(0xffffffff >> int64(ones))
		ipInt := IPV4ToInt(ip.To4())

		for ipCount >= 0 {
			var rip net.IP
			if ipCount == 0 {
				rip = IntToIPV4(ipInt)
			} else {
				rip = IntToIPV4(ipInt + uint32(rand.Intn(int(math.Min(float64(ipCount), float64(*sample))))+1))
			}
			ipCount -= *sample
			ipInt += uint32(*sample)

			ips = append(ips, rip.String())
		}
	}

	rand.Shuffle(len(ips), func(i, j int) {
		ips[i], ips[j] = ips[j], ips[i]
	})

	for i := range ips {
		addTask(ips[i])
	}

	ips = nil

	done := make(chan interface{})
	go func() {
		wg.Wait()
		done <- nil
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

LABEL1:
	for {
		select {
		case <-ch:
			{
				atomic.SwapUint32(&exited, 1)
			}
		case <-done:
			{
				break LABEL1
			}
		}
	}

	bar.Finish()
	time.Sleep(time.Millisecond * 100)

	sort.Sort(allData)

	if *head <= 0 {
		*head = len(allData)
	} else {
		*head = int(math.Min(float64(*head), float64(len(allData))))
	}
	if *head == 0 {
		return
	}

	for i := 0; i < *head; i++ {
		if *showDelay {
			fmt.Fprintf(writer, "%-15s\t%dms\n", allData[i].IP, allData[i].Delay)
		} else {
			fmt.Fprintln(writer, allData[i].IP)
		}
	}

}

func IPV4ToInt(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip)
}

func IntToIPV4(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
