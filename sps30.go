package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	operadatatypes "github.com/Potsdam-Sensors/OPERA-Data-Types"
	"github.com/jacobsa/go-serial/serial"
)

var startCmd = []byte{0x7E, 0x00, 0x00, 0x02, 0x01, 0x03, 0xF9, 0x7E}

// var stopCmd = []byte{0x7E, 0x00, 0x01, 0x00, 0xFE, 0x7E}
var reqDataCmd = []byte{0x7E, 0x00, 0x03, 0x00, 0xFC, 0x7E}

// var sps30DataHeaders = []string{"PM1", "PM2.5", "PM4", "PM10", "PN0.5", "PN1", "PN2.5", "PN4", "PN10", "typical_particle_size"}

const (
	fullMsgByteLen = 47
	stopByte       = byte(0x7E)
)

func drainReader(port io.Reader) {
	buff := make([]byte, 1)
	for {
		_, err := port.Read(buff)
		if err != nil {
			return
		}
	}
}
func startSps30(port io.ReadWriter) error {
	_, err := port.Write(startCmd)
	time.Sleep(5 * time.Second)
	return err
}

// func stopSps30(port io.ReadWriter) error {
// 	_, err := port.Write(stopCmd)
// 	return err
// }

func requestDataSps30(port io.Writer) error {
	_, err := port.Write(reqDataCmd)
	return err
}
func ReadBytes(port io.Reader, buff []byte, timeoutDuration time.Duration) (int, error) {
	timer := time.NewTimer(timeoutDuration)
	defer timer.Stop()
	bytesWant := len(buff)
	bytesRead := 0
	for {
		select {
		case <-timer.C:
			return bytesRead, fmt.Errorf("operation timed out")
		default:
			if n, err := port.Read(buff[bytesRead:]); err != nil {
				return bytesRead + n, err
			} else if bytesRead+n == bytesWant {
				return bytesRead + n, nil
			} else {
				bytesRead += n
			}
		}
	}
}

// var sps30DataHeaders = []string{"PM1", "PM2.5", "PM4", "PM10", "PN0.5", "PN1", "PN2.5", "PN4", "PN10", "typical_particle_size"}

func PopulateFromBytes(d *operadatatypes.Sps30Data, buff []byte) error {
	reader := bytes.NewReader(buff)
	for _, pFloat := range []*float32{&(d.Pm1), &(d.Pm2p5), &(d.Pm4), &(d.Pm10),
		&(d.Pn0p5), &(d.Pn1), &(d.Pn2p5), &(d.Pn4), &(d.Pn10), &(d.TypicalParticleSize)} {
		if err := binary.Read(reader, binary.BigEndian, pFloat); err != nil {
			return err
		} else {
			*pFloat = float32(round(float64(*pFloat), 2))
		}
	}
	return nil
}
func round(val float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(val*scale) / scale
}
func readSps30(port io.ReadWriter, timeoutDuration time.Duration) (*operadatatypes.Sps30Data, error) {
	drainReader(port)
	if err := requestDataSps30(port); err != nil {
		return nil, fmt.Errorf("error requesting data from sps30: %w", err)
	}
	buff := make([]byte, fullMsgByteLen)
	if n, err := ReadBytes(port, buff, timeoutDuration); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("error reading bytes: %w", err)
		} else {
			return nil, nil
		}
	} else if n != fullMsgByteLen {
		return nil, fmt.Errorf("expected %d bytes, got %d", fullMsgByteLen, n)
	}
	buff = doReverseByteStuffingSps30(buff)
	buff = buff[5:] // Discard header
	data := &operadatatypes.Sps30Data{}
	err := PopulateFromBytes(data, buff)
	return data, err
}

func doReverseByteStuffingSps30(buff []byte) []byte {
	return bytes.ReplaceAll(
		bytes.ReplaceAll(
			bytes.ReplaceAll(
				bytes.ReplaceAll(
					buff,
					[]byte{0x7D, 0x5E},
					[]byte{0x7E},
				),
				[]byte{0x7D, 0x5D},
				[]byte{0x7D},
			),
			[]byte{0x7D, 0x31},
			[]byte{0x11},
		),
		[]byte{0x7D, 0x33},
		[]byte{0x13},
	)
}
func openPort(portPath string) (io.ReadWriteCloser, error) {
	portOptions := serial.OpenOptions{
		PortName:              portPath,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		InterCharacterTimeout: 1000,
	}
	return serial.Open(portOptions)
}
