package goperf

import (
	"fmt"
	"io"
)

// OutputData ...
type OutputData struct {
	TS          string
	NSec        uint64
	RateLastSec uint64
	Rate10Sec   uint64
	RateAvg     uint64
	RateMax     uint64
	RateMin     uint64

	JitterLastSec float64
	Jitter10Sec   float64
	JitterAvg     float64
	JitterMax     float64
	JitterMin     float64

	NPktsRecd       uint64
	NPktsDropped    uint64
	NPktsOutOfOrder uint64

	NPktsSent uint64
}

// Output ...
type Output interface {
	DisplayHeader()
	WriteDataLine(data OutputData)
	Println(a ...interface{})
	FetchLastData() *OutputData
}

// SendingOutput ...
type SendingOutput struct {
	writer   io.Writer
	lastData *OutputData
	useTS    bool
}

// NewSendingOutput ...
func NewSendingOutput(writer io.Writer, useTS bool) *SendingOutput {
	return &SendingOutput{
		writer: writer,
		useTS:  useTS,
	}
}

// DisplayHeader implements Output.
func (o *SendingOutput) DisplayHeader() {
	if o.writer == nil {
		return
	}
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "                             ")
	}
	_, _ = fmt.Fprintf(o.writer, "           [ <-------- Data Rate (bps) --------> ]\n")
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "[                  Timestamp]")
	}
	_, _ = fmt.Fprintf(o.writer, "[  # Secs ][ Lst Secnd ][  Lst 10 S ][ Snce Strt ][ # Packets Sent ] \n")
}

// WriteDataLine implements Output.
func (o *SendingOutput) WriteDataLine(data OutputData) {
	o.lastData = &data
	if o.writer == nil {
		return
	}
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "%s", data.TS)
	}

	/* # seconds */
	_, _ = fmt.Fprintf(o.writer, "[ %7d ]", data.NSec)

	/* last second */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.RateLastSec))

	/* last 10 seconds */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.Rate10Sec))

	/* total */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.RateAvg))

	_, _ = fmt.Fprintf(o.writer, "[ %14d ]", data.NPktsSent)

	_, _ = fmt.Fprintf(o.writer, "%s", "\n")
}

// FetchLastData implements Output.
func (o *SendingOutput) FetchLastData() *OutputData {
	return o.lastData
}

// Println implements Output.
func (o *SendingOutput) Println(a ...interface{}) {
	if o.writer == nil {
		return
	}
	_, _ = fmt.Fprintln(o.writer, a...)
}

// ReceivingOutput ...
type ReceivingOutput struct {
	writer   io.Writer
	lastData *OutputData
	useTS    bool
}

// NewReceivingOutput ...
func NewReceivingOutput(writer io.Writer, useTS bool) *ReceivingOutput {
	return &ReceivingOutput{
		writer: writer,
		useTS:  useTS,
	}
}

// DisplayHeader implements Output.
func (o *ReceivingOutput) DisplayHeader() {
	if o.writer == nil {
		return
	}
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "                             ")
	}
	_, _ = fmt.Fprintf(o.writer, "           [ <-------- Data Rate (bps) --------> ]")
	_, _ = fmt.Fprintf(o.writer, "[ <---------- Jitter (ms) -----------> ][ <---- Number of Packets ----> ] \n")
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "[                  Timestamp]")
	}
	_, _ = fmt.Fprintf(o.writer, "[  # Secs ][ Lst Secnd ][  Lst 10 S ][ Snce Strt ]")
	_, _ = fmt.Fprintf(o.writer, "[ Last Sec ][ Last 10 S ][ Since Start ][ Receivd ][ Dropped ][ OutOrdr ] \n")
}

// WriteDataLine implements Output.
func (o *ReceivingOutput) WriteDataLine(data OutputData) {
	o.lastData = &data
	if o.writer == nil {
		return
	}
	if o.useTS {
		_, _ = fmt.Fprintf(o.writer, "%s", data.TS)
	}

	/* # seconds */
	_, _ = fmt.Fprintf(o.writer, "[ %7d ]", data.NSec)

	/* last second */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.RateLastSec))

	/* last 10 seconds */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.Rate10Sec))

	/* total */
	_, _ = fmt.Fprintf(o.writer, "[ %9s ]", formatRate(data.RateAvg))

	_, _ = fmt.Fprintf(o.writer, "[ %8.3f ][ %9.3f ][ %11.3f ][ %7d ][ %7d ][ %7d ]",
		data.JitterLastSec,
		data.Jitter10Sec,
		data.JitterAvg,
		data.NPktsRecd,
		data.NPktsDropped,
		data.NPktsOutOfOrder)

	_, _ = fmt.Fprintf(o.writer, "%s", "\n")
}

// Println implements Output.
func (o *ReceivingOutput) Println(a ...interface{}) {
	if o.writer == nil {
		return
	}
	_, _ = fmt.Fprintln(o.writer, a...)
}

func (o *ReceivingOutput) FetchLastData() *OutputData {
	return o.lastData
}

// DefaultOutput ...
type DefaultOutput struct {
}

// DisplayHeader implements Output.
func (DefaultOutput) DisplayHeader() {
}

// WriteDataLine implements Output.
func (DefaultOutput) WriteDataLine(data OutputData) {
}

// Println implements Output.
func (DefaultOutput) Println(a ...interface{}) {
}

func formatRate(bps uint64) string {

	var label string
	var r float64
	var ret string
	var bpsf float64 = (float64)(bps)

	switch {
	case bps > 1000000000:
		label = "G"
		r = bpsf / 1000000000.

	case bps > 1000000:
		label = "M"
		r = bpsf / 1000000.

	case bps > 1000:
		label = "K"
		r = bpsf / 1000.

	default:
		label = " "
		r = bpsf
	}

	ret = fmt.Sprintf("%5.3f%s", r, label)
	return ret
}
