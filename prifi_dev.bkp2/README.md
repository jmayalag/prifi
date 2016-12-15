Disclamer : this is a temporary README, copy-pasted from an internal discussion. We should clean it up.

# ReadMe

## Base

The concept are described in the README in the master branch. Here, only the code changes.

The essential change in the new code is going from a sequential code to an event-based one; in the new code, messages are event, and can be received at any time; the state of the entity is kept manually, and event handler can decide if a message is valid at a certain point or not.
Switching to an event-based code require some code to process the network messages (at least read and dispatch them), so we conveniently used the SDA-framework (part of the Cothority project) to provide with node creation, message dispatching and decoding.

New code :
https://github.com/lbarman/prifi_dev/tree/PriFi-SDA

## Structure

The new code is organized in two main parts :

1) PriFi-Lib, which is network-agnostic; it takes an interface "MessageSender" that give it functions like SendToRelay(), SendToTrustee, ... and ReceivedMessage()

This help developing PriFi, as without the network, the protocol becomes much simpler (at least 50% less line of codes); I hope we can develop new functionalities without knowing anything about the network, or SDA

this code is located in https://github.com/lbarman/prifi_dev/tree/PriFi-SDA/lib/prifi 

2) PriFi-SDA-Wrapper, that does the mapping between the tree entities of SDA and our roles (Relay, Trustee, etc), and provides the MessageSender interface discussed above.

This binder now uses SDA, which is very convenient, but could use any library. In particular, it *could* use SDA *and* direct TCP/UDP streams in parallel for performance reasons. For now, simply using SDA is great, we will check the performances later.

this code is located in https://github.com/lbarman/prifi_dev/tree/PriFi-SDA/protocols/prifi

## Running

To run the new code, simply run 

./prifi.sh <DEBUG_LVL>

... in the main folder, where DEBUG_LVL is 1,2, or 3. It's a shortcut to start a SDA-simulation in localhost, that creates some SDA-nodes; then the PriFi-SDA-Wrapper protocol is started by SDA, and it assigns the relay, trustees, clients; to some nodes; Finally, the PriFi-Lib is called by the PriFi-SDA-Wrapper (by artificially sending a first message to the relay).

​The mentioned simulation (which contains the number of clients, trustees, etc) is defined in https://github.com/lbarman/prifi_dev/blob/PriFi-SDA/simul/runfiles/prifi_simple.toml

## Debugging

Actually DEBUG_LVL goes to 5, where it also prints the SDA messages

## Implementing new behaviors

To define a new message type, go to https://github.com/lbarman/prifi_dev/blob/PriFi-SDA/simul/runfiles/prifi_simple.toml

A message is simply a struct, with some field. Please keep the terminology SOURCE_DESTINATION_MESSAGE_CONTENT for the name of the message.

Then, we need to tell PriFi-Lib how to handle the message; this is done in the same file, at line 116, where there is a big switch. The PriFi-SDA-Wrapper calls this function for all messages that arrive for this host.
There is no switch (Relay|Trustee|Client) for most of the message, as the name of the message and the name of the handler is explicit about who handle it. 
