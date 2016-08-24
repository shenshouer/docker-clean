# docker-clean

## build 

```
$ make build
```

## build docker image and push 

```
$ make image
```

## usage

```
./docker-clean --docker-host http://localhost:2375 --start-time 16:12 --stop-time 16:14

docker run --name test-docker-clean -v /var/run/docker.sock:/var/run/docker.sock dhub.yunpro.cn/shenshouer/docker-clean:dev --docker-host http://10.50.1.31:2375
```