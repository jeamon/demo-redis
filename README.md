# demo-redis

[![Build Status](https://github.com/jeamon/demo-redis/actions/workflows/tests.yml/badge.svg)](https://github.com/jeamon/demo-redis/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeamon/demo-redis)](https://goreportcard.com/report/github.com/jeamon/demo-redis)
[![codecov](https://codecov.io/gh/jeamon/demo-redis/graph/badge.svg)](https://codecov.io/gh/jeamon/demo-redis)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jeamon/demo-redis)
[![MIT License](https://img.shields.io/github/license/jeamon/demo-redis)](https://github.com/jeamon/demo-redis/blob/main/LICENSE)

This is a small go-based project to craft an `API` Backend which persist to `Redis`. It can be run on bare-metal computer as well
in containerized environment with `Docker` and `Docker-Compose`. Keep watching for more features updates like `redis-based pubsub`  async event for persisting data to file-like storage such as `BoltDB` and authenticate & authorize with `JWT (Json Web Token)` and log monitoring with `Loki/Grafana` and metrics monitoring with `Prometheus/Grafana` and tracing with `OpenTelemetry/Tempo`. The project mimic a DDD approach but in order to keep it minimal (avoid folders/packages) - I crafted it with some flat contextual files. Based on each file name, you can easily restructure to packages-based style.

## Get-Started

You can run this project directly on your host machine (windows or linux or macos) or inside containerized environment (with docker-compose). I will be adding K8s manifest files soon. You can use the Make Tool for common tasks (check the `Makefile` for available actions). Make sure, you have `git` or [`git bash`](https://git-scm.com/downloads) and `make` tools installed along with `docker` or `docker desktop` and `docker-compose`. All these tools could be installed even on windows-based machine. i.e install [make tool on windows](https://gist.github.com/evanwill/0207876c3243bbb6863e65ec5dc3f058#make). You can use a hybrid approach like running some services (especially `Redis` container) inside a linux-based [virtual machine](https://www.vmware.com/pl/products/workstation-player.html) and the `golang app` on your local machine. Feel free to use whatever approach that better suit to your setup. Finally you can always [download golang](https://go.dev/doc/install) from official page.


**[Step 1] -** Clone or download the repository on your local machine

```shell
$ git clone https://github.com/jeamon/demo-redis.git
$ cd demo-redis
```


**[Step 2] -** Open and check the `config.yml` and `config.env` files

*Update the content (especially the redis and app host & port values) based on your setup or leave it like it is.*


**[Step 3] -** Build and/or run the project

* **Method 1:** Use docker-compose to build and spin up the project

```shell
$ make docker.build
$ make docker.run
```

* **Method 2:** Run the Redis in docker and the App locally

Go inside the `config.yml` file and change the server host value to `host: "127.0.0.1"`
Then change the `redis host` value to the exact `IP address` of the host where it is running.
Finally do the same changes inside the `config.env` file for both `server host` and `redis host`.

   * *create volume and start redis instance from your docker host*

        ```shell
        $ docker volume create db.demo.redis.data
        $ docker run -d --name db.demo.redis -v db.demo.redis.data:/data -p 6379:6379 redis redis-server --requirepass "<secret>"
        ```
    
   * *start the server on your local machine (depending if you have make or/and git tools)*

        ```shell
        $ make local.run
        ```

        ```shell
        $ DRAP_REDIS_HOST=<IP.ADDRESS.REDIS.HOST> make local.run
        ```
        
        ```shell
        $ go run -ldflags "-X 'main.GitCommit=$(shell git rev-parse --short HEAD)' -X 'main.GitTag=$(shell git describe --tags --abbrev=0)' -X 'main.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .
        ```
       
        ```shell
        $ go run .
        ```
    
   * *delete the container and volume created before for redis on your docker host*

        ```shell
        $ docker rm -f db.demo.redis
        $ docker volume rm db.demo.redis.data
        ```

   * *connect to redis server from another temporary redis-cli container (on your docker host)*

        ```shell
        $ docker run --rm -it --link db.demo.redis:redis-cli --name redis-cli redis sh
        $ redis-cli -h db.demo.redis
        $ auth default <secret>
        ```


**Step 4:** Check by performing basics requests (from `curl` or `postman` or `browser`)

```shell
## example of basic app checking
$ http://<server-address>:8080/

## example of fetching app status info
$ http://<server-address>:8080/status

## example of all books listing request
$ http://<server-address>:8080/v1/books

## example of pulling in-use app settings
$ http://<server-address>:8080/internal/configs
```

```shell
## example of book creation request

$ curl -X POST http://<server-address>:8080/v1/books \
   -H 'Content-Type: application/json; charset=UTF-8' \
   -d '{"title": "golang programming", "description": "Pratical golang exercices", "author": "Jerome Amon", "price": "10$"}'
```


## Contact

Feel free to [reach out to me](https://blog.cloudmentor-scale.com/contact) before any action. Feel free to connect on [Twitter](https://twitter.com/jerome_amon) or [linkedin](https://www.linkedin.com/in/jeromeamon/)
