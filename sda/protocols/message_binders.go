package protocols

import "gopkg.in/dedis/onet.v1/log"

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiExchangeProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_ALL_ALL_PARAMETERS forwards an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_ALL_ALL_PARAMETERS_NEW(msg Struct_ALL_ALL_PARAMETERS_NEW) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_PARAMETERS_NEW)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_PK forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_TRU_REL_TELL_PK(msg Struct_TRU_REL_TELL_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_TRUSTEES_PK forwards an REL_CLI_TELL_TRUSTEES_PK message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_REL_CLI_TELL_TRUSTEES_PK(msg Struct_REL_CLI_TELL_TRUSTEES_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_TRUSTEES_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiExchangeProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK_1(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK_1) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK_1)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	if endStep == true {
		p.WhenFinished()
		p.Done()
		log.LLvl4("Done")
	}
	return err

}

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiScheduleProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_ALL_ALL_PARAMETERS forwards an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_ALL_ALL_PARAMETERS_NEW(msg Struct_ALL_ALL_PARAMETERS_NEW) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_PARAMETERS_NEW)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_DOWNSTREAM_DATA forwards an REL_CLI_DOWNSTREAM_DATA message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_CLI_DOWNSTREAM_DATA(msg Struct_REL_CLI_DOWNSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_DOWNSTREAM_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG forwards an REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(msg Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_TRUSTEES_PK forwards an REL_CLI_TELL_TRUSTEES_PK message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_CLI_TELL_TRUSTEES_PK(msg Struct_REL_CLI_TELL_TRUSTEES_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_TRUSTEES_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK_2(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK_2) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK_2)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_CLI_REL_UPSTREAM_DATA(msg Struct_CLI_REL_UPSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_UPSTREAM_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_CLI_REL_CLI_REL_OPENCLOSED_DATA(msg Struct_CLI_REL_OPENCLOSED_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_OPENCLOSED_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_DC_CIPHER forwards an TRU_REL_DC_CIPHER message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_DC_CIPHER(msg Struct_TRU_REL_DC_CIPHER) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_DC_CIPHER)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_SHUFFLE_SIG forwards an TRU_REL_SHUFFLE_SIG message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_SHUFFLE_SIG(msg Struct_TRU_REL_SHUFFLE_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_SHUFFLE_SIG)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS forwards an TRU_REL_TELL_NEW_BASE_AND_EPH_PKS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_PK forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_TRU_REL_TELL_PK(msg Struct_TRU_REL_TELL_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(msg Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_TRANSCRIPT forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_TRU_TELL_TRANSCRIPT(msg Struct_REL_TRU_TELL_TRANSCRIPT) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_TRANSCRIPT)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_RATE_CHANGE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiScheduleProtocol) Received_REL_TRU_TELL_RATE_CHANGE(msg Struct_REL_TRU_TELL_RATE_CHANGE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_RATE_CHANGE)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiCommunicateProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_ALL_ALL_PARAMETERS forwards an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_ALL_ALL_PARAMETERS_NEW(msg Struct_ALL_ALL_PARAMETERS_NEW) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_PARAMETERS_NEW)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_DOWNSTREAM_DATA forwards an REL_CLI_DOWNSTREAM_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_CLI_DOWNSTREAM_DATA(msg Struct_REL_CLI_DOWNSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_DOWNSTREAM_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG forwards an REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(msg Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_CLI_TELL_TRUSTEES_PK forwards an REL_CLI_TELL_TRUSTEES_PK message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_CLI_TELL_TRUSTEES_PK(msg Struct_REL_CLI_TELL_TRUSTEES_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_TRUSTEES_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK_1(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK_1) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK_1)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK_2(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK_2) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK_2)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_UPSTREAM_DATA(msg Struct_CLI_REL_UPSTREAM_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_UPSTREAM_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_CLI_REL_CLI_REL_OPENCLOSED_DATA(msg Struct_CLI_REL_OPENCLOSED_DATA) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_OPENCLOSED_DATA)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_DC_CIPHER forwards an TRU_REL_DC_CIPHER message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_TRU_REL_DC_CIPHER(msg Struct_TRU_REL_DC_CIPHER) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_DC_CIPHER)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_SHUFFLE_SIG forwards an TRU_REL_SHUFFLE_SIG message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_TRU_REL_SHUFFLE_SIG(msg Struct_TRU_REL_SHUFFLE_SIG) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_SHUFFLE_SIG)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS forwards an TRU_REL_TELL_NEW_BASE_AND_EPH_PKS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_TRU_REL_TELL_PK forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_TRU_REL_TELL_PK(msg Struct_TRU_REL_TELL_PK) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_PK)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(msg Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_TRANSCRIPT forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_TRU_TELL_TRANSCRIPT(msg Struct_REL_TRU_TELL_TRANSCRIPT) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_TRANSCRIPT)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}

//Received_REL_TRU_TELL_RATE_CHANGE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiCommunicateProtocol) Received_REL_TRU_TELL_RATE_CHANGE(msg Struct_REL_TRU_TELL_RATE_CHANGE) error {
	endStep, state, err := p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_RATE_CHANGE)
	log.LLvl4("Err: ", err, " endStep: ", endStep, " state: ", state)
	return err
}
