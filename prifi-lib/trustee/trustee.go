package trustee

/*
PriFi Trustee
************
This regroups the behavior of the PriFi trustee.
Needs to be instantiated via the PriFiProtocol in prifi.go
Then, this file simple handle the answer to the different message kind :

- ALL_ALL_PARAMETERS - (specialized into ALL_TRU_PARAMETERS) - used to initialize the relay over the network / overwrite its configuration
- REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE - the client's identities (and ephemeral ones), and a base. We react by Neff-Shuffling and sending the result
- REL_TRU_TELL_TRANSCRIPT - the Neff-Shuffle's results. We perform some checks, sign the last one, send it to the relay, and follow by continously sending ciphers.
- REL_TRU_TELL_RATE_CHANGE - Received when the relay requests a sending rate change, the message contains the necessary information needed to perform this change
*/

import (
	"errors"
	"github.com/dedis/cothority/log"
	"github.com/dedis/crypto/abstract"
	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"github.com/lbarman/prifi/prifi-lib/net"
	"github.com/lbarman/prifi/prifi-lib/scheduler"
	"reflect"
	"strconv"
	"time"
)

// Possible states the trustees are in. This restrict the kind of messages they can receive at a given point in time.
const (
	TRUSTEE_STATE_BEFORE_INIT int16 = iota
	TRUSTEE_STATE_INITIALIZING
	TRUSTEE_STATE_SHUFFLE_DONE
	TRUSTEE_STATE_READY
	TRUSTEE_STATE_SHUTDOWN
)

// Possible sending rates for the trustees.
const (
	TRUSTEE_KILL_SEND_PROCESS int16 = iota // kills the goroutine responsible for sending messages
	TRUSTEE_RATE_ACTIVE
	TRUSTEE_RATE_STOPPED
)

// TRUSTEE_BASE_SLEEP_TIME is the base unit for how much time the trustee sleeps between sending ciphers to the relay.
const TRUSTEE_BASE_SLEEP_TIME = 10 * time.Millisecond

// PriFiLibTrusteeInstance contains the mutable state of a PriFi entity.
type PriFiLibTrusteeInstance struct {
	messageSender *net.MessageSenderWrapper
	trusteeState  TrusteeState
}

// NewPriFiClient creates a new PriFi client entity state.
// Note: the returned state is not sufficient for the PrFi protocol
// to start; this entity will expect a ALL_ALL_PARAMETERS message as
// first received message to complete it's state.
func NewPriFiTrustee(msgSender *net.MessageSenderWrapper) *PriFiLibTrusteeInstance {
	prifi := PriFiLibTrusteeInstance{
		messageSender: msgSender,
	}
	return &prifi
}

// NewPriFiClientWithState creates a new PriFi client entity state.
func NewPriFiTrusteeWithState(msgSender *net.MessageSenderWrapper, state *TrusteeState) *PriFiLibTrusteeInstance {
	prifi := PriFiLibTrusteeInstance{
		messageSender: msgSender,
		trusteeState:  *state,
	}
	return &prifi
}

// TrusteeState contains the mutable state of the trustee.
type TrusteeState struct {
	CellCoder        dcnet.CellCoder
	ClientPublicKeys []abstract.Point
	currentState     int16
	ID               int
	MessageHistory   abstract.Cipher
	Name             string
	nClients         int
	neffShuffle      *scheduler.NeffShuffleTrustee
	nTrustees        int
	PayloadLength    int
	privateKey       abstract.Scalar
	PublicKey        abstract.Point
	sendingRate      chan int16
	sharedSecrets    []abstract.Point
	TrusteeID        int
}

// NeffShuffleResult holds the result of the NeffShuffle,
// since it needs to be verified when we receive REL_TRU_TELL_TRANSCRIPT.
type NeffShuffleResult struct {
	base  abstract.Point
	pks   []abstract.Point
	proof []byte
}

