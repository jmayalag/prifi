package client

/**
 * PriFi Client
 * ************
 * This regroups the behavior of the PriFi client.
 * Needs to be instantiated via the PriFiProtocol in prifi.go
 * Then, this file simple handle the answer to the different message kind :
 *
 * - ALL_ALL_SHUTDOWN - kill this client
 * - ALL_ALL_PARAMETERS (specialized into ALL_CLI_PARAMETERS) - used to initialize the client over the network / overwrite its configuration
 * - REL_CLI_TELL_TRUSTEES_PK - the trustee's identities. We react by sending our identity + ephemeral identity
 * - REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG - the shuffle from the trustees. We do some check, if they pass, we can communicate. We send the first round to the relay.
 * - REL_CLI_DOWNSTREAM_DATA - the data from the relay, for one round. We react by finishing the round (sending our data to the relay)
 *
 * local functions :
 *
 * ProcessDownStreamData() <- is called by Received_REL_CLI_DOWNSTREAM_DATA; it handles the raw data received
 * SendUpstreamData() <- it is called at the end of ProcessDownStreamData(). Hence, after getting some data down, we send some data up.
 *
 * TODO : traffic need to be encrypted
 */

import (
	"errors"
	"strconv"

	"github.com/lbarman/prifi/prifi-lib/config"
	"github.com/lbarman/prifi/prifi-lib/crypto"
	prifilog "github.com/lbarman/prifi/prifi-lib/log"
	"github.com/lbarman/prifi/prifi-lib/net"
	"gopkg.in/dedis/kyber.v2"
	"gopkg.in/dedis/onet.v2/log"

	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/lbarman/prifi/prifi-lib/dcnet"
	"github.com/lbarman/prifi/prifi-lib/scheduler"
	"github.com/lbarman/prifi/prifi-lib/utils"
	"github.com/lbarman/prifi/utils"
	"io"
	mathrand "math/rand"
	"time"
)

// Received_ALL_CLI_SHUTDOWN handles ALL_CLI_SHUTDOWN messages.
// When we receive this message, we should clean up resources.
func (p *PriFiLibClientInstance) Received_ALL_ALL_SHUTDOWN(msg net.ALL_ALL_SHUTDOWN) error {
	log.Lvl2("Client " + strconv.Itoa(p.clientState.ID) + " : Received a SHUTDOWN message. ")

	p.stateMachine.ChangeState("SHUTDOWN")

	return nil
}

