package ptouch

import (
	"fmt"
	"strings"
)

// Status is the 32-byte status reply (spec §4, "Status information request").
type Status struct {
	Model      byte
	Err1, Err2 byte
	MediaWidth int  // mm; 0 = no tape
	MediaType  byte // 01h laminated, 03h non-laminated, 11h/17h heat-shrink, FFh incompatible
	Type       byte // StatusReply, StatusPrinted, StatusError, ...
	Phase      byte // 00h editing (can receive), 01h printing
}

const (
	StatusReply      = 0x00
	StatusPrinted    = 0x01
	StatusError      = 0x02
	StatusTurnedOff  = 0x04
	StatusNotify     = 0x05
	StatusPhaseShift = 0x06

	ModelPTE550W = 0x66
	ModelPTP750W = 0x68
)

func ParseStatus(b []byte) (Status, error) {
	if len(b) != 32 || b[0] != 0x80 || b[1] != 0x20 || b[2] != 'B' {
		return Status{}, fmt.Errorf("ptouch: not a status packet: % x", b)
	}
	return Status{
		Model:      b[4],
		Err1:       b[8],
		Err2:       b[9],
		MediaWidth: int(b[10]),
		MediaType:  b[11],
		Type:       b[18],
		Phase:      b[19],
	}, nil
}

func (s Status) ModelName() string {
	switch s.Model {
	case ModelPTE550W:
		return "PT-E550W"
	case ModelPTP750W:
		return "PT-P750W"
	}
	return fmt.Sprintf("unknown model %#02x", s.Model)
}

// Problems lists the error flags in plain words; empty means ready.
func (s Status) Problems() []string {
	var p []string
	flags := []struct {
		set  bool
		text string
	}{
		{s.Err1&0x01 != 0, "no tape"},
		{s.Err1&0x04 != 0, "cutter jam"},
		{s.Err1&0x08 != 0, "weak batteries"},
		{s.Err1&0x40 != 0, "wrong (high-voltage) adapter"},
		{s.Err2&0x01 != 0, "wrong tape loaded"},
		{s.Err2&0x10 != 0, "cover open"},
		{s.Err2&0x20 != 0, "overheating"},
	}
	for _, f := range flags {
		if f.set {
			p = append(p, f.text)
		}
	}
	if s.MediaType == 0xff {
		p = append(p, "incompatible tape")
	}
	return p
}

// Err returns the problems as an error, or nil if the printer is ready.
func (s Status) Err() error {
	if p := s.Problems(); len(p) > 0 {
		return fmt.Errorf("printer: %s", strings.Join(p, ", "))
	}
	return nil
}
