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
######################                                                          ######################
# PriFi-Lib (client) # <--- this can be instanciated as client, relay, etc.     #  PriFi-Lib (relay)
######################                                                          ######################
         ^                                                                                ^
         |                                                                                |
         v                                                                                v
###################### <--- this box is the SDA, provided by DeDiS              ######################
#    SDA-Protocol    # <--- (also called PriFi-SDA-Wrapper)                     #    SDA-Protocol
#         ^          #                                                          #         ^      
#         |          #                                                          #         |  
#         v          #                                                          #         v    
#    SDA-Service     #                                                          #    SDA-Service 
#         ^          #                                                          #         ^  
#         |          #                                                          #         |  
#      SDA-App       # <--- started by ./prifi.sh                               #      SDA-App 
######################                                                          #######################
^^^^^^^^^^^^^^^^^^^^^^                                                          ^^^^^^^^^^^^^^^^^^^^^
	    HOST 1        <================ COMMUNICATES WITH  ====================>       HOST 2
vvvvvvvvvvvvvvvvvvvvvv                                                          vvvvvvvvvvvvvvvvvvv

```

### SOCKS

PriFi anonymizes the traffic via SOCKS proxy. Once PriFi is running, you can configure your SOCKS client (e.g. browser, mail application) to connect to PriFi.

The structure is a big convoluted : we have *two* socks servers. One is *in* the PriFi client code; that's the entry point of your upstream traffic, e.g. your browser connects to the socks server *in* PriFi on your local machine.

Then, PriFi anonymizes the traffic with the help of the other clients and the relay. The anonymized traffic is outputted at the relay.

This anonymized traffic is *SOCKS traffic*. Hence, the relay needs to connect to the second SOCKS server, which is not related to PriFi (but we provide the code for it in `socks/`). It could also be a remote, public SOCKS server.

To sum up, without SOCKS proxy, your browser does not use its SOCKS-client, and connects directly to the internet :

```
.______________.
| Browser      |<--------------------------------------------------> Internet
|              | 
| Socks-Client | <- (unused)
|______________|

```

You can set up your browser to connect to a SOCKS server, in that case it will use its SOCKS client :


```

.______________.
| Browser      |
|              | 
| Socks-Client | <--------------------------------------------------> SOCKS-Server
|______________|                                                           ^
                                                                           |
                                                                           v
                                                                        Internet


```

Finally, using PriFi, the architecture is as follow :

```

._____________.
| Browser     |           PriFi Client
|             |         ._______________.
| Socks-Client| <------>| SOCKS-Server 1| 
|_____________|         |       ^       |
                        |       |       |              PriFi Relay
                        |       v       |            ._______________.
                        | Anonymization | <--------> | Anonymization | 
                        |_______________|            |       ^       |
                                                     |       |       |
                                                     |       v       |
                                                     |  SOCKS-Client | <--->  SOCKS-Server 2
                                                     |_______________|             ^
                                                                                   |
                                                                                   v
                                                                                Internet

^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
     On your machine (localhost)        |            |           On the PriFi relay
```

You could also decide not to use the SOCKS server we provide in `socks/`, and connect to a remote, public server :


```

._____________.
| Browser     |           PriFi Client
|             |         ._______________.
| Socks-Client| <------>| SOCKS-Server 1| 
|_____________|         |       ^       |
                        |       |       |              PriFi Relay
                        |       v       |            ._______________.
                        | Anonymization | <--------> | Anonymization | 
                        |_______________|            |       ^       |
                                                     |       |       |
                                                     |       v       |
                                                     |  SOCKS-Client | <--------------->  Public SOCKS Server
                                                     |_______________|                             ^
                                                                                                   |
                                                                                                   v
                                                                                                Internet

^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^            ^^^^^^^^^^^^^^^^^^^^^^^^^            ^^^^^^^^^^^^^^^^^^^^
     On your machine (localhost)        |            |   On the PriFi relay  |            |    Any machine  
```