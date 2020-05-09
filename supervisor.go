package goperf

import (
	"sync"
	"time"
)

type Data struct {
	sync.Mutex

	NBytes          int
	JitterSum       uint64
	JitterN         uint64
	NPktsSent       uint64
	NPktsRecd       uint64
	NPktsDropped    uint64
	NPktsOutOfOrder uint64

	NSec uint64

	NWrites       uint64
	NDroppedTicks uint64
	RcvdSeqNumber uint64
	SentSeqNumber uint64

	Running  bool
	Shutdown bool
	Extended int
}

type historyTrack struct {
	current      uint64
	last10       [10]uint64
	nFilled      int
	nextInLast10 int
	totalEver    uint64
	totalLast10  uint64
	min          uint64
	max          uint64
}

func Supervise(c *Data, output Output) {
	var localData Data
	needHeader := true

	var nsec uint64

	var r historyTrack
	var j historyTrack

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// grab the values from the generating routine
			c.Lock()

			if !c.Running {
				c.Unlock()
				continue
			}

			localData = *c
			nsec++

			c.NBytes = 0
			c.JitterSum = 0
			c.JitterN = 0
			c.NWrites = 0
			c.NSec = nsec

			c.Unlock()

			t2 := time.Now()
			t2 = t2.UTC()
			ts := t2.Format("[01-02-2006][15:04:05.000000]")

			updateHistoryTrack(&r, (uint64)(localData.NBytes*8))

			var jitterThisPeriod uint64
			if localData.JitterN == 0 {
				jitterThisPeriod = 0
			} else {
				jitterThisPeriod = localData.JitterSum / localData.JitterN
			}
			updateHistoryTrack(&j, jitterThisPeriod)

			data := OutputData{
				TS:          ts,
				NSec:        nsec,
				RateLastSec: r.current,
				Rate10Sec:   r.totalLast10 / (uint64)(r.nFilled),
				RateAvg:     r.totalEver / nsec,
				RateMax:     r.max,
				RateMin:     r.min,

				JitterLastSec: float64(j.current) / 1000000.,
				Jitter10Sec:   float64(j.totalLast10/uint64(j.nFilled)) / 1000000.,
				JitterAvg:     float64(j.totalEver/nsec) / 1000000.,
				JitterMax:     float64(j.max) / 1000000.,
				JitterMin:     float64(j.min) / 1000000.,

				NPktsRecd:       localData.NPktsRecd,
				NPktsDropped:    localData.NPktsDropped,
				NPktsOutOfOrder: localData.NPktsOutOfOrder,

				NPktsSent: localData.NPktsSent,
			}

			if localData.Shutdown {
				output.Println("Final statistics:")
				output.DisplayHeader()
				output.WriteDataLine(data)
				c.Lock()
				c.NSec = 0
				c.Shutdown = false
				c.Unlock()
				return
			} else {
				if needHeader {
					output.DisplayHeader()
					needHeader = false
				}
				output.WriteDataLine(data)
			}
		}
	}

}

func updateHistoryTrack(r *historyTrack, current uint64) {
	r.current = current

	if current < r.min {
		r.min = current
	}

	if current > r.max {
		r.max = current
	}

	r.totalEver += r.current
	r.last10[r.nextInLast10] = r.current

	r.nextInLast10++
	if r.nextInLast10 == 10 {
		r.nextInLast10 = 0
	}

	if r.nFilled < 10 {
		r.nFilled++
	}

	/* last 10 seconds */
	r.totalLast10 = 0
	for i := 0; i < r.nFilled; i++ {
		r.totalLast10 += r.last10[i]
	}
}