// Received_ALL_CLI_PARAMETERS handles ALL_CLI_PARAMETERS messages.
// It uses the message's parameters to initialize the client.
func (p *PriFiLibClientInstance) Received_ALL_ALL_PARAMETERS(msg net.ALL_ALL_PARAMETERS) error {
	clientID := msg.IntValueOrElse("NextFreeClientID", -1)
	e := "Client " + strconv.Itoa(clientID)
	p.stateMachine.SetEntity(e)
	p.messageSender.SetEntity(e)
	nTrustees := msg.IntValueOrElse("NTrustees", p.clientState.nTrustees)
	nClients := msg.IntValueOrElse("NClients", p.clientState.nClients)
	payloadSize := msg.IntValueOrElse("PayloadSize", p.clientState.PayloadSize)
	useUDP := msg.BoolValueOrElse("UseUDP", p.clientState.UseUDP)
	dcNetType := msg.StringValueOrElse("DCNetType", "not initialized")
	disruptionProtection := msg.BoolValueOrElse("DisruptionProtectionEnabled", false)
	equivProtection := msg.BoolValueOrElse("EquivocationProtectionEnabled", false)
	downstreamTrafficEncrypted := msg.BoolValueOrElse("DownstreamTrafficEncrypted", false)

	//sanity checks
	if clientID < -1 {
		return errors.New("ClientID cannot be negative")
	}
	if nTrustees < 1 {
		return errors.New("nTrustees cannot be smaller than 1")
	}
	if nClients < 1 {
		return errors.New("nClients cannot be smaller than 1")
	}
	if payloadSize < 1 {
		return errors.New("PayloadSize cannot be 0")
	}

	switch dcNetType {
	case "Verifiable":
		panic("verifiable not supported yet")
	}

	//set the received parameters
	p.clientState.ID = clientID
	p.clientState.Name = "Client-" + strconv.Itoa(clientID)
	p.clientState.MySlot = -1
	p.clientState.nClients = nClients
	p.clientState.nTrustees = nTrustees
	p.clientState.PayloadSize = payloadSize
	p.clientState.UseUDP = useUDP
	p.clientState.TrusteePublicKey = make([]kyber.Point, nTrustees)
	p.clientState.sharedSecrets = make([]kyber.Point, nTrustees)
	p.clientState.RoundNo = int32(0)
	p.clientState.BufferedRoundData = make(map[int32]net.REL_CLI_DOWNSTREAM_DATA)
	p.clientState.MessageHistory = config.CryptoSuite.XOF([]byte("init")) //any non-nil, non-empty, constant array
	p.clientState.DisruptionProtectionEnabled = disruptionProtection
	p.clientState.EquivocationProtectionEnabled = equivProtection

	if downstreamTrafficEncrypted {
		// pick a key
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			panic(err)
		}

		// pick an IV
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			panic(err)
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			log.Fatal("Client: could not instantiate the stream cipher for downstream traffic", err)
		}

		stream := cipher.NewCTR(block, iv)

		settings := &DownstreamEncryptionSettings{
			EncryptionEnabled: true,
			Key:               key,
			IV:                iv,
			Cipher:            stream,
		}

		p.clientState.DownstreamEncryption = settings
	} else {
		p.clientState.DownstreamEncryption = &DownstreamEncryptionSettings{EncryptionEnabled: false}
	}

	//we know our client number, if needed, parse the pcap for replay
	if p.clientState.pcapReplay.Enabled {
		p.clientState.pcapReplay.PCAPFile = p.clientState.pcapReplay.PCAPFolder + "client" + strconv.Itoa(clientID) + ".pcap"
		packets, err := utils.ParsePCAP(p.clientState.pcapReplay.PCAPFile, p.clientState.PayloadSize)
		if err != nil {
			log.Lvl2("Client", clientID, "Requested PCAP Replay, but could not parse;", err)
		}
		p.clientState.pcapReplay.Packets = packets

		if len(packets) > 0 {
			offset := packets[0].MsSinceBeginningOfCapture
			log.Lvl1("Client", clientID, "loaded corresponding PCAP with", len(packets), "packets, offset", offset, "ms.")
		} else {
			log.Lvl1("Client", clientID, "loaded corresponding PCAP with 0packets.")
		}
	}

	//if by chance we had a broadcast-listener goroutine, kill it
	if p.clientState.StartStopReceiveBroadcast != nil {
		p.clientState.StartStopReceiveBroadcast <- false
	}
	p.clientState.StartStopReceiveBroadcast = make(chan bool, 10)

	//start the broadcast-listener goroutine
	if useUDP {
		go p.messageSender.MessageSender.ClientSubscribeToBroadcast(p.clientState.ID, p.ReceivedMessage, p.clientState.StartStopReceiveBroadcast)
	}

	log.Lvl2("Client " + strconv.Itoa(p.clientState.ID) + " has been initialized by message. ")

	// continue with handling the public keys
	p.Received_REL_CLI_TELL_TRUSTEES_PK(msg.TrusteesPks)

	return nil
}