/*
NewTrusteeState initializes the state of the trustee.
It must be called before anything else.
*/
func NewTrusteeState(trusteeID int, nClients int, nTrustees int, payloadLength int) *TrusteeState {
	params := new(TrusteeState)

	params.ID = trusteeID
	params.Name = "Trustee-" + strconv.Itoa(trusteeID)
	params.CellCoder = config.Factory()
	params.nClients = nClients
	params.nTrustees = nTrustees
	params.PayloadLength = payloadLength
	params.sendingRate = make(chan int16, 10)
	params.TrusteeID = trusteeID

	//gen the key
	pub, priv := crypto.NewKeyPair()

	//generate own parameters
	params.privateKey = priv
	params.PublicKey = pub

	//init the neff shuffle
	neffShuffle := new(scheduler.NeffShuffle)
	neffShuffle.Init()
	params.neffShuffle = neffShuffle.TrusteeView
	params.neffShuffle.Init(trusteeID, priv, pub)

	//placeholders for pubkeys and secrets
	params.ClientPublicKeys = make([]abstract.Point, nClients)
	params.sharedSecrets = make([]abstract.Point, nClients)

	//sets the new state
	params.currentState = TRUSTEE_STATE_INITIALIZING

	return params
}

/*
Received_ALL_TRU_SHUTDOWN handles ALL_REL_SHUTDOWN messages.
When we receive this message we should  clean up resources.
*/
func (p *PriFiLibTrusteeInstance) Received_ALL_TRU_SHUTDOWN(msg net.ALL_ALL_SHUTDOWN) error {
	log.Lvl1("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received a SHUTDOWN message. ")

	//stop the sending process
	p.trusteeState.sendingRate <- TRUSTEE_KILL_SEND_PROCESS

	p.trusteeState.currentState = TRUSTEE_STATE_SHUTDOWN

	return nil
}

/*
Received_ALL_TRU_PARAMETERS handles ALL_REL_PARAMETERS.
It initializes the trustee with the parameters contained in the message.
*/
func (p *PriFiLibTrusteeInstance) Received_ALL_TRU_PARAMETERS(msg net.ALL_ALL_PARAMETERS_NEW) error {

	//this can only happens in the state RELAY_STATE_BEFORE_INIT
	if p.trusteeState.currentState != TRUSTEE_STATE_BEFORE_INIT && !msg.ForceParams {
		log.Lvl1("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received a ALL_ALL_PARAMETERS, but not in state TRUSTEE_STATE_BEFORE_INIT, ignoring. ")
		return nil
	} else if p.trusteeState.currentState != TRUSTEE_STATE_BEFORE_INIT && msg.ForceParams {
		log.Lvl1("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received a ALL_ALL_PARAMETERS && ForceParams = true, processing. ")
	} else {
		log.Lvl3("Trustee : received ALL_ALL_PARAMETERS")
	}

	startNow := msg.BoolValueOrElse("StartNow", false)
	nextFreeTrusteeID := msg.IntValueOrElse("NextFreeTrusteeID", -1)
	nTrustees := msg.IntValueOrElse("NTrustees", p.trusteeState.nTrustees)
	nClients := msg.IntValueOrElse("nClients", p.trusteeState.nClients)
	upCellSize := msg.IntValueOrElse("UpstreamCellSize", p.trusteeState.PayloadLength) //todo: change this name

	p.trusteeState = *NewTrusteeState(nextFreeTrusteeID, nClients, nTrustees, upCellSize)

	if startNow {
		// send our public key to the relay
		p.Send_TRU_REL_PK()
	}

	p.trusteeState.currentState = TRUSTEE_STATE_INITIALIZING

	log.Lvlf5("%+v\n", p.trusteeState)
	log.Lvl2("Trustee " + strconv.Itoa(p.trusteeState.ID) + " has been initialized by message. ")
	return nil
}

/*
Send_TRU_REL_PK tells the relay's public key to the relay
(this, of course, provides no security, but this is an early version of the protocol).
This is the first action of the trustee.
*/
func (p *PriFiLibTrusteeInstance) Send_TRU_REL_PK() error {
	toSend := &net.TRU_REL_TELL_PK{TrusteeID: p.trusteeState.ID, Pk: p.trusteeState.PublicKey}
	p.messageSender.SendToRelayWithLog(toSend, "")
	return nil
}

