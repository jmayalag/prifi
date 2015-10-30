package main

import (
	"encoding/binary"
	"fmt"
	"github.com/lbarman/crypto/abstract"
	"github.com/dedis/prifi/dcnet"
	"io"
	"log"
	"net"
	"time"
	log2 "github.com/lbarman/prifi/log"
)

type Trustee struct {
	pubkey abstract.Point
}

type AnonSet struct {
	suite    abstract.Suite
	trustees []Trustee
}

// Periodic stats reporting
var begin = time.Now()
var report = begin
var numberOfReports = 0
var period, _ = time.ParseDuration("3s")
var totupcells = int64(0)
var totupbytes = int64(0)
var totdowncells = int64(0)
var totdownbytes = int64(0)

var parupcells = int64(0)
var parupbytes = int64(0)
var pardownbytes = int64(0)

func reportStatistics(reportingLimit int) bool {
	now := time.Now()
	if now.After(report) {
		duration := now.Sub(begin).Seconds()

		instantUpSpeed := (float64(parupbytes)/period.Seconds())

		fmt.Printf("@ %fs; cell %f (%f) /sec, up %f (%f) B/s, down %f (%f) B/s\n",
			duration,
			 float64(totupcells)/duration, float64(parupcells)/period.Seconds(),
			 float64(totupbytes)/duration, instantUpSpeed,
			 float64(totdownbytes)/duration, float64(pardownbytes)/period.Seconds())

			// Next report time
		parupcells = 0
		parupbytes = 0
		pardownbytes = 0

		//log2.BenchmarkFloat(fmt.Sprintf("cellsize-%d-upstream-bytes", payloadlen), instantUpSpeed)

		data := struct {
		    Experiment string
		    CellSize int
		    Speed float64
		}{
		    "upstream-speed-given-cellsize",
		    payloadlen,
		    instantUpSpeed,
		}

		log2.JsonDump(data)

		report = now.Add(period)
		numberOfReports += 1

		if(reportingLimit > -1 && numberOfReports >= reportingLimit) {
			return false
		}
	}

	return true
}

