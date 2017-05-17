package utils

import (
	"github.com/Lukasa/gopcap"
	"os"
	"math/rand"
	"gopkg.in/dedis/onet.v1/log"
)

type Packet struct {
	ID uint32
	TimeSent int64 //microseconds
	Data []byte
}

// Parses a .pcap file, and returns all valid packets. A packet is (ID, TimeSent [micros], Data)
func ParsePCAP(path string) []Packet {
	pcapfile, err := os.Open("demo.pcap")
	if err != nil {
		log.Fatal("Cannot open", path, "error is", err)
	}
	parsed, err := gopcap.Parse(pcapfile)
	if err != nil {
		log.Fatal("Cannot parse", path, "error is", err)
	}

	out := make([]Packet, 0)

	if len(parsed.Packets) == 0 {
		return out
	}

	timeDelta := parsed.Packets[0].Timestamp.Nanoseconds()
	for id, pkt := range parsed.Packets {

		p := Packet{
			ID: uint32(id),
			Data: getPayloadOrRandom(pkt),
			TimeSent: (pkt.Timestamp.Nanoseconds()-timeDelta)/1000,
		}

		//basic sanity check
		if p.TimeSent > 0 && len(p.Data) != 0 {
			out = append(out, p)
		}

	}

	return out
}

func getPayloadOrRandom(pkt gopcap.Packet) []byte {
	len := pkt.IncludedLen

	if pkt.Data == nil {
		return randomBytes(len)
	}

	return pkt.Data.LinkData().InternetData().TransportData()
}

func randomBytes(len uint32) []byte {
	if len == uint32(0) {
		return make([]byte, 0)
	}
	out := make([]byte, len)
	written, err := rand.Read(out)
	if err == nil {
		log.Fatal("Could not generate a random packet of length", len, "error is", err)
	}
	if uint32(written) != len {
		log.Fatal("Could not generate a random packet of length", len, "only wrote", written)
	}
	return out
}