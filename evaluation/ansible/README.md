# Ansible for Wasimoff evaluation

This directory contains the inventory file `inventory.ini` and various playbooks
and scripts that are used to setup an evaluation environment in the Hetzner Cloud.

The workflow is tightly coupled to the [throughput evaluation](../eval_throughput/)
and is split into several files to be able to skip certain steps during the eval
runs; you don't need to setup Locust repeatedly, for example. The playbooks also
rely on a Hetzner Cloud account and an available domain with its nameservers
configured to be Hetzner DNS.

* Uncomment and modify `locust`, `broker` and `provider` groups in the inventory
  file as needed. Don't forget to adjust the `scenario` string, too.
* Create servers with the `./01-create-servers.sh` script.
* Setup the servers with playbooks:
  * `ansible-playbook ./02-setup-broker.yaml`
  * `ansible-playbook ./02-setup-provider.yaml`
  * `ansible-playbook ./02-setup-locust.yaml`
  * `ansible-playbook ./02-setup-serverledge.yaml`
* Start the Wasimoff Provider software in a Chromium shell on all nodes with
  `./connect-providers.sh` in a separate window.
* Connect to the Locust node and run the desired workloads.

### sops-encrypted values

Various places use [sops](https://github.com/getsops/sops)-encrypted files for
secret values and private files, e.g. the Hetzner API token and previously
obtained TLS certificates. Ansible expects to be able to just load these files
with the `community.sops.load_vars` role, so you need to make sure to setup
sops beforehand.

I like [age](https://github.com/FiloSottile/age) and used the environment variable
`SOPS_AGE_KEY_FILE` to point sops to the correct key file.