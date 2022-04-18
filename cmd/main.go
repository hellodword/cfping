package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hellodword/cfping/ping"
	"github.com/panjf2000/ants/v2"
	"github.com/schollz/progressbar/v3"
	"math"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

type SortedData []*ping.Data

func (e SortedData) Len() int {
	return len(e)
}

func (e SortedData) Less(i, j int) bool {
	return e[i].Delay < e[j].Delay
}

func (e SortedData) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e SortedData) Remove(i int) SortedData {
	return append(e[:i], e[i+1:]...)
}

func main() {

	cidrPath := flag.String("cidr", "cidr.txt", "path to cidr file")
	every := flag.Int("every", 5, "how many requests for each ip, at least 5")
	sample := flag.Int("sample", 0xff, "rand range for picking samples")
	head := flag.Int("head", 16, "max ip number of output, 0 for all")
	text := flag.Bool("text", false, "default false and output json")
	workers := flag.Int("workers", runtime.NumCPU()*10, "default cpu*10")
	output := flag.String("output", "", "output file path, default stdout")

	flag.Parse()
	if *every < 5 {
		fmt.Fprintf(os.Stderr, "every %d\n", *every)
		os.Exit(1)
	}

	rand.Seed(time.Now().Unix())
	defer ants.Release()
	var wg sync.WaitGroup

	writer := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Create output %s %s\n", *output, err)
			os.Exit(1)
		}
		defer f.Close()
		writer = f
	}

	bar := progressbar.Default(1)

	var allData SortedData
	var allLock sync.Mutex

	pool, err := ants.NewPoolWithFunc(*workers, func(ip interface{}) {
		defer wg.Done()
		defer bar.Add(1)
		var datas SortedData
		for i := 0; i < *every; i++ {
			if data, err := ping.Cloudflare(ip.(string)); err == nil {
				datas = append(datas, data)
			} else {
				return
			}
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

	file, err := os.Open(*cidrPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open cidr file %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	max := 0
	for scanner.Scan() {
		cidr := scanner.Text()
		if cidr == "" {
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

		ipCount := 0xffffffff >> ones
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
			wg.Add(1)
			max += 1
			bar.ChangeMax(max)
			go pool.Invoke(rip.String())
		}
	}

	wg.Wait()

	sort.Sort(allData)

	if *head <= 0 {
		*head = len(allData)
	} else {
		*head = int(math.Min(float64(*head), float64(len(allData))))
	}
	if *head == 0 {
		return
	}

	if *text {
		for i := 0; i < *head; i++ {
			fmt.Fprintln(writer, allData[i].IP)
		}

	} else {
		b, _ := json.MarshalIndent(allData[:*head], "", "  ")
		fmt.Fprintln(writer, string(b))
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
