# tcping
```
case `uname -m` in aarch64|arm64) VER="arm64";; x86_64|amd64) VER="amd64";; *) VER=`read -p "Arch:"`;; esac; wget -qO ./tcping "https://raw.githubusercontent.com/MoeClub/tcping/main/${VER}/linux/tcping" && chmod a+x ./tcping

./tcping -i 1 -w 1 google.com 443
./tcping -h


./tcping -i -1 -c 1000 baidu.com 443

./tcping -c 0 -dns 8.8.8.8:53 baidu.com

```
