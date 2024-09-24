# TagIt 
[![Go Report Card](https://goreportcard.com/badge/github.com/ncode/tagit)](https://goreportcard.com/report/github.com/ncode/tagit)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![codecov](https://codecov.io/gh/ncode/tagit/graph/badge.svg?token=ISXEH274YD)](https://codecov.io/gh/ncode/tagit)


Update consul registration tags with outputs of a script.
It copies the current service registration and appends the output of the script line by line as tags, while keeping the original tags.

## Why?

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

## How to test it?

```bash
$ git clone github.com/ncode/tagit
$ cd configs/development
$ make
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

## Todo

- [ ] Adds a systemd unit file generator
