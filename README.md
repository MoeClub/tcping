# tcping
```
case `uname -m` in aarch64|arm64) VER="arm64";; x86_64|amd64) VER="amd64";; *) VER=`read -p "Arch:"`;; esac; wget -qO ./tcping "https://raw.githubusercontent.com/MoeClub/tcping/main/${VER}/linux/tcping" && chmod a+x ./tcping

./tcping -i 1 -w 1 -p 443 google.com

```
