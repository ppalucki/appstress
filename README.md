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

# run benchmarks scenarios
## batch tb
```
sudo systemd-run --unit=appstress /home/core/appstress -all -name tb -b 5000 pull rmall sleep tb sleep rmall
```

## batch tb + bridge
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -net bridge -influx 'http://127.0.0.1:8086/write?db=docker' -all -b 5000 pull tb
```

## batch tb + host
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -net host -influx 'http://127.0.0.1:8086/write?db=docker' -all -b 5000 rmall pull tb
```

## parallel tn
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tn -n 1000 pull rmall sleep tn sleep rmall
```

## batch & parallel
```
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tnb -n 500 -b 100 pull rmall sleep tnb sleep rmall
```

## parallel increase 256 * 10 (max)
```
# double n (up to n clients increasd by factor 2, each running 10 containers)
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name doublen -n 256 -b 10 pull rmall doublen
# double b (up to b batch size increasd by factor 2, run by n clients)
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name doubleb -n 10 -b 256 pull rmall doubleb 
```

## paralllel direct to influx
```
ssh -R 8086:127.0.0.1:8086 dockerhost
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name doubleb -influx='http://127.0.0.1:8086/write?db=docker' -feedLines=10 -n 10 -b 256 pull rmall doubleb 
```

## other workloads
# fetch stress
```
./appstress -image jess/stress -influx null -cmd 'watch -n 1 -- stress -c 1 -t 1' -dockerUrl unix://var/run/docker.sock t1
sudo systemd-run --unit=appstress -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -image jess/stress -tty -cmd 'watch -n 1 stress -c 1 -t 1' -b 512 doubleb
```
# batch stress

## watch appstress logs
```
tailf /influx.data
cat /influx.data | wc -l
```

## monitor appstress
```
systemctl status appstress
journalctl -u appstress -f
# reset
systemctl stop appstress
sudo systemctl reset-failed
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