func startRelay(reportingLimit int) {
	tg := dcnet.TestSetup(nil, suite, factory, nclients, ntrustees)
	me := tg.Relay

	// Start our own local HTTP proxy for simplicity.
	/*
		go func() {
			proxy := goproxy.NewProxyHttpServer()
			proxy.Verbose = true
			println("Starting HTTP proxy")
			log.Fatal(http.ListenAndServe(":8888", proxy))
		}()
	*/

	lsock, err := net.Listen("tcp", bindport)
	if err != nil {
		panic("Can't open listen socket:" + err.Error())
	}

	// Wait for all the clients and trustees to connect
	ccli := 0
	ctru := 0
	csock := make([]net.Conn, nclients)
	tsock := make([]net.Conn, ntrustees)
	for ccli < nclients || ctru < ntrustees {
		fmt.Printf("Waiting for %d clients, %d trustees\n",
			nclients-ccli, ntrustees-ctru)

		conn, err := lsock.Accept()
		if err != nil {
			panic("Listen error:" + err.Error())
		}

		b := make([]byte, 1)
		n, err := conn.Read(b)
		if n < 1 || err != nil {
			panic("Read error:" + err.Error())
		}

		node := int(b[0] & 0x7f)
		if b[0]&0x80 == 0 && node < nclients {
			if csock[node] != nil {
				panic("Oops, client connected twice")
			}
			csock[node] = conn
			ccli++
		} else if b[0]&0x80 != 0 && node < ntrustees {
			if tsock[node] != nil {
				panic("Oops, trustee connected twice")
			}
			tsock[node] = conn
			ctru++
		} else {
			panic("illegal node number")
		}
	}
	println("All clients and trustees connected")

	// Create ciphertext slice buffers for all clients and trustees
	clisize := me.Coder.ClientCellSize(payloadlen)
	cslice := make([][]byte, nclients)
	for i := 0; i < nclients; i++ {
		cslice[i] = make([]byte, clisize)
	}
	trusize := me.Coder.TrusteeCellSize(payloadlen)
	tslice := make([][]byte, ntrustees)
	for i := 0; i < ntrustees; i++ {
		tslice[i] = make([]byte, trusize)
	}

	conns := make(map[int]chan<- []byte)
	downstream := make(chan connbuf)
	nulldown := connbuf{} // default empty downstream cell
	window := 2           // Maximum cells in-flight
	inflight := 0         // Current cells in-flight


	for {

		// Show periodic reports
		if(!reportStatistics(reportingLimit)) {
			println("Reporting limit matched; exiting the relay")
			break;
		}

		// See if there's any downstream data to forward.
		var downbuf connbuf
		select {
			case downbuf = <-downstream: // some data to forward downstream
				//fmt.Println("Downstream data...")
				//fmt.Printf("v %d\n", len(downbuf)-6)
			default: // nothing at the moment to forward
				downbuf = nulldown
		}
		dlen := len(downbuf.buf)
		dbuf := make([]byte, 6+dlen)
		binary.BigEndian.PutUint32(dbuf[0:4], uint32(downbuf.cno))
		binary.BigEndian.PutUint16(dbuf[4:6], uint16(dlen))
		copy(dbuf[6:], downbuf.buf)

		// Broadcast the downstream data to all clients.
		for i := 0; i < nclients; i++ {
			//fmt.Printf("client %d -> %d downstream bytes\n",
			//		i, len(dbuf)-6)
			n, err := csock[i].Write(dbuf)
			if n != 6+dlen {
				panic("Write to client: " + err.Error())
			}
		}
		totdowncells++
		totdownbytes += int64(dlen)
		pardownbytes += int64(dlen)
		//fmt.Printf("sent %d downstream cells, %d bytes \n",
		//		totdowncells, totdownbytes)

		inflight++
		if inflight < window {
			continue // Get more cells in flight
		}

		me.Coder.DecodeStart(payloadlen, me.History)


		// Collect a cell ciphertext from each trustee
		for i := 0; i < ntrustees; i++ {
			//say hello to the trustees
			/*
			msg := make([]byte, 4)
			binary.BigEndian.PutUint32(msg[0:4], uint32(1))
			_, err2 := tsock[i].Write(msg)
			if err2 != nil {
				panic("can't say hello to trustee: " + err2.Error())
			}
			*/
			n, err := io.ReadFull(tsock[i], tslice[i])
			if n < trusize {
				panic("Read from trustee: " + err.Error())
			}
			//println("trustee slice")
			//println(hex.Dump(tslice[i]))
			me.Coder.DecodeTrustee(tslice[i])
		}

		// Collect an upstream ciphertext from each client
		for i := 0; i < nclients; i++ {
			n, err := io.ReadFull(csock[i], cslice[i])
			if n < clisize {
				panic("Read from client: " + err.Error())
			}
			//println("client slice")
			//println(hex.Dump(cslice[i]))
			me.Coder.DecodeClient(cslice[i])
		}

		outb := me.Coder.DecodeCell()
		inflight--

		totupcells++
		totupbytes += int64(payloadlen)
		parupcells++
		parupbytes += int64(payloadlen)
		//fmt.Printf("received %d upstream cells, %d bytes\n",
		//		totupcells, totupbytes)

		// Process the decoded cell
		if outb == nil {
			continue // empty or corrupt upstream cell
		}
		if len(outb) != payloadlen {
			panic("DecodeCell produced wrong-size payload")
		}

		// Decode the upstream cell header (may be empty, all zeros)
		cno := int(binary.BigEndian.Uint32(outb[0:4]))
		uplen := int(binary.BigEndian.Uint16(outb[4:6]))
		//fmt.Printf("^ %d (conn %d)\n", uplen, cno)
		if cno == 0 {
			continue // no upstream data
		}
		conn := conns[cno]
		if conn == nil { // client initiating new connection
			conn = relayNewConn(cno, downstream)
			conns[cno] = conn
		}
		if 6+uplen > payloadlen {
			log.Printf("upstream cell invalid length %d", 6+uplen)
			continue
		}

		//fmt.Printf("\nReceived byte %v (len: %d)\n", outb, len(outb))

		conn <- outb[6 : 6+uplen]
	}
}