/*
Received_REL_CLI_DOWNSTREAM_DATA handles REL_CLI_DOWNSTREAM_DATA messages which are part of PriFi's main loop.
This is what happens in one round, for this client. We receive some downstream data.
It should be encrypted, and we should test if this data is for us or not; if so, push it into the SOCKS/VPN chanel.
For now, we do nothing with the downstream data.
Once we received some data from the relay, we need to reply with a DC-net cell (that will get combined with other client's cell to produce some plaintext).
If we're lucky (if this is our slot), we are allowed to embed some message (which will be the output produced by the relay). Either we send something from the
SOCKS/VPN data, or if we're running latency tests, we send a "ping" message to compute the latency. If we have nothing to say, we send 0's.
*/
func (p *PriFiLibClientInstance) Received_REL_CLI_DOWNSTREAM_DATA(msg net.REL_CLI_DOWNSTREAM_DATA) error {

	if msg.RoundID == 1 {
		p.clientState.pcapReplay.time0 = uint64(MsTimeStampNow())
	}

	//check if it is in-order
	if msg.RoundID == p.clientState.RoundNo {
		//process downstream data
		return p.ProcessDownStreamData(msg)
	} else if msg.RoundID < p.clientState.RoundNo {
		log.Lvl3("Client " + strconv.Itoa(p.clientState.ID) + " : Received a REL_CLI_DOWNSTREAM_DATA for round " + strconv.Itoa(int(msg.RoundID)) + " but we are in round " + strconv.Itoa(int(p.clientState.RoundNo)) + ", discarding.")
	} else if msg.RoundID > p.clientState.RoundNo {
		log.Lvl3("Client "+strconv.Itoa(p.clientState.ID)+" : Skipping from round", p.clientState.RoundNo, "to round", msg.RoundID)
		p.clientState.RoundNo = msg.RoundID
		return p.ProcessDownStreamData(msg)
	}

	return nil
}

/*
Received_REL_CLI_UDP_DOWNSTREAM_DATA handles REL_CLI_UDP_DOWNSTREAM_DATA messages which are part of PriFi's main loop.
This is what happens in one round, for this client.
We receive some downstream data. It should be encrypted, and we should test if this data is for us or not; is so, push it into the SOCKS/VPN chanel.
For now, we do nothing with the downstream data.
Once we received some data from the relay, we need to reply with a DC-net cell (that will get combined with other client's cell to produce some plaintext).
If we're lucky (if this is our slot), we are allowed to embed some message (which will be the output produced by the relay). Either we send something from the
SOCKS/VPN data, or if we're running latency tests, we send a "ping" message to compute the latency. If we have nothing to say, we send 0's.
*/
func (p *PriFiLibClientInstance) Received_REL_CLI_UDP_DOWNSTREAM_DATA(msg net.REL_CLI_DOWNSTREAM_DATA_UDP) error {

	return p.Received_REL_CLI_DOWNSTREAM_DATA(msg.REL_CLI_DOWNSTREAM_DATA)
}

func (p *PriFiLibClientInstance) decryptDownstreamData(data []byte) []byte {
	if !p.clientState.DownstreamEncryption.EncryptionEnabled {
		return data
	}

	cipher := p.clientState.DownstreamEncryption.Cipher

	plaintext := make([]byte, len(data))
	cipher.XORKeyStream(plaintext, data)

	log.Lvl3("Client", p.clientState.ID, "): Decrypting from relay")
	log.Lvlf3("\n%+v", hex.Dump(data))
	log.Lvl3("---------------------")
	log.Lvlf3("\n%+v", hex.Dump(plaintext))

	return plaintext
}

