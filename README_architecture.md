# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/dedis/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi) [![Go Report Card](https://goreportcard.com/badge/github.com/lbarman/prifi)](https://goreportcard.com/report/github.com/lbarman/prifi) [![Coverage Status](https://coveralls.io/repos/github/dedis/prifi/badge.svg?branch=master)](https://coveralls.io/github/dedis/prifi?branch=master)

[back to main README](README.md)

## Architecture, and SOCKS proxies

### Structure

The current code is organized in two main parts :

1) `PriFi-Lib`, which is network-agnostic; it takes an interface "MessageSender" that give it functions like `SendToRelay()`, `SendToTrustee()`, ... and `ReceivedMessage()`

This is the core of the protocol PriFi. 

2) `PriFi-SDA-Wrapper` (what is in folder `sda`), that does the mapping between the tree entities of SDA and our roles (Relay, Trustee, Client), and provides the MessageSender interface discussed above.

The [ONet](https://github.com/dedis/onet) is a framework for Secure Distributed Algorithms, developped by DeDiS, EPFL. It help bootstrapping secure protocols. The "wrapper" is simply the link between this framework and our library `PriFi-lib` (which does not know at all about `sda`).

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

This setting is decided globally by the relay, not on a per-client basis.

### SDA call stack

The call order is :

1) the sda/app is called by the user/scripts

2) the clients/trustees/relay start their services

3) the clients/trustees services use their autoconnect() function

4) when he decides so, the relay (via ChurnHandler) spawns a new protocol :

5) this file is called; in order :

5.1) init() that registers the messages

5.2) NewPriFiSDAWrapperProtocol() that creates a protocol (and contains the tree given by the service)

5.3) in the service, setConfigToPriFiProtocol() is called, which calls the protocol (this file) 's SetConfigFromPriFiService()

5.3.1) SetConfigFromPriFiService() calls both buildMessageSender() and registerHandlers()

5.3.2) SetConfigFromPriFiService() calls New[Relay|Client|Trustee]State(); at this point, the protocol is ready to run

6) the relay's service calls protocol.Start(), which happens here

7) on the other entities, steps 5-6) will be repeated when a new message from the prifi protocols comes

[back to main README](README.md)
