# PriFi: Low-Latency Tracking-Resistant Mobile Computing

### Table of Contents

1. [Introduction](#Introduction) 
2. [Running the System](#Running-the-System)

    2.1. [Compiling PriFi](#Compiling-PriFi)

	2.2. [Initial Configuration](#Initial-Configuration)

	2.3. [Running a Node](#Running-a-Node)

3. [Protocol Description](#Protocol-Description)
4. [Equivocation Protection](#Equivocation)
5. [Disruption Protection](#Disruption)
6. [Coding Style](#Code-Style)
7. [References](#References)

<a id="Introduction"></a>
## 1. Introduction

PriFi is an anonymous communication protocol with provable traffic-analysis resistance and small latency suitable for wireless networks. This provides a network access mechanism for protecting members of an organization who access the Internet while on-site (via privacy-preserving WiFi networking) and while off-site (via privacy-preserving virtual private networking or VPN). The small latency cost is acheved by leveraging the client-relay-server topology common in WiFi networks. Main entities of PriFi are: relay, trustee server (or Trustees), and clients. These collaborate to implement a Dining Cryptographer's network (DC-Net) that can anonymize the client upstream traffic. The relay is a WiFi router that can process normal TCP/IP traffic in addition to running our protocol

<a id="Running-the-System"></a>
## 2. Running the System

<a id="Compiling-PriFi"></a>
### 2.1. Compiling PriFi

Use the following command to compile the project:

	go build -o prifi main.go

Run this in the project main directory. This will create an executable file named "prifi" that can run ant type of node (client, relay, or trustee).

<a id="Initial-Configuration"></a>
### 2.2. Initial Configuration

After compiling the program, run the following command to create configuration directories for each node:

	prifi -config

This will create configuration data for a default setting, which consists of one client, one trustee server and one relay. To generate configuration data for a specific setting, run:

 prifi -config -nclients=3 -ntrustees=2

The configuraton generator will create one directory for each node in the local users directory. Depending on the node's type, its config directory will be named with the following format:

	prifi-client-<client's sequence number>
	prifi-trustee-<trustee's sequence number>
	prifi-relay

For example, for the setting with 3 clients and 2 trustees, five directories will be created:

	prifi-client-0
	prifi-client-1
	prifi-client-2
	prifi-trustee-0
	prifi-trustee-1
	prifi-relay

For each client and trustee, the config directory will contain two files:

1. config.tml: A human-readable TOML-format file with the node's configuration information;
2. A .sec file containing the node's secret key.

For the relay, the config directory will contain a the above two files as well as a file named "roster" which will contain a roster of all public keys of all clients and trustees.

<a id="Running-a-Node"></a>
### 2.3. Running a Node

A node can be run using the following command:

 `prifi -node=<name of the node>`

The name of the node is set automatically by the configuration generator and is euqal to the name of the node's configuration directory (see Initial Configuration section).

<a id="Protocol-Description"></a>
## 3. Protocol Description

We define downstream communication as the data from the Internet to one of the clients, and upstream communication as the data from one of the clients towards the Internet. The downstream data is simply broadcasted from the relay to every client. The upstream d ata goes through PriFi's anonymization protocol and is sent by the relay towards the Internet. We refer to each time the clients anonymously sending a messages to the relay as a round.

PriFi consists of three main protocols: setup, scheduling, and anonymization. In the setup protocol, each client agrees with each servers on a shared secret, which is known to both of them but is secret to others. This secret is then used to seed a pseudo-random generator to obtain a stream of pseudo-random bits from which the clients and the servers will compute their ciphertexts.

The scheduling protocol is run in each round to determine which client gets to transmit his message in which round. The scheduling information needs to remain secret to all entities, as otherwise it can completely break the anonymity of the clients. In PriFi, the servers randomly and verifiably shuffle (using [Neff03]) a set of public keys corresponding to the clients.

The secret permutation is then sent to all clients each of whom is only able to recognize his own public key in the sequence; other keys look unrelated to anyone without the associated private key. In the anonymization protocol, every client sends a ciphertext to the relay; one of the ciphertexts contains the message M to be sent to the Internet, and the rest contain empty messages. Every server also sends a ciphertext to the relay. The relay then participates in a distributed protocol jointly with the servers to obtain M from the collected ciphertexts.

<a id="Authentication"></a>
### 3.1. Authentication

When a node (client or trustee) comes online, he sends an authentication request to the relay. This request contains the client's unique ID and the client's preferred method of authentication. PriFi currently supports three authentication methods: basic, anonymous, and trust-on-first-use (TOFU).

We assume the relay already knows the public keys of all nodes who want to join using the basic or the anonymous authentication method. We refer to these as long-term public key. Through these authentication methods, the node proves to the relay that he possesses the corresponding private key.

<a id="Basic-Authentication"></a>
### 3.1.1. Basic Authentication

 This protocol is similar to SSH using the Schnorr's signature scheme [Schnorr91]:

1. The relay sends a random message (challenge) to the node. The challenge is ElGamal-encrypted using the node's long-term public key;

2. The node decrypts the challenge, signs it with its private key using Schnorr's scheme, and sends the signature to the relay;

3. The relay verifies the signature and responds with an accept/reject message.

<a id="Anonymous-Authentication"></a>
### 3.1.2. Anonymous Authentication

PriFi prevents intersection attacks such as "who is online?" using an anonymous authentication scheme called Deniable Anonymous Group Authentication (DAGA) [SPW+14], wherein members of the organization prove their membership without divulging their identity (i.e., their long-term public keys).

The following protocols are run between a client, the relay, and a group of trustees:

#### Authentication setup

1. The relay sends the number of clients with long-term public keys to all trustees;
2. The j-th trustee generates a per-round secret r_j and sends a commitment R_j = g^r_j to other trustees;
3. For each client i, the trustee generates a per-round random generator h_i collectively with other trustees;

#### Client authentication

1. A client sends an authentication request to the relay;
2. The relay sends the IP/port address of the first trustees to the client;
3. The client connects to that trustee and requests an authentication context;
4. The trustee sends (H,p,g) to the client;
5. The client computes an initial linkage tag T_0 and proves in zero-knowledge to the trustee that he has correctly computed T_0, and that he knows one of the long-term private (via "OR" proof);
6. The trustee sends T_1 to the next trustee along with a proof that he has correctly computed T_1 and knows r_1;
7. Once the last trustee computes the final linkage tag T_f, he sends it to the first trustee;
8. The first trustee sends T_f to the client.

The client's proof is the interactive protocol of Camenisch and Stadler [CS97]. The trustee's proof is a non-interactive protocol based on Schnorr's proof of knowledge of discrete logarithms [Schnorr91] and proof of equality of discrete logarithms [CP92].

<a id="Equivocation"></a>
## 4. Equivocation Protection
An untrusted relay can preform equivocation attacks by sending different
(inconsistent) downstream messages to the clients to de-anonymize them.
For example, in an unencrypted communication, the relay can slightly modify the
downstream message for each client, thereby sending a unique message to each of
them.

These unique messages affect the requests that clients send in subsequent
rounds; so the relay may be able, in these subsequent rounds, to determine
which client sent each request.

In PriFi, the owner encrypts its upstream message with a random key and
includes the blinded key in its message to the relay to let its open the
encrypted message later. The key is blinded with the client's downstream
history, which itself is blinded with the secrets that the client shares with
the servers.

The client does not blind the history to hide it from the relay as it already
knows what it has sent to the client. The client does this to "bind" its
version of the history to its shared secrets, so that when it is combined with
other clients' blinded history as well as the servers' pieces of the shared
secrets, the key can be opened.
The message is encrypted by XORing it with the key. Finally, the key is blinded
by multiplying it with the blinded history.

This algorithm is implemented in `dcnet.ownedCoder` struct.

<a id="Disruption"></a>
## 5. Disruption Protection
In order to prevent disruption, the anonymous slot owner and trustees introduce
an additional trap layer to the owner's cleartext message.  The slot owner
shares a secret with each trustee to produce an additional ciphertext stream.
The secret derives from the slot owner's DH key revealed during the shuffle and
a per-interval DH key provided by the trustee prior to the start of an
interval.  The client produces two seeds: one for generating ciphertext and
another for selecting a trap bits.  Both seeds consist of a hash of the shared
secrets, cell index, and a 0 for generating ciphertexts or a 1 for selecting
trap bits -- hash(secret_1 | ... | secret_n | cell index | 0/1).  The client
picks one bit out of every n-bits to be a trap bit.  The trap bit remains
unchanged while every other bit is set to 0.

After selecting the trap bits, the client embeds messages without modifying the
trap bits.  To do so, the client splits his cleartext message into n-bit blocks
and prepends a header equal with the number of bits equal to the number of
n-bit blocks.  Each bit in the header belongs to the set of n-bits at the same
index within the message.  The header bit is used as an inversion flag.  If the
flag is 0, then the data can be stored without toggling the trap bit.
Otherwise he chooses a 1 bit and uses the complement of those n-bits in order
to avoid toggling the trap bit.

At the end of an interval, the relay transmits the output of each exchange to
the trustees.  The trustees then reveal their trap secrets in order to
determine the trap bits.  If no trap bits have been triggered, they continue on
to the next interval.  If a trap bit has been triggered, the trustees perform
the blame analysis as described in Dissent in Numbers.

<a id="Coding-Style"></a>
## 6. Coding Style

#### Gofmt
 We use Gofmt which is the official formating style for Go. Gofmt automatically formats Go source code, and thus there's no need to spend time lining up the code or think how many spaces are needed between math operators. Tis formatting style is recommended by [Effective Go](https://golang.org/doc/effective_go.html#formatting)  More details on how to use [Gofmt](https://blog.golang.org/go-fmt-your-code).

#### Line Width
The number of characters per line should be 80. Queries and regular expressions may go beyond this for obvious reasons. Gofmt uses tabs for indentation so feel free to change your editor's tab length if 80 characters is still large for your screen.

####Comments
Comments should start with a capital letter and there's one space after // and /*. For example:

	// Prepare crypto parameters
	rand := config.CryptoSuite.Cipher([]byte(nodeState.Name))

 Every exported (public) function should have at least one line of comment right before the function signature describing what the function is supposed to do.

<a id="References"></a>
## 7. References
* [CP92] http://link.springer.com/chapter/10.1007%2F3-540-48071-4_7
* [CS97] ftp://ftp.inf.ethz.ch/pub/crypto/publications/CamSta97b.pdf
* [Neff03] http://freehaven.net/anonbib/cache/shuffle:ccs01.pdf
* [Schnorr91] http://link.springer.com/article/10.1007%2FBF00196725
* [SPW+14] http://cpsc.yale.edu/sites/default/files/files/TR1486.pdf
