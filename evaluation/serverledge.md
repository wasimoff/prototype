# Serverledge Deployment

### Preparations

Following along the commands in the [`deploy.sh` script from the artifacts paper](https://gist.github.com/grussorusso/1c1cca879616a038a99641ce8097a81f), it can mostly be condensed to:

* Install Docker (and configure your user to allow access)
* Download the [release ZIP for tag `v1.0.0-percom23`](https://github.com/grussorusso/serverledge/archive/refs/tags/v1.0.0-percom23.zip) (or checkout the submodule)
* Pull the Python runtime image `grussorusso/serverledge-python310`

### Compile the binary

Compile the `serverledge` binary simply with `make`:

```
make
```

I ran `make serverledge{,-cli}` instead and that seemed to install all necessary binaries, too.

**Note:** for ease of use I put a ZIP with the built binaries into the git repository. If you're on Linux amd64, simply extract and use those.

### Start `etcd` server

To start even a single node, you need `etcd`. Mirroring the command from `scripts/start-etcd.sh` but with a different name, you can use:

```
docker run -d --rm --name serverledge-etcd \
  -p 2379:2379 -p 2380:2380 \
  -e ALLOW_NONE_AUTHENTICATION=yes \
  -e ETCD_ADVERTISE_CLIENT_URLS=http://localhost:2379 \
  bitnami/etcd:latest
```

This alone gets my fans going already because `etcd` has quite some base load. You can check *all the values* in the store with `etcdctl`. Optionally, add `-w=json`, but this double-encodes everything because apparently etcd can contain binary keys and values.

```
docker exec serverledge-etcd etcdctl get "" --prefix
```

### Run the Node

The main `serverledge` binary accepts only a config filename as an argument. Just start it with the default configuration:

```
bin/serverledge
```

To customize a configuration, look at the YAML files in `examples/` and check the [configuration docs](https://github.com/grussorusso/serverledge/blob/main/docs/configuration.md).

### Register a function

Register the `isprime.py` example with:

```
bin/serverledge-cli create --function isprime --src examples/isprime.py --runtime python310 --handler "isprime.handler"
```

The `--memory ...` argument from the artifact paper does not seem to be *required* at this point.

### Invoke a function

This step fails for me because *of course* I am running this in my usual rootless Docker instance .. and the runtime container that is created upon invocation uses a `slirp4netns` network bridge .. which isn't routable from the host because the subnet is not known on any of the host interfaces.

Adding a strategic `NetworkMode: "host"` in `internal/container/docker.go` didn't help either because now the runtime attempted to connect to `http://:8080/invoke`. 

Instead, I added myself to the `docker` group (and either re-login or use `sudo su - "$(id -un)"`), which allows me to use the system's Docker daemon by setting:

```
export DOCKER_HOST=unix:///var/run/docker.sock
```

That finally seemed to work but the very first cold-start was very slow because I forgot to pull the Python runtime image into the system daemon beforehand:

```
bin/serverledge-cli invoke -f isprime -p "n: 13"
```

Forgetting to pass any parameters yields a "success" with no result, which is fun. But to be fair, this is entirely up to the simple handler implementation in `isprime.py`.

```json
{
	"Success": true,
	"Result": "{}",
	"ResponseTime": 1.023181089,
	"IsWarmStart": false,
	"InitTime": 1.018023954,
	"OffloadLatency": 0,
	"Duration": 0.005149610999999998,
	"SchedAction": ""
}
```

The warm-start time is impressive though. The whole invocation returns in around 10 milliseconds measured in the shell.

# Use a Loadbalancer?

Okay, so after some experimenting and reading the code, I confirmed that the setup closest to my Client—Broker—Provider model is running a Serverledge LoadBalancer and multiple Serverledge Nodes, then running all invocations against the LB.

```
                      /×--- Cloud Node
CLI --> LoadBalancer (×---- Cloud Node
                      \×--- Cloud Node
```

* The LoadBalancer can be compiled like the rest, with `make lb`.
* LB and Nodes must be in the same region (`registry.area: REGION`).
* The Nodes **must** be cloud nodes for the LB to pick them up! (`cloud: yes`)
* When running on the same machine, use different ports, of course! (`api.port`)
* Using `serverledge-cli status` panics but running functions seems to work so far.

It will be interesting to see how the performance compares because the LoadBalancer [uses a simple RoundRobin balancer](https://github.com/grussorusso/serverledge/blob/8d20606682cc81998441f371911a8024b9e13037/internal/lb/lb.go#L20C29-L20C29). I'm not sure if the Cloud Nodes will then additionally offload among each other, when full?

Turns out I can also use full nodes and specify `edgeonly` policy in the configuration. Not sure how exactly that compares to the LoadBalancer but it seems conceptually similar because the first node is not processing any requests anymore. During my testing with the `edgeonly` policy I often saw that Serverledge decided to schedule the tasks on only one of the nodes. Probably because according to the `memory` parameter the nodes would still have space available, while long running out of processing power. The Hetzner nodes with which I am debugging have only three shared cores! So I am going to use a LoadBalancer after all.

# Create a custom TSP image

To be able to compare my Travelling Salesman binary between wasimoff and Serverledge, I'll need to create a custom Docker image. Thankfully, [the docs describe how to do that](https://github.com/grussorusso/serverledge/blob/main/docs/custom_runtime.md) and base your image on their simple Alpine image with a Go executor calling a custom binary.

In order to pass my `[ "rand", "10" ]` args to the binary I need a wrapper script. I chose to only pass the number in a simple JSON struct like `{"n": 12}`, so you can use the `-p` flag to `invoke`.

For now, I amended the `travelling_salesman` program with a Makefile target to build the `docker.io/ansemjo/serverledge-custom:tsp` image. This can be registered as a function like so:

```
bin/serverledge-cli -H provider00.ansemjo.de create --function tsp --runtime custom --custom_image docker.io/ansemjo/serverledge-custom:tsp --memory 128
```

**NOTE:** You can specify `--cpu 1.0` instead of the memory requirement, when you know it's going to be CPU-heavy, like the TSP. That should clear up congestion problems, I think? **NO! It actually makes it worse.** Since there appears to be no "waiting" for free resources in the scheduler, it almost immediately replies with `429 Too Many Requests` when you have more users than cores ... even with only six users I get like 15% failures.

During debugging, you can quickly "clean up" on a node to force it to pull the image anew with:

```
systemctl stop serverledge && docker ps -aq | xargs -rn1 docker rm -f && docker rmi ansemjo/serverledge-custom:tsp && systemctl start serverledge
```

### Preliminary Results

First, it's interesting to note that at 12 random cities, there isn't even much difference between native and wasmtime runs of the `tsp` binary.

Secondly, building `tsp` with `--target x86_64-unknown-linux-musl` (in order to build a static binary, required for the customized Alpine base image) *makes the binary slower*. This is probably due to a slow memory allocator in the musl libc; but it is *really* noticeable.

Finally, actually running TSP workloads between Wasimoff and Serverledge showed a small advantage for Wasimoff! More in the full evaluation later ...
