# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

## Introduction


This repository implements PriFi, an anonymous communication protocol with provable traffic-analysis resistance and small latency suitable for wireless networks. PriFi provides a network access mechanism for protecting members of an organization who access the Internet while on-site (via privacy-preserving WiFi networking) and while off-site (via privacy-preserving virtual private networking or VPN). The small latency cost is achieved by leveraging the client-relay-server topology common in WiFi networks. The main entities of PriFi are: relay, trustee server (or Trustees), and clients. These collaborate to implement a Dining Cryptographer's network ([DC-nets](https://en.wikipedia.org/wiki/Dining_cryptographers_problem)) that can anonymize the client upstream traffic. The relay is a WiFi router that can process normal TCP/IP traffic in addition to running our protocol.

For an extended introduction, please check our [website](https://lbarman.ch/prifi/).

For more details about PriFi, please check our [WPES 2016 paper](http://www.cs.yale.edu/homes/jf/PriFi-WPES2016.pdf).


**Warning: This software is experimental and still under development. Do not use it yet for security-critical purposes. Use at your own risk!**

## Getting PriFi

First, [get the Go language](https://golang.org/dl/). They have `.tar.gz`, but I personally prefer to use my package manager :
`sudo apt-get install golang` for Ubuntu, or `sudo dnf install golang` for Fedora 24.

Then, get PriFi by doing:

```
go get github.com/lbarman/prifi
./prifi.sh install
```
Ignore the `No buildable source` after the first step, that's OK. This script gets all the dependencies (via `go get`), and make sure everything is correctly set.

## Running PriFi

PriFi uses [SDA](https://github.com/dedis/cothority) as a network framework. It is easy to run all components (trustees, relay, clients) on one machine for testing purposes, or on different machines for the real setup.

Each component has a *SDA configuration* : an identity (`identity.toml`, containing a private and public key), and some knowledge of the others participants via `group.toml`. For your convenience, we pre-generated some identities in `config.localhost`.

### Testing PriFi, all components in localhost

You can test PriFi by running `./prifi.sh all-localhost`. This will run a SOCKS server, a PriFi relay, a Trustee, and three clients on your machine. They will use the identities in `config.localhost`. You can check what is going on by doing `tail -f {clientX|relay|trusteeX}.log`. You can test browsing through PriFi by setting your browser to use a SOCKS proxy on `localhost:8081`.

### Using PriFi in a real setup

To test a real PriFi deployement, first, re-generates your identity (so your private key is really private) :
```
./prifi.sh gen
```
This will create two files `config.real/{group|identity}.toml`. You can then run a PriFi client with this identity :
```
./prifi.sh client real
```

## Understanding the architecture

### Structure

The current code is organized in two main parts :

1) `PriFi-Lib`, which is network-agnostic; it takes an interface "MessageSender" that give it functions like SendToRelay(), SendToTrustee, ... and ReceivedMessage()

This is the core of the protocol PriFi. 

2) `PriFi-SDA-Wrapper` (what is in folder `sda`), that does the mapping between the tree entities of SDA and our roles (Relay, Trustee, Client), and provides the MessageSender interface discussed above.

The SDA is a framework for Secure Distributed Algorithm, developped by DeDiS, EPFL. It help bootstrapping secure protocols. The "wrapper" is simply the link between this framework and our library `PriFi-lib` (which does not know at all about `sda`).

### SOCKS

PriFi anonymizes the traffic via SOCKS proxy. Once PriFi is running, you can configure your SOCKS client (e.g. browser, mail application) to connect to PriFi.

The structure is a big convoluted : we have *two* socks servers. One is *in* the PriFi client code; that's the entry point of your upstream traffic, e.g. your browser connects to the socks server *in* PriFi on your local machine.

Then, PriFi anonymizes the traffic with the help of the other clients and the relay. The anonymized traffic is outputted at the relay.

This anonymized traffic is *SOCKS traffic*. Hence, the relay needs to connect to the second SOCKS server, which is not related to PriFi (but we provide the code for it in `socks/`). It could also be a remote, public SOCKS server.

## More documentation :

[README about the Architecture and SOCKS Proxies](README_architecture.md)
[README about ./prifi.sh startup script](README_prifi.sh.md)

## API Documentation

The PriFi API documentation can be found in  `doc/doc.html`. 