/*
ProcessDownStreamData handles the downstream data. After determining if the data is for us (this is not done yet), we test if it's a
latency-test message, test if the resync flag is on (which triggers a re-setup).
When this function ends, it calls SendUpstreamData() which continues the communication loop.
*/
func (p *PriFiLibClientInstance) ProcessDownStreamData(msg net.REL_CLI_DOWNSTREAM_DATA) error {

	timing.StartMeasure("round-processing")

	/*
	 * HANDLE THE DOWNSTREAM DATA
	 */

	//if it's just one byte, no data. Otherwise, process
	if len(msg.Data) > 1 {

		decrypted := p.decryptDownstreamData(msg.Data)

		//pass the data to the VPN/SOCKS5 proxy, if enabled
		if p.clientState.DataOutputEnabled {
			p.clientState.DataFromDCNet <- decrypted
		}
		//test if it is the answer from our ping (for latency test)
		if p.clientState.LatencyTest.DoLatencyTests && len(decrypted) > 2 {

			actionFunction := func(roundRec int32, roundDiff int32, timeDiff int64) {
				log.Lvl3("Measured latency is", timeDiff, ", for client", p.clientState.ID, ", roundDiff", roundDiff, ", received on round", msg.RoundID)
				p.clientState.timeStatistics["measured-latency"].AddTime(timeDiff)
				p.clientState.timeStatistics["measured-latency"].ReportWithInfo("measured-latency")
			}
			prifilog.DecodeLatencyMessages(decrypted, p.clientState.ID, msg.RoundID, actionFunction)
		}
	}

	//test if we have latency test to send
	now := time.Now()
	if p.clientState.LatencyTest.DoLatencyTests && p.clientState.ID == 0 && now.After(p.clientState.LatencyTest.NextLatencyTest) {
		log.Lvl1("Client 0 wants to send a latency test")
		newLatTest := &prifilog.LatencyTestToSend{
			CreatedAt: now,
		}
		p.clientState.LatencyTest.LatencyTestsToSend = append(p.clientState.LatencyTest.LatencyTestsToSend, newLatTest)
		p.clientState.LatencyTest.NextLatencyTest = now.Add(p.clientState.LatencyTest.LatencyTestsInterval)
		p.clientState.LatencyTest.NextLatencyTest = p.clientState.LatencyTest.NextLatencyTest.Add(time.Duration(mathrand.Intn(1000)) * time.Millisecond)
	}

	//if the flag "Resync" is on, we cannot write data up, but need to resend the keys instead
	if msg.FlagResync == true {

		log.Lvl1("Client ", p.clientState.ID, "Relay wants to resync, going to state BEFORE_INIT ")
		p.stateMachine.ChangeState("BEFORE_INIT")

		//TODO : regenerate ephemeral keys ?

		return nil

		//if the flag FlagOpenClosedRequest
	} else if msg.FlagOpenClosedRequest == true {

		log.Lvl3("Client", p.clientState.ID, "Relay wants to open/closed schedule slots ")

		//do the schedule
		bmc := new(scheduler.BitMaskSlotScheduler_Client)
		bmc.Client_ReceivedScheduleRequest(p.clientState.nClients)

		//check if we want to transmit
		if p.WantsToTransmit() {
			bmc.Client_ReserveRound(p.clientState.MySlot)
			log.Lvl3("Client ", p.clientState.ID, "Gonna reserve slot", p.clientState.MySlot, "(we are in round", msg.RoundID, ")")
		}
		contribution := bmc.Client_GetOpenScheduleContribution()

		//produce the next upstream cell

		upstreamCell := p.clientState.DCNet.EncodeForRound(p.clientState.RoundNo, false, contribution)

		//send the data to the relay
		toSend := &net.CLI_REL_OPENCLOSED_DATA{
			ClientID:       p.clientState.ID,
			RoundID:        p.clientState.RoundNo,
			OpenClosedData: upstreamCell}
		p.messageSender.SendToRelayWithLog(toSend, "(round "+strconv.Itoa(int(p.clientState.RoundNo))+")")

	} else {
		//send upstream data for next round
		p.SendUpstreamData(msg.OwnershipID)
	}

	//clean old buffered messages
	delete(p.clientState.BufferedRoundData, int32(p.clientState.RoundNo-1))

	t := timing.StopMeasure("round-processing")
	timeMs := t.Nanoseconds() / 1e6
	//log.Lvl1("Client", p.clientState.ID, "Round", p.clientState.RoundNo, "duration", t)

	//one round just passed
	p.clientState.RoundNo++

	p.clientState.timeStatistics["round-processing"].AddTime(timeMs)
	p.clientState.timeStatistics["round-processing"].ReportWithInfo("round-processing")

	//now we will be expecting next message. Except if we already received and buffered it !
	if msg, hasAMessage := p.clientState.BufferedRoundData[int32(p.clientState.RoundNo)]; hasAMessage {
		p.Received_REL_CLI_DOWNSTREAM_DATA(msg)
	}

	return nil
}

