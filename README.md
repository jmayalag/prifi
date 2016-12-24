# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

## Introduction


This repository implements PriFi, an anonymous communication protocol with provable traffic-analysis resistance and small latency suitable for wireless networks. PriFi provides a network access mechanism for protecting members of an organization who access the Internet while on-site (via privacy-preserving WiFi networking) and while off-site (via privacy-preserving virtual private networking or VPN). The small latency cost is achieved by leveraging the client-relay-server topology common in WiFi networks. The main entities of PriFi are: relay, trustee server (or Trustees), and clients. These collaborate to implement a Dining Cryptographer's network ([DC-nets](https://en.wikipedia.org/wiki/Dining_cryptographers_problem)) that can anonymize the client upstream traffic. The relay is a WiFi router that can process normal TCP/IP traffic in addition to running our protocol.

For more details about PriFi, please check our [WPES 2016 paper](http://www.cs.yale.edu/homes/jf/PriFi-WPES2016.pdf).


**Warning: This software is experimental and still under development. Do not use it yet for security-critical purposes. Use at your own risk!**

## Getting and running PriFi

PriFi is coded in Go. At some point, we will distribute binaries, but for now you will have to compile it. 

### Prerequisite

Simply [get the Go language](https://golang.org/dl/). They have `.tar.gz`, but I personally prefer to use my package manager :
```
sudo add-apt-repository ppa:gophers/go
sudo apt-get update
sudo apt-get install golang
```
... for Ubuntu, or 
`sudo dnf install golang`
... for Fedora 24.

### Get PriFi

```
go get github.com/lbarman/prifi
./prifi.sh install
```
(ignore the `No buildable source` after the first step, that's OK.)


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

## Getting PriFi

Simply do
```
git clone https://github.com/lbarman/prifi_dev
```

WORK IN PROGRESS

Fixed in cothority's test_ism_2_699 branch.
But this branch will not be merged into anything, DeDiS working on a new version.
So for the time being, we need to check out test_ism_2_699 on $GOPATH/src/github/dedis/cothority

## Running PriFi

### SOCKS Preamble

As explained, you need a non-prifi SOCKS server running to handle the traffic from the relay. If you don't have one, run ours :
```
./run-socks-proxy.sh 8090
```

## Running PriFi

There is one big startup script `run-prifi.sh`. 

```
./run-prifi.sh 
Usage: run-prifi.sh role/operation [params]
	role: client, relay, trustee
	operation: sockstest, all, deploy-all
	params for role relay: [socks_server_port] (optional, numeric)
	params for role trustee: id (required, numeric)
	params for role client: id (required, numeric), [prifi_socks_server_port] (optional, numeric)
	params for operation all, deploy: none
	params for operation sockstest, deploy: [socks_server_port] (optional, numeric), [prifi_socks_server_port] (optional, numeric)

```

For instance, you can start a relay like this : 

```
./run-prifi.sh relay
```

... or to specify the port of the second, non-prifi socks server, like this :

```
./run-prifi.sh relay 8090
```

You can start a client like this :

```
./run-prifi.sh client 0
```

and to specify the port of the first socks proxy integrated in PriFi :

```
./run-prifi.sh client 0 8080
```

A typical deployement could be :

```
./run-prifi.sh relay 8090
./run-prifi.sh trustee 0
./run-prifi.sh client 0 8080
./run-prifi.sh client 1 8081
```

## Configuration

The PriFi configuration file is in `config.demo/prifi.toml`

- `DataOutputEnbaled (bool)`: Enables the link from and to the socks proxy.
- `NTrustees (int)`: Number of trustees.
- `CellSizeUp (int)`: Size of upstream data sent in one PriFi round (?)
- `CellSizeDown (int)`: Size of upstream data sent in one PriFi round (?)
- `RelayWindowSize (int)`: Number of ciphers from each trustee to buffer
- `RelayUseDummyDataDown (bool)`: When true, the relay always send
CellSizeDown bits down. When false, it may send only 1 bit.
- `RelayReportingLimit (int)`: Unused, was for the statistics.
- `UseUDP (bool)`: Enable or disable UDP broadcast for downstream data (?)
- `DoLatencyTests (bool)`: Enable or disable latency tests.
- `ReportingLimit (int)`: PriFi shuts down after this number of rounds if
not equal to `-1`.

## API Documentation

The PriFi API documentation can be found in  `doc/doc.html`. 

