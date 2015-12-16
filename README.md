# How to build and run on CoreOS


after copying(cloning) sources to destination machine:
```sh
DOCKER_HOST=unix:///var/run/early-docker.sock docker run -ti --name golang -v `pwd`:/go/src/app/ --net host golang go get -v app && go build -v app
DOCKER_HOST=unix:///var/run/early-docker.sock docker start golang
./dockerstress
```

(use early-docker to not influence primary docker engine)



# Get the traces

wget 'http://localhost:8080/debug/pprof/profile' -O t1.pprof
wget 'http://localhost:8080/debug/pprof/trace?seconds=30' -O t1.trace

go tool pprof -output t1.pprof http://localhost:8080/debug/pprof/profile