// WantsToTransmit returns true if [we have a latency message to send] OR [we have data to send]
func (p *PriFiLibClientInstance) WantsToTransmit() bool {

	//we have some pcap to send
	if p.clientState.pcapReplay.Enabled && len(p.clientState.pcapReplay.Packets) > 0 && p.clientState.pcapReplay.currentPacket < len(p.clientState.pcapReplay.Packets) {
		relativeNow := uint64(MsTimeStampNow()) - p.clientState.pcapReplay.time0
		currentPacket := p.clientState.pcapReplay.Packets[p.clientState.pcapReplay.currentPacket]

		if currentPacket.MsSinceBeginningOfCapture <= relativeNow {
			return true
		}
	}

	// if we have a latency test message
	if len(p.clientState.LatencyTest.LatencyTestsToSend) > 0 {
		return true
	}

	// if we have already ready-to-send data
	if p.clientState.NextDataForDCNet != nil {
		return true
	}

	// if we transmitted in the last second, keep reserving (but don't do this with pcaps)
	if !p.clientState.pcapReplay.Enabled {
		now := time.Now()
		//if we transmitted in the last second, keep reserving slots
		if now.Before(p.clientState.LastWantToSend.Add(5 * time.Second)) {
			log.Lvl3("WantToSend < 5 sec,  true")
			return true
		}
	}

	// otherwise, poll the channel
	select {
	case myData := <-p.clientState.DataForDCNet:
		p.clientState.LastWantToSend = time.Now()
		p.clientState.NextDataForDCNet = &myData
		log.Lvl3("WantToSend has data, true")
		return true

	default:
		log.Lvl3("WantToSend           false")
		return false
	}
}

/*
SendUpstreamData determines if it's our round, embeds data (maybe latency-test message) in the payload if we can,
creates the DC-net cipher and sends it to the relay.
*/
func (p *PriFiLibClientInstance) SendUpstreamData(ownerSlotID int) error {

	var upstreamCellContent []byte

	//how much data we can send
	actualPayloadSize := p.clientState.PayloadSize
	if p.clientState.DisruptionProtectionEnabled {
		actualPayloadSize -= 32
		if actualPayloadSize <= 0 {
			log.Fatal("Client", p.clientState.ID, "Cannot have disruption protection with less than 32 bytes payload")
		}
	}

	//if we can send data
	slotOwner := false
	if ownerSlotID == p.clientState.MySlot {
		slotOwner = true
	}
	if slotOwner {

		//this data has already been polled out of the DataForDCNet chan, so send it first
		//this is non-nil when OpenClosedSlot is true, and that it had to poll data out
		if p.clientState.NextDataForDCNet != nil {
			upstreamCellContent = *p.clientState.NextDataForDCNet
			p.clientState.NextDataForDCNet = nil
		} else {

			//if there are some pcap packets to replay
			if p.clientState.pcapReplay.Enabled && p.clientState.pcapReplay.currentPacket < len(p.clientState.pcapReplay.Packets) {

				if p.clientState.pcapReplay.currentPacket >= len(p.clientState.pcapReplay.Packets)-2 {
					log.Fatal("Client", p.clientState.ID, "End of experiment, client sent all packets!")
				}
				//if it is time to send some packet
				relativeNow := uint64(MsTimeStampNow()) - p.clientState.pcapReplay.time0

				payload := make([]byte, 0)
				payloadRealLength := 0 // payload actually only contains the headers
				currentPacket := p.clientState.pcapReplay.Packets[p.clientState.pcapReplay.currentPacket]

				//all packets >= currentPacket AND <= relativeNow should be sent
				basePacketID := p.clientState.pcapReplay.currentPacket
				lastPacketID := p.clientState.pcapReplay.currentPacket
				for currentPacket.MsSinceBeginningOfCapture <= relativeNow && payloadRealLength+currentPacket.RealLength <= actualPayloadSize {

					// add this packet
					payload = append(payload, currentPacket.Header...)
					payloadRealLength += currentPacket.RealLength
					p.clientState.pcapReplay.currentPacket++
					currentPacket = p.clientState.pcapReplay.Packets[p.clientState.pcapReplay.currentPacket]
					lastPacketID = p.clientState.pcapReplay.currentPacket
				}
				totalPackets := len(p.clientState.pcapReplay.Packets)
				log.Lvl2("Client", p.clientState.ID, "Adding pcap packets", basePacketID, "-", lastPacketID, "/", totalPackets)
				if basePacketID%100 == 0 || basePacketID+10 > totalPackets {
					log.Lvl2("Client", p.clientState.ID, "PCAP: added pcap packets", basePacketID, "-", lastPacketID, "/", totalPackets)
				}

				upstreamCellContent = payload
			} else {

				select {

				//either select data from the data we have to send, if any
				case myData := <-p.clientState.DataForDCNet:
					upstreamCellContent = myData

				//or, if we have nothing to send, and we are doing Latency tests, embed a pre-crafted message that we will recognize later on
				default:
					upstreamCellContent = make([]byte, actualPayloadSize)

					if len(p.clientState.LatencyTest.LatencyTestsToSend) > 0 {

						logFn := func(timeDiff int64) {
							p.clientState.timeStatistics["latency-msg-stayed-in-buffer"].AddTime(timeDiff)
							p.clientState.timeStatistics["latency-msg-stayed-in-buffer"].ReportWithInfo("latency-msg-stayed-in-buffer")
						}

						bytes, outMsgs := prifilog.LatencyMessagesToBytes(p.clientState.LatencyTest.LatencyTestsToSend,
							p.clientState.ID, p.clientState.RoundNo, actualPayloadSize, logFn)

						p.clientState.LatencyTest.LatencyTestsToSend = outMsgs
						upstreamCellContent = bytes
					}
				}
			}

			//content := make([]byte, len(upstreamCellContent))
			//copy(content[:], upstreamCellContent[:])
			//p.clientState.DataHistory[p.clientState.RoundNo] = content
		}
	}

	//produce the next upstream cell
	var hmac []byte
	if p.clientState.DisruptionProtectionEnabled {
		hmac = p.computeHmac256(upstreamCellContent)
	}
	payload := append(hmac, upstreamCellContent...)
	upstreamCell := p.clientState.DCNet.EncodeForRound(p.clientState.RoundNo, slotOwner, payload)

	//send the data to the relay
	toSend := &net.CLI_REL_UPSTREAM_DATA{
		ClientID: p.clientState.ID,
		RoundID:  p.clientState.RoundNo,
		Data:     upstreamCell,
	}

	p.messageSender.SendToRelayWithLog(toSend, "(round "+strconv.Itoa(int(p.clientState.RoundNo))+")")

	return nil
}

