package protocols

import "gopkg.in/dedis/onet.v1/log"

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiExchangeProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_ALL_ALL_PARAMETERS forwards an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_ALL_ALL_PARAMETERS_NEW(msg Struct_ALL_ALL_PARAMETERS_NEW) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_PARAMETERS_NEW)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_PK forward an TRU_REL_TELL_PK message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_TRU_REL_TELL_PK(msg Struct_TRU_REL_TELL_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_PK)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_TRUSTEES_PK forwards an REL_CLI_TELL_TRUSTEES_PK message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_REL_CLI_TELL_TRUSTEES_PK(msg Struct_REL_CLI_TELL_TRUSTEES_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_TRUSTEES_PK)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	if endStep == true {
		p.WhenFinished(p.prifiLibInstance)
		p.Done()
		log.Lvl4("Done")
	}
	return err

}

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiScheduleProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_SERVICE_REL_TELL_PK_AND_EPH_PK forwards an SERVICE_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_SERVICE_REL_TELL_PK_AND_EPH_PK(msg Struct_SERVICE_REL_TELL_PK_AND_EPH_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.SERVICE_REL_TELL_PK_AND_EPH_PK)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE forward an REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(msg Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS forwards an TRU_REL_TELL_NEW_BASE_AND_EPH_PKS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_TRANSCRIPT forward an REL_TRU_TELL_TRANSCRIPT message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_TRU_TELL_TRANSCRIPT(msg Struct_REL_TRU_TELL_TRANSCRIPT) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_TRANSCRIPT)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_SHUFFLE_SIG forwards an TRU_REL_SHUFFLE_SIG message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_SHUFFLE_SIG(msg Struct_TRU_REL_SHUFFLE_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_SHUFFLE_SIG)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	if endStep == true {
		p.WhenFinished(p.prifiLibInstance)
		p.Done()
		log.Lvl4("Done")
	}
	return err
}

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiCommunicateProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_SERVICE_REL_SHUFFLE_SIG forwards an SERVICE_REL_SHUFFLE_SIG message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_SERVICE_REL_SHUFFLE_SIG(msg Struct_SERVICE_REL_SHUFFLE_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.SERVICE_REL_SHUFFLE_SIG)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG forwards an REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(msg Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_UPSTREAM_DATA(msg Struct_CLI_REL_UPSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_UPSTREAM_DATA)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_DOWNSTREAM_DATA forwards an REL_CLI_DOWNSTREAM_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_CLI_DOWNSTREAM_DATA(msg Struct_REL_CLI_DOWNSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_DOWNSTREAM_DATA)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_CLI_REL_OPENCLOSED_DATA forwards an CLI_REL_OPENCLOSED_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_CLI_REL_OPENCLOSED_DATA(msg Struct_CLI_REL_OPENCLOSED_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_OPENCLOSED_DATA)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_DC_CIPHER forwards an TRU_REL_DC_CIPHER message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_TRU_REL_DC_CIPHER(msg Struct_TRU_REL_DC_CIPHER) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_DC_CIPHER)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_READY forward an REL_TRU_TELL_READY message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_TRU_TELL_READY(msg Struct_REL_TRU_TELL_READY) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_READY)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_RATE_CHANGE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_TRU_TELL_RATE_CHANGE(msg Struct_REL_TRU_TELL_RATE_CHANGE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_RATE_CHANGE)
	log.Lvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}
