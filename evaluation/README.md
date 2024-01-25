# Wasimoff Evaluation

This evaluation uses [Ansible](https://docs.ansible.com/) to prepare a set of virtual machines and [Locust](https://locust.io/) to generate workload on the network.

* a [separate README](./serverledge.md) describes my journey with Serverledge
* and [a few notes about the `hcloud` tool](./hetzner.md) for the Hetzner API

All of the workloads will be based on the Travelling Saleman Problem for better comparability between systems. Its runtime can be adjusted with the set size of random cities that should be computed.

## Preparations

### Virtual Machines

[Hetzner Cloud machines](https://www.hetzner.com/cloud) were used for the evaluation. Partly because an account with Hetzner was already available and because the provided easy and cheap access to servers with *dedicated* vCPUs, which is important to avoid scheduling issues due to colocation with other customers. They can also be quickly "reset" by rebuilding the server from a base image.

During development of these scripts, mostly `CPX21` instances were used. The actual performance evaluation ran on `CCX{13..16}` machines with 2 to 48 dedicated cores and **Debian 12** as a base image.

The deployment can be automated if you user the Hetzner Cloud and have a DNS domain hosted with Hetzner, too. Configure your tokens in the sops-encrypted file `./ansible/files/hetzner_token.sops.yaml`. Otherwise you're free to use your own server hosting and domain names, of course. Though some Ansible playbooks might need tweaking then.

### Ansible Role

For some of the Ansible playbooks, an additional Ansible role `geerlingguy.docker` is required. It can be quickly installed with `ansible-galaxy`:

```
sudo mkdir -p /etc/ansible/roles
sudo ansible-galaxy role install \
  -p /etc/ansible/roles/ geerlingguy.docker
```

Make sure the local `ansible.cfg` searches that path and you're good to go.

## Tips and Tricks

### Get interactive Ansible facts

The default output from `ansible -m setup ...` is not machine-readable because there's some clobber before the JSON. Use these environment variables to clean that up and search the facts interactively with `fx`:

```
ANSIBLE_LOAD_CALLBACK_PLUGINS=true ANSIBLE_STDOUT_CALLBACK=json \
ansible -m setup serverledge[0] \
| jq ".plays[0].tasks[0].hosts" \
| fx
```

### Hetzner API `hcloud`

To speed up some steps – in lieu of full automation with Terraform or the like – [the `hcloud` CLI](https://github.com/hetznercloud/cli) can be used to interact with the servers. In order to quickly clean up between different configurations, [rebuild the hosts from their base Debian image](https://docs.hetzner.cloud/#server-actions-rebuild-a-server-from-an-image) like so:

```
$ hcloud server list -o noheader -o columns=name \
   | xargs -rn1 hcloud server rebuild --image debian-12
10s [====================================] 100.00%
Server 37470876 rebuilt with image debian-12
...
```

Of course, the Hetzner API can also be automated with Ansible ...

## Pingtest

A `ping` test is run pairwise between [each region](https://docs.hetzner.com/cloud/general/locations/) to find the usual latency along the paths. With five locations that is ten tests in total.

```
ping -i 0.2 -c 256 <destination>
```

| eu-central            | us-east              | us-west                |
| --------------------- | -------------------- | ---------------------- |
| DE Falkenstein `fsn1` | US Ashburn, VA `ash` | US Hillsboro, OR `hil` |
| DE Nuremberg `nbg1`   |                      |                        |
| FI Helsinki `hel1`    |                      |                        |

## Run the Throughput Evaluation

Before each run, decide on your configuration of servers:

* In which region(s) do you deploy your servers? Are they all together? Client and Broker colocated, Providers far away? Globally distributed?
* How powerful are the machines? Many small Providers? Few "beefy" machines?
  * The Broker can probably just always use `broker` size. Monitor the CPU usage and maybe rescale if it's becoming a bottleneck.

The following are my notes on how I decided which scenarios to run through:

| ID (always `n=6,10`) | Providers                                                   | Geo Spread                                                   | Notes                                 |
| -------------------- | ----------------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------- |
| `lots-of-small`      | {2..32:2}× CCX13 "tiny"                                     | Client–Broker in EU, Broker–Provider: `nbg–nbg`, `ngb–fsn`, `ngb–hel`, `nbg–ash`, `nbg–hil` |                                       |
| ..., `optimal`       | ...                                                         | *All* in same location                                       |                                       |
| ..., `badconn`       | ...                                                         | Broker–Provider in same, Client in US                        |                                       |
| ..., `worstcase`     | ...                                                         | US-E – EU – US-W                                             | loooong latencies                     |
| `some-medium`        | {2..8:2}× CCX33 "medium"                                    | ...                                                          | **repeat** spread like `lots-of-tiny` |
| `geospread`          | {4,8,16,32}x CCX13 "tiny"                                   | Client–Broker in EU, Providers Spread across the rest        |                                       |
| `david-v-goliath`    | [48,16],[2·32],[4·16],[8·8],[16·4],[32·2] *(by core count)* |                                                              |                                       |

* "lots of tiny", n=8,10, p=2..32
  * with all in same region; broker–provider: nbg–nbg, ngb–fsn, ngb–hel, nbg–ash, nbg–hil
  * with distributed providers across all regions, p=4,8,16,32
* "a few medium", n=8,10, p=1..8
  * all in same region as above
  * distributed as above but p=4,8
* "david vs goliath", n=8,10, all in same region
  * p=[48,16],[2·32],[4·16],[8·8],[16·4],[32·2]
  * the [32·2] test is a "symlink" to the largest "lots of tiny"

### Useful Hetzner machines

| Name    | Alias    | vCPU [n]      | RAM [GB] |
| ------- | -------- | ------------- | -------- |
| `CPX11` |          | *shared:* 2   | 2        |
| `CPX21` |          | *shared:* 3   | 4        |
| `CPX31` |          | *shared:* 4   | 8        |
| `CCX13` | `tiny`   | *dedict.:* 2  | 8        |
| `CCX23` | `broker` | *dedict.:* 4  | 16       |
| `CCX33` | `medium` | *dedict.:* 8  | 32       |
| `CCX43` |          | *dedict.:* 16 | 64       |
| `CCX53` | `large`  | *dedict.:* 32 | 128      |
| `CCX63` | `huge`   | *dedict.:* 48 | 192      |

### Perform a Run

* Edit the servers and workload descriptions in `inventory.ini` and set the scenario string accordingly (edit the `hc_provds_string`, the rest is templated).

  * Start with the **largest setup** of a scenario! This means that you will only need to setup all the providers once. After that you can incrementally delete unnecessary ones as you perform smaller scenarios.

* On the first run, create the server landscape and setup all the nodes:

  ```
  ./create-servers.sh \
  && ./setup-locust.yaml \
  && ./setup-wasimoff.yaml \
  && ./setup-serverledge.yaml \
  && ./notify.sh "Scenario '$(./getivar.sh "{{ scenario }}")' created!" \
  || ./notify.sh "Scenario creation failed."
  ```

* Stop serverledge services, connect to the locust node, start the providers and run the wasimoff workload:

  ```
  ./startstop-serverledge.sh stop
  ./connect-locust.sh
    ./start-providers
    ^A-d (detach)
    ./run-wasimoff 10
    ./run-wasimoff 8
    tmux at
    ^C (until all are stopped)
  ^D (exit ssh and see that rsync copies results)
  ```

* Start the serverledge services (possibly from another shell, then you don't need to disconnect from locust above), connect to the locust node again and run the serverledge workload:

  ```
  ./startstop-serverledge.sh start
  ./connect-locust.sh
    ./run-serverledge 10
    # --> in between runs use './startstop-serverledge.sh restart' to clean up
    ./run-serverledge 8
  ^D (exit ssh and see that rsync copies results)
  ```

  As noted, restart the serverledge services in between runs with different parameters, to clean up all Docker containers and simulate cold starts for the next run.

* On the next scenario, when you only *remove* servers, you can simply reconfigure the scripts on locust and don't need to touch the Broker or Providers at all.
  ```
  ./setup-locust.yaml
  ```

  Continue at "stopping serverledge services" above.

### Wasimoff

* Setup all the nodes with `ansible-playbook setup-wasimoff.yaml`. This will setup the `broker`, the `provider`s and finally the `locust` node.

  * The playbook needs to get a valid Letsencrypt certificate, so make sure it has a proper DNS name and is publicly reachable. The certificate is required for QUIC to work.

* In a separate window, use `connect-providers.sh` to start a `tmux` session, which connects to all the Providers and starts the `chromium-shell` on them.
  Optionall pass the argument `show`, to display the `console.log` output. It's hidden by default because it eats up quite a bit of SSH bandwidth.

* Upload the Wasm binaries necessary for your chosen workload to the Broker, if this wasn't done by the Ansible playbook yet.

* Connect to the workload client with `connect-locust.sh` and select the workload file to run. You should already be in the correct directory and can start the workload with:
  ```
  ./locust -f workload_<...>.py
  ```

  The script will have opened a local port forwarding to the locust node, so you can open the web interface on `http://localhost:8089` to start the run or use proper arguments for locust to start it immediately.

### Serverledge

* Similar to the first step above, run `ansible-playbook setup-serverledge.yaml`. The playbook is designed so it can run "on top" of the Wasimoff setup without a rebuild in between.

  * This whole setup is unauthenticated and unencrypted. The `etcd` instance is open to the world with `NONE` authentication. Do not leave this up longer than necessary.

* The nodes are all started by `systemd` services. You can connect to all of the providers at once, nonetheless, to monitor the evaluation with `journalctl` by running `connect-serverledge.sh` in a separate window.

* Make sure to register the necessary functions in advance with `serverledge-cli`:
  ```
  bin/serverledge-cli -H broker.ansemjo.de \
    create --function tsp --runtime custom \
    --custom_image docker.io/ansemjo/serverledge-custom:tsp \
    --memory 128
  ```

  (See the separate README on why I don't use `--cpu 1.0` here ...)

* Finally, connect to the Locust node as above and pick the appropriate workload file.

## Additional Evaluations

* `hyperfine` tests of native vs. native static vs. wasmtime vs. local wasimoff
  * runtime × parameter `n`
  * gnuplot errorlines/errorbars
* live scaling with additional resources
  * extract locust plot as svg? probably rather my own gnuplot
  * use `--csv=... --headless -t 10m` [to save statistics](https://docs.locust.io/en/stable/retrieving-stats.html)
* after finding "max" throughput, do throughput × response time?
* complete trace of a single run
  * compilation and upload are treated separately
  * but then: send request, select provider, transfer request, tracing in browser, transfer result
