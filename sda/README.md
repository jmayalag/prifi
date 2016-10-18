# PriFi app with an example protocol

This branch contains an example app that can start the PriFi service in the three different modes, client, trustee or
relay. Instead of running the PriFi protocol it executes a naive protocol to serve as an example to familiarize with the
project.

The example protocol is started when the relay node is started and will try to send a message containing
"Hello children !" to all the other nodes in it's `group.toml` config file (which will be children of the relay node in
the SDA communication tree).

To get familiar with the code read the not-too-long `app/prifi.go`, `services/service.go` and `protocols/protocol.go`
source files.

To quickly run the code a script and confguration files for the relay, one client and one trustee are included in
`prifi_run/`. Simply make sure to start the client and the trustee before the relay as it will send it's message on
startup and then stay idle forever. In `prifi_run/` type:

```
./run.sh client 1
./run.sh trustee 1
./run.sh relay
```

You should see the received messages on the standard output of the client and trustee processes.