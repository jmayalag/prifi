# Demo 1 : PriFi @ lbarman.ch

On lbarman.ch, we set up the following entities : a relay, one trustee, and a socks proxy exit. With the proper configuration, you can simply indicate your local PriFi client to connect to it, and it should work.

_warning_ : expect very bad performance, because :

1. this is an alpha version that is very slow and not optimized at all

2. this server is not on a LAN, but on the internet (further degrading the experience).

## preamble : install PriFi

This has been covered by other READMEs, so I will just recap what to do :

```
go get github.com/lbarman/prifi
./prifi.sh install
```

## set up your PriFi client

Step 1. Change `./prifi.sh` and set `try_use_real_identities="true"`. You might also want to put `dbg_lvl=1`, at `3` you will not have time to read anything.

Step 2. Run `./prifi.sh gen-id`, press `c` for client, and `0` for the 0-th client. Press enter a few times. Override file if asked.

You just generated a public/private key pair for PriFi.

Step 3. Replace the contents of `config/identities_real/client0/group.toml` by the following :

```
Description = "PriFi"

[[servers]]
  Address = "tcp://46.101.101.102:7000"
  Public = "t4QidTLKGE947E7nLm2PmyLqvn6+dzI1BtNsvHSHp2U="
  Description = "relay"
```

Step 4. Run the PriFi client `./prifi.sh client 0`. Connect your browser to the SOCKS server at `localhost:8080`.