/*
Send_TRU_REL_DC_CIPHER sends DC-net ciphers to the relay continuously once started.
One can control the rate by sending flags to "rateChan".
*/
func (p *PriFiLibTrusteeInstance) Send_TRU_REL_DC_CIPHER(rateChan chan int16) {

	stop := false
	currentRate := TRUSTEE_RATE_ACTIVE
	roundID := int32(0)

	for !stop {
		select {
		case newRate := <-rateChan:
			if currentRate != newRate {
				log.Lvl2("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : rate changed from " + strconv.Itoa(int(currentRate)) + " to " + strconv.Itoa(int(newRate)))
				currentRate = newRate
			}

			if newRate == TRUSTEE_KILL_SEND_PROCESS {
				stop = true
			}

		default:
			if currentRate == TRUSTEE_RATE_ACTIVE {
				roundID, _ = sendData(p, roundID)
				time.Sleep(TRUSTEE_BASE_SLEEP_TIME)

			} else if currentRate == TRUSTEE_RATE_STOPPED {
				time.Sleep(TRUSTEE_BASE_SLEEP_TIME)

			} else {
				log.Lvl2("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : In unrecognized sending state")
			}

		}
	}
}

/*
Received_REL_TRU_TELL_RATE_CHANGE handles REL_TRU_TELL_RATE_CHANGE messages
by changing the cipher sending rate.
Either the trustee must stop sending because the relay is at full capacity
or the trustee sends normally because the relay has emptied up enough capacity.
*/
func (p *PriFiLibTrusteeInstance) Received_REL_TRU_TELL_RATE_CHANGE(msg net.REL_TRU_TELL_RATE_CHANGE) error {

	if msg.WindowCapacity == 0 {
		p.trusteeState.sendingRate <- TRUSTEE_RATE_STOPPED
	} else {
		p.trusteeState.sendingRate <- TRUSTEE_RATE_ACTIVE
	}

	return nil
}

/*
sendData is an auxiliary function used by Send_TRU_REL_DC_CIPHER. It computes the DC-net's cipher and sends it.
It returns the new round number (previous + 1).
*/
func sendData(p *PriFiLibTrusteeInstance, roundID int32) (int32, error) {
	data := p.trusteeState.CellCoder.TrusteeEncode(p.trusteeState.PayloadLength)

	//send the data
	toSend := &net.TRU_REL_DC_CIPHER{
		RoundID:   roundID,
		TrusteeID: p.trusteeState.ID,
		Data:      data}
	p.messageSender.SendToRelayWithLog(toSend, "(round "+strconv.Itoa(int(roundID))+")")

	return roundID + 1, nil
}

