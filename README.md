# PriFi synchronization

This branch contains the code that will allow participants to
dynamically join or leave PriFi. The synchronization is done by
exchanging messages between services. The clients and trustees can send
`ConnectionRequest` and `DisconnectionRequest` messages to the relay 
services which handles them with the `HandleConnection` and
`HandleDisconnection` methods.

The strategy that is currently is very naive; the PriFi protocol is
restarted as soon as one node connects or disconnects.

## Running the code

To run the code configuration files and a helper script are provided in
`sda/prifi_run`. From this folder, we can run

```
./run.sh relay
```

to start the relay. We can now start a trustee and two clients to allow
the protocol to start:

```
./run.sh trustee 0
./run.sh client 0
./run.sh client 1
```

We can finally start a third client to see the protocol restart:

```
./run.sh client 2
```
