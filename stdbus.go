package stdbus

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sigurn/crc16"
	"github.com/tarm/serial"
)

const STX uint8 = 0xc0
const ETX uint8 = 0xc1
const DIST uint8 = 0x7d

type STDBUS struct {
	pstSerialPort *serial.Port
	pstCRCTable   *crc16.Table
	pn8SendPacket []byte
	pn8RcvPacket  []byte
}

func GetSTDBUS(port string, baud int) (*STDBUS, error) {

	stCRCTable := crc16.MakeTable(crc16.CRC16_MODBUS)

	stSerialPort, err := serial.OpenPort(&serial.Config{Name: port, Baud: baud, ReadTimeout: time.Millisecond * 10})
	if err != nil {
		return nil, err
	}

	return &STDBUS{pstSerialPort: stSerialPort, pstCRCTable: stCRCTable}, err
}

func (ego *STDBUS) Packetsend(a_an8Packet []byte) ([]byte, error) {

	retCRC, err := ego.makeCRC(a_an8Packet)
	if err != nil {
		return nil, err
	}

	retEncode, err := ego.packetEncode(retCRC)
	if err != nil {
		return nil, err
	}

	sendRes, err := ego.pstSerialPort.Write(retEncode)
	if err != nil {
		return nil, err
	}

	_ = sendRes

	rcvData, err := ego.packetReceive()
	if err != nil {
		return nil, err
	}

	resCRC, err := ego.calcCRC(rcvData)
	if err != nil {
		return nil, err
	}

	resDecode, err := ego.packetDecode(resCRC)
	if err != nil {
		return nil, err
	}
	return resDecode, nil
}

func (ego *STDBUS) makeCRC(a_an8Packet []byte) ([]byte, error) {

	if len(a_an8Packet) == 0 {
		return nil, errors.New("makeCRC : Data size is 0")
	}

	checksum := crc16.Checksum(a_an8Packet, ego.pstCRCTable)
	a_an8Packet = append(a_an8Packet, (uint8)(checksum&0xff))
	a_an8Packet = append(a_an8Packet, (uint8)((checksum>>8)&0xff))
	return a_an8Packet, nil
}

func (ego *STDBUS) packetEncode(a_an8Packet []byte) ([]byte, error) {

	if len(a_an8Packet) == 0 {
		return nil, errors.New("packetEncode : Data size is 0")
	}

	temp := make([]byte, 0)

	for _, v := range a_an8Packet {
		if (v == STX) || (v == ETX) || (v == DIST) {
			temp = append(temp, 0x7d)
			temp = append(temp, v^0x7d)

		} else {
			temp = append(temp, v)
		}

	}

	temp = append([]byte{STX}, temp...)
	temp = append(temp, ETX)

	return temp, nil
}

func (ego *STDBUS) packetReceive() ([]byte, error) {

	//n32Timeout := 0
	const MODE_WAIT = 1
	const MODE_RCV = 2
	var mode = MODE_WAIT

	timeout := 0
	rcvTemp := make([]byte, 512)
	rcvData := make([]byte, 0, 512)

	for {
		//ego.pstSerialPort.
		//fmt.Printf(" pstSerialPortn")
		n, err := ego.pstSerialPort.Read(rcvTemp)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf(" %d\n", n)

		if n > 0 {
			timeout = 0

			for i := 0; i < n; i++ {
				if mode == MODE_RCV {
					if rcvTemp[i] == ETX {
						mode = MODE_WAIT
						return rcvData, nil
					} else {
						rcvData = append(rcvData, rcvTemp[i])
					}
				} else if mode == MODE_WAIT {
					if rcvTemp[i] == STX {
						mode = MODE_RCV
						rcvTemp = make([]byte, 512)
						rcvData = make([]byte, 0, 512)
					}
				}
			}
			continue
		}
		timeout++
		if timeout >= 3 {
			return rcvData, errors.New("packetReceive : Receive Timeout")

		}
	}

}

func (ego *STDBUS) calcCRC(a_an8Packet []byte) ([]byte, error) {
	checksum := crc16.Checksum(a_an8Packet, ego.pstCRCTable)

	temp := a_an8Packet[0 : len(a_an8Packet)-2]

	if checksum != 0 {

		fmt.Println(a_an8Packet)
		fmt.Printf("calcCRC: %x\n", checksum)
		fmt.Printf("calcCRC: %x\n", a_an8Packet[len(a_an8Packet)-1])
		fmt.Printf("calcCRC: %x\n", a_an8Packet[len(a_an8Packet)-2])

		return temp, errors.New("calcCRC: Received data CRC wrong")
	}
	return temp, nil
}

func (ego *STDBUS) packetDecode(a_an8Packet []byte) ([]byte, error) {

	if len(a_an8Packet) == 0 {
		return nil, errors.New("packetDecode : Data size is 0")
	}
	temp := make([]byte, 0)
	flag := false

	for _, v := range a_an8Packet {

		if flag {
			temp = append(temp, v^0x7d)
			flag = false

		} else {
			if v == DIST {
				flag = true
			} else {
				temp = append(temp, v)
			}
		}
	}
	return temp, nil
}
