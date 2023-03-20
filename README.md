# demo-redis

This is a small go-based project to craft an `API` Backend which persist to `Redis`. It can be run on bare-metal computer as well
in containerized environment with `Docke`r and `Docker-Compose`. Keep watching for more features updates like persistence to file-like storage such as `Bolt` and authenticate & authorize with `JWT (Json Web Token)` and more. The project mimic a DDD approach but in order
to keep it minimal (avoid folders/packages) - I crafted with flat contextual files. Based on each file name, you can easily restructure
to packages-based style.

**create volume and start redis instance**

```shell
$ docker volume create redis-data
$ docker run -d --name redis-demo -v redis-data:/data -p 6379:6379 redis redis-server --requirepass "secret"
```

**delete the container and volume**

```shell
docker rm -f redis-demo
docker volume rm redis-data
```

**connect to redis server from another temporary redis-cli container.**

```shell
docker run --rm -it --link redis-demo:redis-cli --name redis-cli redis sh

redis-cli -h redis-demo

auth default secret
```
