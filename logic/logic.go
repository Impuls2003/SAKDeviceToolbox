package logic

import (
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Тип подключенного устройства
type DeviceType int

const (
	Scanner DeviceType = iota
	ScalesCAS
	ScalesCASRequest
	ScalesKeliRequest
	ScalesMassaKRequest
	EmulatorCAS
)

type Device struct {
	Port         string
	Type         DeviceType
	LastError    string
	serialConfig serial.Mode
	serialPort   serial.Port
}

func (d *Device) Connect() error {

	// В зависимости от типа устройства выбираем разные параметры подключения
	switch d.Type {
	case Scanner:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
	}

	// Открываем порт
	port, err := serial.Open(d.Port, &d.serialConfig)
	port.SetReadTimeout(500 * time.Millisecond)

	// Если были ошибки - пишем в LastError и выходим
	if err != nil {
		d.LastError = err.Error()
		return err
	}

	// Если ошибок не было прописываем порт в структуру и выходим без ошибок
	d.serialPort = port

	return nil
}

func (d *Device) Disconnect() {
	if d.serialPort != nil {
		d.serialPort.Close()
		d.serialPort = nil
	}
}

func (d *Device) Process() (string, error) {
	if d.serialPort != nil {
		// В зависимости от типа устройства выбираем разные параметры подключения
		switch d.Type {
		case Scanner:
			return startScanTest(d)
		}
	}

	return "", nil
}

func GetAvailablePortList() []string {

	// Список доступных портов
	portNames := []string{}

	ports, err := enumerator.GetDetailedPortsList()

	// Если не было ошибок заполняем список портов и возвращаем результат.
	// В противном случае вернется пустой список
	if err == nil {
		for _, p := range ports {
			portNames = append(portNames, p.Name)
		}
	}

	return portNames
}

func startScanTest(d *Device) (string, error) {

	var res strings.Builder
	buf := make([]byte, 128)

	for {
		n, err := d.serialPort.Read(buf) // Прочитали
		// Если ошибка
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}

		if n == 0 {
			break
		}

		res.WriteString(string(buf[:n]))
	}

	return res.String(), nil
}
