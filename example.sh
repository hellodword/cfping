#!/bin/bash

set -e
set -x

[ -z "$INTERFACE" ] && echo "INTERFACE" && exit 1
[ -z "$CIDR" ] && echo "CIDR" && exit 1
[ -z "$URL" ] && echo "URL" && exit 1
[ -z "$STATUS" ] && echo "STATUS" && exit 1

# record ip
IP=$(curl -s -4 --interface $INTERFACE https://myip.ipip.net | grep -o '[0-9]*\.[0-9]*\.[0-9]*\.[0-9]*')
echo "IP = $IP"

NOW="$(date +%Y-%m-%d-%H-%M-%S)"

TARGET="results/$NOW"

mkdir -p "$TARGET"

echo "$IP" > "$TARGET/ip.txt"
cp $CIDR "$TARGET/cidr.txt"

# 先随机扫一批出来
./cfping $CUSTOM_ARGS \
         -interface $INTERFACE \
         -url $URL -status $STATUS \
         -cidr "$TARGET/cidr.txt" \
         -output "./$TARGET/first.txt" \
         -every 5 -sample 31 \
         -head 0 \
         -workers 16

# 以这一批为 /24，并且保留它为 /32
sed -i -e 's/^\(.\+\)\.\([0-9]\+\)$/\1\.\2\n\1\.0\/24/g' "./$TARGET/first.txt"
cat "./$TARGET/first.txt" | sort -nr | uniq > "./$TARGET/second.txt"

./cfping $CUSTOM_ARGS \
         -interface $INTERFACE \
         -url $URL -status $STATUS \
         -cidr "./$TARGET/second.txt" \
         -output "./$TARGET/third.txt" \
         -every 5 -sample 7 \
         -head 0 \
         -workers 16

head -n 256 "./$TARGET/third.txt" > "./$TARGET/third-256.txt"

# 再精细扫描选择最快
for i in {1..5}
do
  echo >> "./$TARGET/results.txt"
  ./cfping $CUSTOM_ARGS \
           -interface $INTERFACE \
           -url $URL -status $STATUS \
           -cidr "./$TARGET/third-256.txt" \
           -every 20 -sample 1 \
           -head 16 \
           -workers 8 \
           -show_delay | tee -a "./$TARGET/results.txt"
done
