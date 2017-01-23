# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi) [![Go Report Card](https://goreportcard.com/badge/github.com/lbarman/prifi)](https://goreportcard.com/report/github.com/lbarman/prifi) [![Coverage Status](https://coveralls.io/repos/github/lbarman/prifi/badge.svg?branch=master)](https://coveralls.io/github/lbarman/prifi?branch=master)

## Introduction


This repository implements PriFi, an anonymous communication protocol with provable traffic-analysis resistance and small latency suitable for wireless networks. PriFi provides a network access mechanism for protecting members of an organization who access the Internet while on-site (via privacy-preserving WiFi networking) and while off-site (via privacy-preserving virtual private networking or VPN). The small latency cost is achieved by leveraging the client-relay-server topology common in WiFi networks. The main entities of PriFi are: relay, trustee server (or Trustees), and clients. These collaborate to implement a Dining Cryptographer's network ([DC-nets](https://en.wikipedia.org/wiki/Dining_cryptographers_problem)) that can anonymize the client upstream traffic. The relay is a WiFi router that can process normal TCP/IP traffic in addition to running our protocol.

For an extended introduction, please check our [website](https://lbarman.ch/prifi/).

For more details about PriFi, please check our [WPES 2016 paper](http://www.cs.yale.edu/homes/jf/PriFi-WPES2016.pdf).


**Warning: This software is experimental and still under development. Do not use it yet for security-critical purposes. Use at your own risk!**

## Getting PriFi

First, [get the Go language](https://golang.org/dl/), >= 1.7. They have some `.tar.gz`, but I personally prefer to use my package manager :
`sudo apt-get install golang` for Ubuntu, or `sudo dnf install golang` for Fedora 24.

Then, get PriFi by doing:

```
go get github.com/lbarman/prifi/sda/app
cd $GOPATH/src/github.com/lbarman/prifi
./prifi.sh install
```

## Running PriFi

PriFi uses [ONet](https://github.com/dedis/onet) as a network framework. It is easy to run all components (trustees, relay, clients) on one machine for testing purposes, or on different machines for the real setup.

Each component has a *SDA configuration* : an identity (`identity.toml`, containing a private and public key), and some knowledge of the others participants via `group.toml`. For your convenience, we pre-generated some identities in `config/identities_default`.

### Testing PriFi, all components in localhost

You can test PriFi by running `./prifi.sh all-localhost`. This will run a SOCKS server, a PriFi relay, a Trustee, and three clients on your machine. They will use the identities in `config/identities_default`. You can check what is going on by doing `tail -f {clientX|relay|trusteeX|socks}.log`. You can test browsing through PriFi by setting your browser to use a SOCKS proxy on `localhost:8081`.

### Using PriFi in a real setup

To test a real PriFi deployement, first, re-generates your identity (so your private key is really private). The processed is detailed in the [README about ./prifi.sh startup script](README_prifi.sh.md).

## More documentation :

 - [README about the Architecture and SOCKS Proxies](README_architecture.md)

 - [README about ./prifi.sh startup script](README_prifi.sh.md)

 - [README about contributing to this repository](README_contributing.md)

## API Documentation

The PriFi API documentation can be found in  `doc/doc.html`. 