/*
Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE handles REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE messages.
Those are sent when the connection to a relay is established.
They contain the long-term and ephemeral public keys of the clients,
and a base given by the relay. In addition to deriving the secrets,
the trustee uses the ephemeral keys to perform a Neff shuffle. It remembers
this shuffle in order to check the correctness of the chain of shuffle afterwards.
*/
func (p *PriFiLibTrusteeInstance) Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(msg net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE) error {

	//this can only happens in the state TRUSTEE_STATE_INITIALIZING
	if p.trusteeState.currentState != TRUSTEE_STATE_INITIALIZING {
		e := "Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received a REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE, but not in state TRUSTEE_STATE_INITIALIZING, in state " + strconv.Itoa(int(p.trusteeState.currentState))
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE")

	//begin parsing the message
	clientsPks := msg.Pks
	clientsEphemeralPks := msg.EphPks

	//sanity check
	if len(clientsPks) < 1 || len(clientsEphemeralPks) < 1 || len(clientsPks) != len(clientsEphemeralPks) {
		e := "Trustee " + strconv.Itoa(p.trusteeState.ID) + " : One of the following check failed : len(clientsPks)>1, len(clientsEphemeralPks)>1, len(clientsPks)==len(clientsEphemeralPks)"
		log.Error(e)
		return errors.New(e)
	}

	//only at this moment we really learn the number of clients
	nClients := len(clientsPks)
	p.trusteeState.nClients = nClients
	p.trusteeState.ClientPublicKeys = make([]abstract.Point, nClients)
	p.trusteeState.sharedSecrets = make([]abstract.Point, nClients)

	//fill in the clients keys
	for i := 0; i < len(clientsPks); i++ {
		p.trusteeState.ClientPublicKeys[i] = clientsPks[i]
		p.trusteeState.sharedSecrets[i] = config.CryptoSuite.Point().Mul(clientsPks[i], p.trusteeState.privateKey)
	}

	toSend, err := p.trusteeState.neffShuffle.ReceivedShuffleFromRelay(msg.Base, msg.EphPks, true)
	if err != nil {
		return errors.New("Could not do ReceivedShuffleFromRelay, error is " + err.Error())
	}

	//send the answer
	p.messageSender.SendToRelayWithLog(toSend, "")

	//change state
	p.trusteeState.currentState = TRUSTEE_STATE_SHUFFLE_DONE

	return nil
}

/*
Received_REL_TRU_TELL_TRANSCRIPT handles REL_TRU_TELL_TRANSCRIPT messages.
Those are sent when all trustees have already shuffled. They need to verify all the shuffles, and also that
their own shuffle has been included in the chain of shuffles. If that's the case, this trustee signs the *last*
shuffle (which will be used by the clients), and sends it back to the relay.
If everything succeed, starts the goroutine for sending DC-net ciphers to the relay.
*/
func (p *PriFiLibTrusteeInstance) Received_REL_TRU_TELL_TRANSCRIPT(msg net.REL_TRU_TELL_TRANSCRIPT) error {

	//this can only happens in the state TRUSTEE_STATE_SHUFFLE_DONE
	if p.trusteeState.currentState != TRUSTEE_STATE_SHUFFLE_DONE {
		e := "Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received a REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE, but not in state TRUSTEE_STATE_SHUFFLE_DONE, in state " + strconv.Itoa(int(p.trusteeState.currentState))
		log.Error(e)
		return errors.New(e)
	}
	log.Lvl3("Trustee " + strconv.Itoa(p.trusteeState.ID) + " : Received_REL_TRU_TELL_TRANSCRIPT")

	// PROTOBUF FLATTENS MY 2-DIMENSIONAL ARRAY. THIS IS A PATCH
	a := msg.EphPks
	b := make([][]abstract.Point, p.trusteeState.nTrustees)
	if len(a) > p.trusteeState.nTrustees {
		for i := 0; i < p.trusteeState.nTrustees; i++ {
			b[i] = make([]abstract.Point, p.trusteeState.nClients)
			for j := 0; j < p.trusteeState.nClients; j++ {
				v := a[i*p.trusteeState.nTrustees+j][0]
				b[i][j] = v
			}
		}
		msg.EphPks = b
	} else {
		log.Print("Probably the Protobuf lib has been patched ! you might remove this code.")
	}
	// END OF PATCH

	toSend, err := p.trusteeState.neffShuffle.ReceivedTranscriptFromRelay(msg.Bases, msg.EphPks, msg.Proofs)
	if err != nil {
		return errors.New("Could not do ReceivedTranscriptFromRelay, error is " + err.Error())
	}

	//send the answer
	p.messageSender.SendToRelayWithLog(toSend, "")

	//we can forget our shuffle
	//p.trusteeState.neffShuffleToVerify = NeffShuffleResult{base2, ephPublicKeys2, proof}

	//change state
	p.trusteeState.currentState = TRUSTEE_STATE_READY

	//everything is ready, we start sending
	go p.Send_TRU_REL_DC_CIPHER(p.trusteeState.sendingRate)

	return nil
}

// ReceivedMessage must be called when a PriFi host receives a message.
// It takes care to call the correct message handler function.
func (p *PriFiLibTrusteeInstance) ReceivedMessage(msg interface{}) error {

	var err error

	switch typedMsg := msg.(type) {
	case net.ALL_ALL_PARAMETERS_NEW:
		err = p.Received_ALL_TRU_PARAMETERS(typedMsg) //todo change this name
	case net.ALL_ALL_SHUTDOWN:
		err = p.Received_ALL_TRU_SHUTDOWN(typedMsg)
	case net.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE:
		err = p.Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(typedMsg)
	case net.REL_TRU_TELL_TRANSCRIPT:
		err = p.Received_REL_TRU_TELL_TRANSCRIPT(typedMsg)
	case net.REL_TRU_TELL_RATE_CHANGE:
		err = p.Received_REL_TRU_TELL_RATE_CHANGE(typedMsg)
	default:
		err = errors.New("Unrecognized message, type" + reflect.TypeOf(msg).String())
	}

	//no need to push the error further up. display it here !
	if err != nil {
		log.Error("ReceivedMessage: got an error, " + err.Error())
		return err
	}

	return nil
}