func (p *PriFiLibClientInstance) computeHmac256(message []byte) []byte {
	key := []byte("client-secret" + strconv.Itoa(p.clientState.ID))
	h := hmac.New(sha256.New, key)
	h.Write(message)
	return h.Sum(nil)
}

/*
Received_REL_CLI_TELL_TRUSTEES_PK handles REL_CLI_TELL_TRUSTEES_PK messages. These are sent when we connect.
The relay sends us a pack of public key which correspond to the set of pre-agreed trustees.
Of course, there should be check on those public keys (each client need to trust one), but for now we assume those public keys belong indeed to the trustees,
and that clients have agreed on the set of trustees.
Once we receive this message, we need to reply with our Public Key (Used to derive DC-net secrets), and our Ephemeral Public Key (used for the Shuffle protocol)
*/
func (p *PriFiLibClientInstance) Received_REL_CLI_TELL_TRUSTEES_PK(trusteesPks []kyber.Point) error {

	// sanity check
	if len(trusteesPks) < 1 {
		e := "Client " + strconv.Itoa(p.clientState.ID) + " : len(msg.Pks) must be >= 1"
		log.Error(e)
		return errors.New(e)
	}

	p.clientState.TrusteePublicKey = make([]kyber.Point, p.clientState.nTrustees)
	p.clientState.sharedSecrets = make([]kyber.Point, p.clientState.nTrustees)

	for i := 0; i < len(trusteesPks); i++ {
		p.clientState.TrusteePublicKey[i] = trusteesPks[i]
		p.clientState.sharedSecrets[i] = config.CryptoSuite.Point().Mul(p.clientState.privateKey, trusteesPks[i])
	}

	p.clientState.DCNet = dcnet.NewDCNetEntity(p.clientState.ID,
		dcnet.DCNET_CLIENT, p.clientState.PayloadSize, p.clientState.EquivocationProtectionEnabled, p.clientState.sharedSecrets)

	// then, generate our ephemeral keys (used for shuffling)
	p.clientState.EphemeralPublicKey, p.clientState.ephemeralPrivateKey = crypto.NewKeyPair()

	var key []byte
	var IV []byte

	// if the downstream traffic should be encrypted, transmit the keys
	if p.clientState.DownstreamEncryption.EncryptionEnabled {
		key = p.clientState.DownstreamEncryption.Key
		IV = p.clientState.DownstreamEncryption.IV
	}

	// send the keys to the relay
	toSend := &net.CLI_REL_TELL_PK_AND_EPH_PK{
		ClientID:            p.clientState.ID,
		Pk:                  p.clientState.PublicKey,
		EphPk:               p.clientState.EphemeralPublicKey,
		DownstreamCipherKey: key,
		DownstreamCipherIV:  IV,
	}
	p.messageSender.SendToRelayWithLog(toSend, "")

	p.stateMachine.ChangeState("EPH_KEYS_SENT")

	return nil
}

