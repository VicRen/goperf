package goperf

import (
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"time"
)

var udpSeqNumberNextSeqNumSend uint64 = 1

var udpSeqNumRecd uint64
var udpHighestSeqNumRecd uint64
var udpPrevSeqNumRecd uint64
var udp2BackSeqNumRecd uint64

var udpSenderNano uint64
var udpPrevSenderNano uint64

var udpTimeRecdNano uint64
var udpPrevTimeRecdNano uint64
var udp2BackTimeRecdNano uint64

var udpNPacketsRecd uint64
var udpNPacketsDropped uint64
var udpNPacketsOutOfOrder uint64

func RunUDPServer(reader io.Reader, output Output, nb, ns int64) error {
	var totalBytesRecd int64
	var nsec uint64

	udpPrevSenderNano = udpSenderNano
	udpPrevTimeRecdNano = udpTimeRecdNano

	var c Data
	//c.Extended = LOG_EXTEND_UDP_SERVER
	c.Running = false
	go Supervise(&c, output)

	defer func() {
		c.Lock()
		c.Shutdown = true
		c.Unlock()
		time.Sleep(2 * time.Second)
	}()

	buf := make([]byte, 1024)

	runtime.LockOSThread()

	for {
		var jitter uint64

		n, err := reader.Read(buf)
		udpTimeRecdNano = timeNanoseconds()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		udpSenderNano = binary.BigEndian.Uint64(buf[0:8])
		udpSeqNumRecd = binary.BigEndian.Uint64(buf[8:16])

		// this packet and the previous 2 all in order, and
		// they are the last 3 received?
		if udp2BackTimeRecdNano != 0 &&
			udpSeqNumRecd == udpHighestSeqNumRecd+1 &&
			udpSeqNumRecd == udpPrevSeqNumRecd+1 &&
			udpPrevSeqNumRecd == udp2BackSeqNumRecd+1 {

			// calculate jitter: abs((t2-t1) - (t3-t2)) = abs(t2*2 - t1 - t3) = abs(t2*2 - (t1 + t3))
			t2x2 := udpPrevTimeRecdNano << 1
			t3plusT1 := udp2BackTimeRecdNano + udpTimeRecdNano
			if t2x2 > t3plusT1 {
				jitter = t2x2 - t3plusT1
			} else {
				jitter = t3plusT1 - t2x2
			}
		} else {

			// highest seq number so far?
			if udpSeqNumRecd > udpHighestSeqNumRecd {
				// dropped packet(s)?
				if udpSeqNumRecd != udpHighestSeqNumRecd+1 {
					udpNPacketsDropped += udpSeqNumRecd - udpHighestSeqNumRecd - 1
				}
			} else {
				// must be out-of-order

				// decrement dropped packets counter, since it must have
				// been incremented when seq number gap was first detected
				udpNPacketsDropped--
				udpNPacketsOutOfOrder++
			}

			jitter = 0
		}

		// update all of the seq number tracker variables.
		udp2BackSeqNumRecd = udpPrevSeqNumRecd
		udpPrevSeqNumRecd = udpSeqNumRecd

		udp2BackTimeRecdNano = udpPrevTimeRecdNano
		udpPrevTimeRecdNano = udpTimeRecdNano

		if udpSeqNumRecd > udpHighestSeqNumRecd {
			udpHighestSeqNumRecd = udpSeqNumRecd
		}

		// count the packet
		udpNPacketsRecd++

		// update the data rate counter
		c.Lock()
		c.NBytes += n
		c.JitterSum += jitter
		c.JitterN++
		c.NPktsRecd = udpNPacketsRecd
		c.RcvdSeqNumber = udpHighestSeqNumRecd
		c.NPktsDropped = udpNPacketsDropped
		c.NPktsOutOfOrder = udpNPacketsOutOfOrder
		c.Running = true
		nsec = c.NSec
		c.Unlock()

		totalBytesRecd += int64(n)
		if nb != -1 && totalBytesRecd >= nb {
			output.Println(fmt.Sprintf("\nByte limit (%d) reached, quitting, %d total bytes received \n", nb, totalBytesRecd))
			return nil
		}

		if ns != -1 && nsec >= uint64(ns) {
			output.Println(fmt.Sprintf("\nTime limit (%d seconds) reached, quitting \n", ns))
			return nil
		}
	}
}

func RunUDPClient(writer io.Writer, output Output, pps, psize, ns, nb int64) error {
	var totalBytesSent int64
	var nsec uint64

	var c Data
	c.Running = true
	go Supervise(&c, output)

	defer func() {
		c.Lock()
		c.Shutdown = true
		c.Unlock()
		time.Sleep(2 * time.Second)
	}()

	// create payload slice
	pl := make([]byte, psize*2+16)

	// create slice for nanosecond counter
	nanoSlice := pl[:8]

	// create slice for sequence number
	snSlice := pl[8:16]

	var i int64
	for i = 0; i < psize*2-16; i++ {
		pl[i+16] = (byte)(i % 256)
	}

	// get a ticker to time the outgoing packets
	interval := time.NewTicker(time.Nanosecond * (time.Duration)(1000000000/pps))
	defer interval.Stop()

	runtime.LockOSThread()

	output.Println(fmt.Sprintf("start UDP client, pps: %d, psize: %d, ns: %d, nb: %d", pps, psize, ns, nb))

	for {
		select {
		case <-interval.C:
			nano := timeNanoseconds()
			binary.BigEndian.PutUint64(nanoSlice, nano)

			binary.BigEndian.PutUint64(snSlice, udpSeqNumberNextSeqNumSend)
			udpSeqNumberNextSeqNumSend++

			runtime.Gosched()
			n, err := writer.Write(pl[0 : psize+16])
			if err != nil {
				return err
			}

			c.Lock()
			c.NBytes += n
			c.NPktsSent++
			c.SentSeqNumber = udpSeqNumberNextSeqNumSend - 1
			nsec = c.NSec
			c.Unlock()

			totalBytesSent += int64(n)
			if nb != -1 && totalBytesSent > nb {
				output.Println(fmt.Sprintf("\nSend byte limit (%d) reached, quitting, sent %d bytes \n", nb, totalBytesSent))
				return nil
			}

			if ns != -1 && nsec >= uint64(ns) {
				output.Println(fmt.Sprintf("\nTime limit (%d seconds) reached, quitting \n", ns))
				return nil
			}
			break
		}
	}

}

func timeNanoseconds() uint64 {
	return uint64(time.Now().UnixNano())
}
