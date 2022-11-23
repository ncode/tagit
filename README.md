## TagIt

Update consul registration tags with outputs of a script.
It copies the current service registration and appends the output of the script line by line as tags, while keeping the original tags.

### Why?

Basically because it's a very useful feature that is missing from consul. Read more about it [here](https://github.com/hashicorp/consul/issues/1048).
A few scenarios where this can be useful:

1. Your databases are under mydb.service.consul, and you would like to ensure that all the writes go to the leader
   1. You run a script that checks the leader and updates the tag
2. You have a service that is not consul aware, but you would like to use consul for service discovery
   1. You run a script that checks the service and updates the tags
3. You have a load or a webserver, and you would like to have tags for all vhosts that are served by this server
   1. You run a script that checks the vhosts and updates the tags
4. Pretty much any services that are not consul aware, but you would like to use consul for service discovery
   1. You run a script that checks the service and updates the tags

### How to test it?

```bash
$ git clone github.com/ncode/tagit
$ go build
$ consul agent -dev &
$ curl --request PUT --data @examples/consul/my-service1.json http://127.0.0.1:8500/v1/agent/service/register
$ ./tagit run --consul-addr=127.0.0.1:8500 --service-id=my-service1 --script=./examples/tagit/example.sh --interval=5s --tag-prefix=tagit
INFO[0000] running command                               command=./examples/tagit/example.sh service=my-service1
INFO[0000] updating service tags                         service=my-service1 tags="[v1 tagit-nice tagit-it tagit-works]"
INFO[0005] running command                               command=./examples/tagit/example.sh service=my-service1
INFO[0010] running command                               command=./examples/tagit/example.sh service=my-service1
INFO[0015] running command                               command=./examples/tagit/example.sh service=my-service1
$ ./tagit cleanup --consul-addr=127.0.0.1:8500 --service-id=my-service1 --tag-prefix=tagit
INFO[0000] current service tags                          service=my-service1 tags="[v1 tagit-nice tagit-it tagit-works]"
INFO[0000] updating service tags                         service=my-service1 tags="[v1]"
```


```mermaid
sequenceDiagram
    participant tagit
    participant consul
    loop execute script on interval
        tagit->>consul: Do you have a service with id my-service1?
        consul->>tagit: Yes, here it is and that's the current registration
        tagit->>consul: Update current registration adding or removing prefixed tags wiht the output of the script
    end
```

### Todo

- [ ] Adds support for multiple services (currently only supports one service)
- [ ] Adds a systemd unit file generator