/*
Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG handles REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG messages.
These are sent after the Shuffle protocol has been done by the Trustees and the Relay.
The relay is sending us the result, so we should check that the protocol went well :
1) each trustee announced must have signed the shuffle
2) we need to locate which is our slot
When this is done, we are ready to communicate !
As the client should send the first data, we do so; to keep this function simple, the first data is blank
(the message has no content / this is a wasted message). The actual embedding of data happens only in the
"round function", that is Received_REL_CLI_DOWNSTREAM_DATA().
*/
func (p *PriFiLibClientInstance) Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(msg net.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG) error {

	//verify the signature
	neff := new(scheduler.NeffShuffle)
	mySlot, err := neff.ClientVerifySigAndRecognizeSlot(p.clientState.ephemeralPrivateKey, p.clientState.TrusteePublicKey, msg.Base, msg.EphPks, msg.GetSignatures())

	if err != nil {
		e := "Client " + strconv.Itoa(p.clientState.ID) + "; Can't recognize our slot ! err is " + err.Error()
		log.Error(e)
	}

	//prepare for commmunication
	p.clientState.MySlot = mySlot
	p.clientState.RoundNo = int32(0)
	p.clientState.BufferedRoundData = make(map[int32]net.REL_CLI_DOWNSTREAM_DATA)

	//if by chance we had a broadcast-listener goroutine, kill it
	if p.clientState.UseUDP {
		if p.clientState.StartStopReceiveBroadcast == nil {
			e := "Client " + strconv.Itoa(p.clientState.ID) + " wish to start listening with UDP, but doesn't have the appropriate helper."
			log.Error(e)
			return errors.New(e)
		}
		p.clientState.StartStopReceiveBroadcast <- true
		log.Lvl3("Client", p.clientState.ID, "indicated the udp-helper to start listening.")
	}

	//change state
	p.stateMachine.ChangeState("READY")
	log.Lvl3("Client", p.clientState.ID, "ready to communicate.")

	//produce a blank cell (we could embed data, but let's keep the code simple, one wasted message is not much)
	data := make([]byte, p.clientState.PayloadSize)
	slotOwner := false
	if p.clientState.DisruptionProtectionEnabled {
		if p.clientState.PayloadSize < 32 {
			log.Fatal("Client", p.clientState.ID, "Cannot have disruption protection with less than 32 bytes payload")
		}
		data2 := make([]byte, p.clientState.PayloadSize-32)
		hmac := p.computeHmac256(data2)

		data = append(hmac, data2...)
	}
	if p.clientState.ID == 0 {
		slotOwner = true // we need one guy that takes the responsability for this first slot
	}

	upstreamCell := p.clientState.DCNet.EncodeForRound(0, slotOwner, data)

	//send the data to the relay
	toSend := &net.CLI_REL_UPSTREAM_DATA{
		ClientID: p.clientState.ID,
		RoundID:  p.clientState.RoundNo,
		Data:     upstreamCell,
	}
	p.messageSender.SendToRelayWithLog(toSend, "(round "+strconv.Itoa(int(p.clientState.RoundNo))+")")

	p.clientState.RoundNo++

	return nil
}
