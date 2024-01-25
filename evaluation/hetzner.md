# Hetzner API with `hcloud`

While thinking about semi-automating more of the evaluation, I noticed that I'll be deleting and creating many servers. *There must be a better way.* For some reason I thought that I wouldn't be able to actually *buy* servers through `hcloud` but alas, that's not a problem.

### List Locations

```
$ hcloud location list
ID   NAME   DESCRIPTION             NETWORK ZONE   COUNTRY   CITY
1    fsn1   Falkenstein DC Park 1   eu-central     DE        Falkenstein
2    nbg1   Nuremberg DC Park 1     eu-central     DE        Nuremberg
3    hel1   Helsinki DC Park 1      eu-central     FI        Helsinki
4    ash    Ashburn, VA             us-east        US        Ashburn, VA
5    hil    Hillsboro, OR           us-west        US        Hillsboro, OR
```

### List Server-Types

(abbreviated)

```
$ hcloud server-type list
ID    NAME    CORES   CPU TYPE   ...  MEMORY     DISK
...
22    cpx11   2       shared          2.0 GB     40 GB
23    cpx21   3       shared          4.0 GB     80 GB
24    cpx31   4       shared          8.0 GB     160 GB
25    cpx41   8       shared          16.0 GB    240 GB
26    cpx51   16      shared          32.0 GB    360 GB
...
96    ccx13   2       dedicated       8.0 GB     80 GB
97    ccx23   4       dedicated       16.0 GB    160 GB
98    ccx33   8       dedicated       32.0 GB    240 GB
99    ccx43   16      dedicated       64.0 GB    360 GB
100   ccx53   32      dedicated       128.0 GB   600 GB
101   ccx63   48      dedicated       192.0 GB   960 GB
```

### IP Addresses

If I create a bunch of static IPv4 addresses beforehand I should be able to assign them to servers and delete servers without losing the IP addresses. So I wouldn't need to bother with the DNS console.

### Ansible

*Of course,* all of this is scriptable through Ansible as well: https://docs.ansible.com/ansible/latest/collections/hetzner/hcloud/hcloud_server_module.html#ansible-collections-hetzner-hcloud-hcloud-server-module

So in the end, both server creation and DNS updates were automated in Ansible and I didn't need to use the
`hcloud` CLI all that much.