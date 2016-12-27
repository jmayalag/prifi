package protocols

//Received_ALL_ALL_SHUTDOWN shuts down the PriFi-lib if it is running
func (p *PriFiSDAProtocol) Received_ALL_ALL_SHUTDOWN(msg Struct_ALL_ALL_SHUTDOWN) error {
	p.Stop()
	err := p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_SHUTDOWN)
	return err
}

//Received_ALL_ALL_PARAMETERS forwards an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_ALL_ALL_PARAMETERS(msg Struct_ALL_ALL_PARAMETERS) error {
	return p.prifiLibInstance.ReceivedMessage(msg.ALL_ALL_PARAMETERS)
}

//Received_REL_CLI_DOWNSTREAM_DATA forwards an REL_CLI_DOWNSTREAM_DATA message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_CLI_DOWNSTREAM_DATA(msg Struct_REL_CLI_DOWNSTREAM_DATA) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_DOWNSTREAM_DATA)
}

//Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG forwards an REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG(msg Struct_REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_EPH_PKS_AND_TRUSTEES_SIG)
}

//Received_REL_CLI_TELL_TRUSTEES_PK forwards an REL_CLI_TELL_TRUSTEES_PK message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_CLI_TELL_TRUSTEES_PK(msg Struct_REL_CLI_TELL_TRUSTEES_PK) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_CLI_TELL_TRUSTEES_PK)
}

//Received_CLI_REL_TELL_PK_AND_EPH_PK forwards an CLI_REL_TELL_PK_AND_EPH_PK message to PriFi's lib
func (p *PriFiSDAProtocol) Received_CLI_REL_TELL_PK_AND_EPH_PK(msg Struct_CLI_REL_TELL_PK_AND_EPH_PK) error {
	return p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_TELL_PK_AND_EPH_PK)
}

//Received_CLI_REL_UPSTREAM_DATA forwards an CLI_REL_UPSTREAM_DATA message to PriFi's lib
func (p *PriFiSDAProtocol) Received_CLI_REL_UPSTREAM_DATA(msg Struct_CLI_REL_UPSTREAM_DATA) error {
	return p.prifiLibInstance.ReceivedMessage(msg.CLI_REL_UPSTREAM_DATA)
}

//Received_TRU_REL_DC_CIPHER forwards an TRU_REL_DC_CIPHER message to PriFi's lib
func (p *PriFiSDAProtocol) Received_TRU_REL_DC_CIPHER(msg Struct_TRU_REL_DC_CIPHER) error {
	return p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_DC_CIPHER)
}

//Received_TRU_REL_SHUFFLE_SIG forwards an TRU_REL_SHUFFLE_SIG message to PriFi's lib
func (p *PriFiSDAProtocol) Received_TRU_REL_SHUFFLE_SIG(msg Struct_TRU_REL_SHUFFLE_SIG) error {
	return p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_SHUFFLE_SIG)
}

//Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS forwards an TRU_REL_TELL_NEW_BASE_AND_EPH_PKS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS(msg Struct_TRU_REL_TELL_NEW_BASE_AND_EPH_PKS) error {
	return p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_NEW_BASE_AND_EPH_PKS)
}

//Received_TRU_REL_TELL_PK forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_TRU_REL_TELL_PK(msg Struct_TRU_REL_TELL_PK) error {
	return p.prifiLibInstance.ReceivedMessage(msg.TRU_REL_TELL_PK)
}

//Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE(msg Struct_REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_CLIENTS_PKS_AND_EPH_PKS_AND_BASE)
}

//Received_REL_TRU_TELL_TRANSCRIPT forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_TRU_TELL_TRANSCRIPT(msg Struct_REL_TRU_TELL_TRANSCRIPT) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_TRANSCRIPT)
}

//Received_REL_TRU_TELL_RATE_CHANGE forward an ALL_ALL_PARAMETERS message to PriFi's lib
func (p *PriFiSDAProtocol) Received_REL_TRU_TELL_RATE_CHANGE(msg Struct_REL_TRU_TELL_RATE_CHANGE) error {
	return p.prifiLibInstance.ReceivedMessage(msg.REL_TRU_TELL_RATE_CHANGE)
}
