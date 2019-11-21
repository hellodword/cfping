# cfping

> ping cloudflare IPs with TLS 1.3

## Summary

默认利用 `https://www.cloudflare.com/cdn-cgi/trace` 进行检测，不关注带宽（没有发现过有限速），只关注 https 请求的成功与否以及延迟。

也不关注延迟具体多少，不去计算 rtt，只关注整体延迟。

## Usage

```
Usage
  -cidr string
        path to cidr file (default "cidr.txt")
  -every int
        how many requests for each ip, at least 5 (default 5)
  -head int
        max ip number of output, 0 for all (default 16)
  -http2
        force attempt http2
  -insecure
        tls skip verify
  -interface string
        use specific interface
  -output string
        output file path, default stdout
  -proxy string
        http://127.0.0.1:1081 socks5://127.0.0.1:1080 socks5h://127.0.0.1:1080
  -sample int
        rand range for picking samples (default 255)
  -show_delay
        show_delay
  -status int
        status code of your url (default 200)
  -timeout int
        milliseconds (default 1000)
  -tls int
        0=tls1.0, 1=tls1.1, 2=tls1.2, 3=tls1.3 (default 3)
  -url string
        your url (default "https://www.cloudflare.com/cdn-cgi/trace")
  -verbose
        show verbose output
  -workers int
        default cpu*10 (default 80)
```

## Build

```shell
go build -o cfping .
```

## Example

```shell
env INTERFACE=eth0 \
    CIDR=./resource/ips-v4 \
    URL='https://example.com/path' \
    STATUS=200 \
    CUSTOM_ARGS='-proxy socks5h://127.0.0.1:1080' \
      bash example.sh
```

## TODO

- [ ] refactor with cmdr
- [ ] 定期以及长期检测，所以需要降低压力，且配合数据库
- [ ] 分散请求，而不是一次性请求完同一个 IP
- [ ] ja3
