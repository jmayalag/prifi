--------------------------------------------------------------------------------
PriFi: Low-Latency Tracking-Resistant Mobile Computing
--------------------------------------------------------------------------------

   Table Of Contents:

      1. Introduction
      2. Running the system
         2.1. Compiling PriFi
         2.2. Initial configuration
         2.3. Running a node
      3. Protocol description (brief)
         3.1. Authentication
      4. Code structure
      5. Coding style
      6. References

1. Introduction
---------------
   
   PriFi is an anonymous communication protocol with provable traffic-analysis resistance and 
   small latency suitable for wireless networks. This provides a network access mechanism for 
   protecting members of an organization who access the Internet while on-site (via 
   privacy-preserving WiFi networking) and while off-site (via privacy-preserving virtual private 
   networking or VPN). The small latency cost is acheved by leveraging the client-relay-server 
   topology common in WiFi networks.
   
   Main entities of PriFi are: relay, trustee server (or Trustees), and clients. These
   collaborate to implement a Dining Cryptographer's network (DC-Net) that can anonymize the client
   upstream traffic. The relay is a WiFi router that can process normal TCP/IP traffic in
   addition to running our protocol

2. Running the system
---------------------

2.1. Compiling PriFi

   Use the following command to compile the project:

		go build -o prifi main.go

   Run this in the project main directory. This will create an executable file named "prifi" that
   can run ant type of node (client, relay, or trustee).

2.2. Initial configuration

   After compiling the program, run the following command to create configuration directories for
   each node:

		prifi -config

   This will create configuration data for a default setting, which consists of one client, one
   trustee server and one relay. To generate configuration data for a specific setting, run:

		prifi -config -nclients=3 -ntrustees=2

   The configuraton generator will create one directory for each node in the local users directory.
   Depending on the node's type, its config directory will be named with the following format:

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
   (1) config.tml: A human-readable TOML-format file with the node's configuration information;
   (2) a .sec file containing the node's secret key.

   For the relay, the config directory will contain a the above two files as well as a file named
   "roster" which will contain a roster of all public keys of all clients and trustees.

2.3. Running a node

   A node can be run using the following command:

		prifi -node=<name of the node>

   The name of the node is set automatically by the configuration generator and is euqal to the name
   of the node's configuration directory (see Initial Configuration section).
   
3. Protocol description (brief)
-------------------------------

   We define downstream communication as the data from the Internet to one of the clients, and
   upstream communication as the data from one of the clients towards the Internet. The
   downstream data is simply broadcasted from the relay to every client. The upstream d ata goes
   through PriFi's anonymization protocol and is sent by the relay towards the Internet. We refer
   to each time the clients anonymously sending a messages to the relay as a round.

   PriFi consists of three main protocols: setup, scheduling, and anonymization. In the setup
   protocol, each client agrees with each servers on a shared secret, which is known to both of
   them but is secret to others. This secret is then used to seed a pseudo-random generator to
   obtain a stream of pseudo-random bits from which the clients and the servers will compute
   their ciphertexts.

   The scheduling protocol is run in each round to determine which client gets to transmit his
   message in which round. The scheduling information needs to remain secret to all entities, as
   otherwise it can completely break the anonymity of the clients. In PriFi, the servers randomly
   and verifiably shuffle (using [Neff03]) a set of public keys corresponding to the clients.

   The secret permutation is then sent to all clients each of whom is only able to recognize his
   own public key in the sequence; other keys look unrelated to anyone without the associated
   private key. In the anonymization protocol, every client sends a ciphertext to the relay; one
   of the ciphertexts contains the message M to be sent to the Internet, and the rest contain
   empty messages. Every server also sends a ciphertext to the relay. The relay then participates
   in a distributed protocol jointly with the servers to obtain M from the collected ciphertexts.

3.1. Authentication

   When a node (client or trustee) comes online, he sends an authentication request to the relay.
   This request contains the client's unique ID and the client's preferred method of
   authentication. PriFi currently supports three authentication methods: basic, anonymous, and
   trust-on-first-use (TOFU).

   We assume the relay already knows the public keys of all nodes who want to join using the basic
   or the anonymous authentication method. We refer to these as long-term public key. Through these
   authentication methods, the node proves to the relay that he posseses the corresponding
   private key.

3.1.1 Basic authentication

   This protocol is similar to SSH using the Schnorr's signature scheme [Schnorr91]:

   (1) The relay sends a random message (challenge) to the node. The challenge is ElGamal-encrypted
       using the node's long-term public key;

   (2) The node decrypts the challenge, signs it with its private key using Schnorr's scheme, and
       sends the signature to the relay;

   (3) The relay verifies the signature and responds with an accept/reject message.

3.1.2. Anonymous authentication

   PriFi prevents intersection attacks such as "who is online?" using an anonymous authentication
   scheme called Deniable Anonymous Group Authentication (DAGA) [SPW+14], wherein members of the
   organization prove their membership without divulging their identity (i.e., their long-term
   public keys).

   The following protocol is run between a client, the relay, and a group of trustees:

   (1) The relay sends the public key of one of the trustees to the client;

   (2) The client sends to the trustee an initial linkage tag, T0, and proves in zero-knowledge
       that he has correctly computed T0, and that he knows one of the long-term private (via
       "OR" proof).

   (3) The trustee verifies the proof and computes T1 by removing s1 from T0 and adding r1
       to T0. He then sends T1 to the next trustee along with a proof that he has correctly
       computed T1 and knows r1.

   The client's proof is the interactive protocol of Camenisch and Stadler [CS97].
   The trustee's proof is a non-interactive protocol based on Schnorr's proof of knowledge of
   discrete logarithms [Schnorr91] and proof of equality of discrete logarithms [CP92].

3.1.3. Trust on first use

   TBD

4. Code structure
-----------------

   TBD

5. Coding style
---------------

   Gofmt:
      We use Gofmt which is the official formating style for Go. Gofmt automatically formats Go
      source code, and thus there's no need to spend time lining up the code or think how many 
      spaces are needed between math operators. Tis formatting style is recommended by 
      Effective Go: https://golang.org/doc/effective_go.html#formatting
       
      More details on how to use Gofmt: https://blog.golang.org/go-fmt-your-code
       
   Line width:
      The preferred number of characters per line is 120. This makes coding in laptops easier.
      One or two characters beyond 120 is fine. Queries and regular expressions may go beyond
      this for obvious reasons. Gofmt uses tabs for indentation so feel free to change your
      editor's tab length if 120 characters is still large for your screen.
        
   Comments:
      Comments should start with a capital letter and there's one space after // and /*.
      For example:

      // Prepare crypto parameters
      rand := config.CryptoSuite.Cipher([]byte(nodeState.Name))

      Every exported (public) function should have at least one line of comment right before
      the function signature describing what the function is supposed to do.

6. References
-------------

		[CP92] http://link.springer.com/chapter/10.1007%2F3-540-48071-4_7
		[CS97] ftp://ftp.inf.ethz.ch/pub/crypto/publications/CamSta97b.pdf
		[Neff03] http://freehaven.net/anonbib/cache/shuffle:ccs01.pdf
		[Schnorr91] http://link.springer.com/article/10.1007%2FBF00196725
		[SPW+14] http://cpsc.yale.edu/sites/default/files/files/TR1486.pdf

   
