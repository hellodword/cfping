# cfping

> ping cloudflare IPs with TLS 1.3

## Summary
利用 `https://www.cloudflare.com/cdn-cgi/trace` 进行检测，不关注带宽，只关注 https+TLS1.3 请求的成功与否以及延迟。

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
  -interface string

  -output string
        output file path, default stdout
  -sample int
        rand range for picking samples (default 255)
  -text
        default false and output json
  -workers int
        default cpu*10
```

## Build

```shell
go build -o cfping ./cmd
```

## Example
```shell
./cfping -cidr ./cidr.txt -output output.txt -every 5 -sample 255 -head 10 -workers 10 -text
```

## TODO

- [ ] 定期以及长期检测，所以需要降低压力，且配合数据库
- [ ] 分散请求，而不是一次性请求完同一个 IP
- [ ] ja3