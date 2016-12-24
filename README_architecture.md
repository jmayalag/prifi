# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

[back to main README](README.md)

## Architecture, and SOCKS proxies

### Structure

The current code is organized in two main parts :

1) `PriFi-Lib`, which is network-agnostic; it takes an interface "MessageSender" that give it functions like `SendToRelay()`, `SendToTrustee()`, ... and `ReceivedMessage()`

This is the core of the protocol PriFi. 

2) `PriFi-SDA-Wrapper` (what is in folder `sda`), that does the mapping between the tree entities of SDA and our roles (Relay, Trustee, Client), and provides the MessageSender interface discussed above.

The [SDA](https://github.com/dedis/cothority) is a framework for Secure Distributed Algorithms, developped by DeDiS, EPFL. It help bootstrapping secure protocols. The "wrapper" is simply the link between this framework and our library `PriFi-lib` (which does not know at all about `sda`).

To sum up, the architecture is as follow :

```
######################
#      PriFi-Lib     # <--- this can be instanciated as client, relay, etc.
######################
         ^
         |
         v
###################### <--- this box is the SDA, provided by DeDiS
#    SDA-Protocol    # <--- (also called PriFi-SDA-Wrapper)
#         ^          #
#         |          #
#         v          #
#    SDA-Service     #
#         ^          #
#         |          #
#      SDA-App       # <--- started by the CLI
######################
```

### SOCKS

PriFi anonymizes the traffic via SOCKS proxy. Once PriFi is running, you can configure your SOCKS client (e.g. browser, mail application) to connect to PriFi.

The structure is a big convoluted : we have *two* socks servers. One is *in* the PriFi client code; that's the entry point of your upstream traffic, e.g. your browser connects to the socks server *in* PriFi on your local machine.

Then, PriFi anonymizes the traffic with the help of the other clients and the relay. The anonymized traffic is outputted at the relay.

This anonymized traffic is *SOCKS traffic*. Hence, the relay needs to connect to the second SOCKS server, which is not related to PriFi (but we provide the code for it in `socks/`). It could also be a remote, public SOCKS server.
