# PriFi: A Low-Latency, Tracking-Resistant Protocol for Local-Area Anonymity [![Build Status](https://travis-ci.org/lbarman/prifi.svg?branch=master)](https://travis-ci.org/lbarman/prifi)

[back to main README](README.md)

## More details on ./prifi.sh

### tl;dr

Run everything (including the SOCKS proxy #2) in localhost with :
```
./prifi.sh all-localhost
```
... and sets your browser to connect to the SOCKS server `127.0.0.1:8081` or `127.0.0.1:8082`

## Running PriFi :

### SOCKS Preamble

As explained, you need a non-prifi SOCKS server running to handle the traffic from the relay. If you don't have one, run ours :
```
cd ./socks && ./run-socks-proxy.sh 8090
```
(you don't need to do this if you run `./prifi.sh all-localhost`, it is done for you)

## Running PriFi

There is one big startup script `prifi.sh`. 

```
./prifi.sh

PriFi, a tracking-resistant protocol for local-area anonymity

Usage: run-prifi.sh role/operation [params]
	role: client, relay, trustee
	operation: install, sockstest, all-localhost, gen-id
	params for role relay: [socks_server_port] (optional, numeric)
	params for role trustee: id (required, numeric)
	params for role client: id (required, numeric), [prifi_socks_server_port] (optional, numeric)
	params for operation install: none
	params for operation all-localhost: none
	params for operation gen-id: none
	params for operation sockstest: [socks_server_port] (optional, numeric), [prifi_socks_server_port] (optional, numeric)

Man-page:
	install: get the dependencies, and tests the setup
	relay: starts a PriFi relay
	trustee: starts a PriFi trustee, using the config file trusteeid
	client: starts a PriFi client, using the config file clientid
	all-localhost: starts a Prifi relay, a trustee, three clients all on localhost
	sockstest: starts the PriFi and non-PriFi SOCKS tunnel, without PriFi anonymization
	gen-id: interactive creation of identity.toml
	Lost ? read https://github.com/lbarman/prifi/README.md

```

For instance, you can start a relay like this : 

```
./prifi.sh relay 8090
```

You can start a client like this :

```
./prifi.sh client 0
```

and to specify the port of the first socks proxy integrated in PriFi :

```
./prifi.sh client 0 8080
```

A typical deployement could be :

```
./prifi.sh relay 8090
./prifi.sh trustee 0
./prifi.sh client 0 8080
./prifi.sh client 1 8081
```

## Default or Real identities.

An *identity*, stored in `identity.toml`, is a private/public key pair. By default, PriFi will use pre-generated identities in `config/identities_default`.

You should not use those for a real PriFi development, as the private key is public (and thus you literally get 0 security).

You can generate your own `identity.toml` by calling `./prifi.sh gen-id`. It will create a fresh identity in `config/identities_real/{entity}/`.

In `prifi.sh`, there is a variable `try_use_real_identities`. If `false`, the script will always fetch identities in `config/identities_default/{entity}/`. If `true`, the script will fetch identities in `config/identities_real/{entity}/` if they exist, and fall back to `config/identities_default/{entity}/` otherwise.


## Configuration

The PriFi configuration file is in `config/prifi.toml`

 - `CellSizeUp (int)` : Size of upstream data sent in one PriFi round
 - `CellSizeDown (int)` : Size of downstream data sent in one PriFi round
 - `RelayWindowSize (int)` : Number of in-flight, non-acknowledged ciphers
 - `RelayUseDummyDataDown (bool)` : If true, data-down is always equal to CellSizeDown. Otherwise, it is as small as 1 bit.
 - `RelayReportingLimit (int)` : If -1, no limit. Otherwise, the relay shutdowns after this amount of rounds.
 - `UseUDP (bool)` : Whether the relay uses UDP for broadcast or not
 - `DoLatencyTests` : Whether the clients do latency tests when they have nothing to send
 - `SocksServerPort (int)` : The port number of the SOCKS Server 1, in PriFi
 - `SocksClientPort (int)` : The port number of the SOCKS Server 2, outside PriFi

[back to main README](README.md)