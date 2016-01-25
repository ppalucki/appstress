# build & deploy
```
go build
scp ./appstress dockerhost:appstress
scp $(go tool -n pprof) dockerhost:pprof
```

# docker configuration
```
ssh dockerhost
sudo systemctl edit --full docker
# add
Environment=GODEBUG=schedtrace=2000
--debug
sudo systemctl cat docker
sudo systemctl enable docker
sudo systemctl start docker
sudo systemctl status docker
sudo systemctl cat docker

# resurrect (checkout systemd killmode)
sudo systemctl restart docker
sudo systemctl kill docker
./appstress rmall
docker info
```

# dockerlog 
## start
```
sudo systemd-run --unit=dockerlog bash -c 'journalctl --unit docker --follow --output cat >/var/log/docker.log'
sudo systemctl status dockerlog
tail -f /var/log/docker.log
```
## reset
```
sudo systemctl stop dockerlog
sudo systemctl reset-failed
```

# help
```
./appstress -h
```

# run benchmarks
## batch tb
```
sudo systemd-run --unit=appstress /home/core/appstress -all -name tb -b 5000 pull rmall sleep tb sleep rmall
```

## parallel tn
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tn -n 1000 pull rmall sleep tn sleep rmall
```

## batch & parallel
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tnb -n 5 -b 100 pull rmall sleep tnb sleep rmall
```

## watch appstress logs
```
tailf /influx.data
cat /influx.data | wc -l
```

## monitor appstress
```
systemctl status appstress
journalctl -u appstress -f
```

# analyze
```
# copy results
scp dockerhost:/influx.data influx-`date -I`.data
# feed influx
./appstress -feedInflux influx-`date -I`.data -influx "http://127.0.0.1:8086/write?db=docker"
```

# ulimits for ssh
```
systemctl edit --full sshd@
LimitNOFILE=infinity
```

