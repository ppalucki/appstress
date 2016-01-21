# docker configuration
```
sudo systemctl edit --full docker
# add
Environment=GODEBUG=schedtrace=2000
--debug
sudo systemctl enable docker
sudo systemctl start docker
sudo systemctl restart docker
sudo systemctl status docker
```

# dockerlog 
## start
```
sudo systemd-run --unit=dockerlog bash -c 'journalctl --unit docker --follow --output cat >/var/log/docker.log'
sudo systemctl status dockerlog
tail /var/log/docker.log
```
## reset
```
sudo systemctl stop dockerlog
sudo systemctl reset-failed
```

# build & deploy
```
go build
scp ./appstress dockerhost:appstress
```

# help
```
./appstress -h
```

# run benchmarks
## batch tb
```
sudo systemd-run --unit=appstress /home/core/appstress -all -name tb 1000 pull rmall sleep tb sleep rmall
systemctl status appstress
journalctl -u appstress -f
```

## parallel tn
```
sudo systemd-run --unit=appstress_tn -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tn -n 1000 pull rmall sleep tn sleep rmall
systemctl status appstress_tn
journalctl -u appstress_tn -f
```

## batch & parallel
```
sudo systemd-run --unit=appstress_tnb -p LimitNOFILE=1048576 -p LimitNPROC=1048576 /home/core/appstress -all -name tnb -n 50 -b 100 pull rmall sleep tnb sleep rmall
systemctl status appstress_tnb
journalctl -u appstress_tnb -f
```

## watch logs
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
./appstress -feedInflux influx-`date -I`.data -influx "http://127.0.0.1:8083/write?db=docker"
```

# ulimits for ssh
```
systemctl edit --full sshd@
LimitNOFILE=infinity
```

