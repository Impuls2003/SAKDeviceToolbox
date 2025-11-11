package logic

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Тип подключенного устройства
type DeviceType int

const (
	Scanner             DeviceType = iota // Чтение данных от сканера
	ScalesCAS                             // Чтение данных от весов CAS с непрерывной передачей данных
	ScalesCASRequest                      // Чтение данных от весов CAS с запросом веса
	ScalesKeliRequest                     // Чтение данных от весов Keli с запросом веса
	ScalesMassaKRequest                   // Чтение данных от весов MassaK с запросом веса
	EmulatorCAS                           // Эмуляция весов CAS с непрерывной передачей данных
	EmulatorCASRequest                    // Эмуляция весов CAS с передачей данных по запросу
	EchoTest                              // ECHO тест. Пишем в com порт и сразу читаем. Если пришло что отправили значит все хорошо
)

type Device struct {
	Port         string
	Type         DeviceType
	LastError    string
	serialConfig serial.Mode
	serialPort   serial.Port
	processFunc  func(*Device) (string, error) // функция обработки
}

func (d *Device) Connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// поймали panic, превращаем в ошибку
			d.LastError = "Непредвиденная ошибка при открытии порта"
			err = fmt.Errorf(d.LastError)
		}
	}()

	// В зависимости от типа устройства выбираем разные параметры подключения
	switch d.Type {
	// Сканер
	case Scanner:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startScanTest
	// CAS непрерывная передача данных
	case ScalesCAS:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startReadWeightCAS
	// CAS по запросу
	case ScalesCASRequest:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startReadWeightCASRequest
	// Keli по запросу
	case ScalesKeliRequest:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startReadWeightKeliRequest
	// MassaK по запросу
	case ScalesMassaKRequest:
		d.serialConfig = serial.Mode{
			BaudRate: 57600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startReadWeightMassaKRequest
	// Эмуляция весов CAS с непрерывной передачей данных
	case EmulatorCAS:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startEmulateCAS
	// Эмуляция весов CAS с передачей данных по запросу
	case EmulatorCASRequest:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startEmulateCASRequest
	// ECHO тест
	case EchoTest:
		d.serialConfig = serial.Mode{
			BaudRate: 9600,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		}
		d.processFunc = startEchoTest
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
		d.processFunc = nil
	}
}

func (d *Device) Process() (string, error) {
	if d.serialPort != nil {
		if d.processFunc != nil {
			return d.processFunc(d)
		} else {
			d.LastError = "Не реализован обработчик данной функции"
			return "", fmt.Errorf(d.LastError)
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

// Сканирование
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

// Чтение веса CAS непрерывная передача данных
func startReadWeightCAS(d *Device) (string, error) {
	const (
		endSeqFirst  = byte(0x0D) // '\r'
		endSeqSecond = byte(0x0A) // '\n'
		msgSize      = 21         // читаем 21 байт после конца предыдущего пакета
	)

	var lastTwo []byte
	var data []byte

	buf := make([]byte, 1)

	startBlockFound := false

	for {
		n, err := d.serialPort.Read(buf) // Прочитали
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}
		if n == 0 {
			break
		}

		if startBlockFound {
			data = append(data, buf[0])
		} else {
			lastTwo = append(lastTwo, buf[0])
			if len(lastTwo) > 2 {
				lastTwo = lastTwo[1:]
			}
			if lastTwo[0] == endSeqFirst && lastTwo[1] == endSeqSecond {
				startBlockFound = true
			}
		}

		if len(data) > msgSize {
			break
		}
	}

	if len(data) > 0 {
		return strings.ReplaceAll(string(data), "\r\n", ""), nil
	}

	return "", nil
}

// Чтение веса CAS вес по запросу
func startReadWeightCASRequest(d *Device) (string, error) {

	const (
		msgSize = 22 // читаем 22 байт после конца предыдущего пакета
	)
	buf := make([]byte, 1)
	var data []byte

	// Отправляем в порт запрос на получение веса
	buf[0] = 68 //D
	_, err := d.serialPort.Write(buf)
	if err != nil {
		d.LastError = err.Error()
		return "", err
	}

	// Читаем из порта
	for {
		n, err := d.serialPort.Read(buf)
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}
		if n == 0 {
			break
		}

		data = append(data, buf[0])

		if len(data) >= msgSize {
			break
		}
	}

	if len(data) > 0 {
		return strings.ReplaceAll(string(data), "\r\n", ""), nil
	}

	return "", nil
}

// Чтение веса Keli вес по запросу
func startReadWeightKeliRequest(d *Device) (string, error) {

	const (
		msgSize = 16 // читаем 16 байт после конца предыдущего пакета
	)
	sendBuf := make([]byte, 3)
	buf := make([]byte, 1)
	var data []byte

	// Отправляем в порт запрос на получение веса
	sendBuf[0] = 02
	sendBuf[1] = 65
	sendBuf[2] = 03

	_, err := d.serialPort.Write(sendBuf)
	if err != nil {
		d.LastError = err.Error()
		return "", err
	}

	// Читаем из порта
	for {
		n, err := d.serialPort.Read(buf)
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}
		if n == 0 {
			break
		}

		data = append(data, buf[0])

		if len(data) >= msgSize {
			break
		}
	}

	if len(data) > 0 {
		return strings.ReplaceAll(string(data), "\r\n", ""), nil
	}

	return "", nil
}

// Чтение веса Massa-K вес по запросу
func startReadWeightMassaKRequest(d *Device) (string, error) {

	const (
		msgSize = 14 // читаем 14 байт после конца предыдущего пакета
	)
	sendBuf := make([]byte, 8)
	buf := make([]byte, 1)
	var data []byte

	// Отправляем в порт запрос на получение веса
	sendBuf[0] = 248
	sendBuf[1] = 85
	sendBuf[2] = 206
	sendBuf[3] = 1
	sendBuf[4] = 0
	sendBuf[5] = 160
	sendBuf[6] = 160
	sendBuf[7] = 0

	_, err := d.serialPort.Write(sendBuf)
	if err != nil {
		d.LastError = err.Error()
		return "", err
	}

	// Читаем из порта
	for {
		n, err := d.serialPort.Read(buf)
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}
		if n == 0 {
			break
		}

		data = append(data, buf[0])

		if len(data) >= msgSize {
			break
		}
	}

	if len(data) > 0 {
		if (data[0] == 248) && (data[1] == 85) && (data[2] == 206) {
			if len(data) == 14 {
				value := binary.LittleEndian.Uint32(data[6:10])
				return string(strconv.Itoa(int(value))), nil
			} else {
				return "Overload", nil
			}
		}
	}

	return "", nil
}

// Эмуляция весов CAS
func startEmulateCAS(d *Device) (string, error) {

	r := rand.New(rand.NewSource(time.Now().UnixNano())) // Инициализируем генератор случайных чисел

	// Открываем порт

	buf := make([]byte, 22)

	// Генерируем управляющую строку для передачи веса
	buf[0] = 83                      //S
	buf[1] = 84                      //T
	buf[2] = 44                      //,
	buf[3] = 78                      //N
	buf[4] = 84                      //T
	buf[5] = 44                      //,
	buf[6] = 1                       //
	buf[7] = 188                     //�
	buf[8] = 44                      //,
	buf[9] = 32                      //
	buf[10] = 32                     //
	buf[11] = 32                     //
	buf[12] = 48 + (byte)(r.Intn(9)) // ?
	buf[13] = 48 + (byte)(r.Intn(9)) // ?
	buf[14] = 46                     //.
	buf[15] = 48 + (byte)(r.Intn(9)) // ?
	buf[16] = 48 + (byte)(r.Intn(9)) // ?
	buf[17] = 32                     //
	buf[18] = 107                    //k
	buf[19] = 103                    //g
	buf[20] = 13                     // /r
	buf[21] = 10                     // /n

	_, err := d.serialPort.Write(buf)
	if err != nil {
		d.LastError = err.Error()
		return "", err
	}

	time.Sleep(500 * time.Millisecond)

	return strings.ReplaceAll(string(buf), "\r\n", ""), nil
}

// Эмуляция весов CAS
func startEmulateCASRequest(d *Device) (string, error) {

	buf := make([]byte, 1)

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

		if buf[0] == 68 {
			return startEmulateCAS(d)
		}
	}
	return "", nil
}

// Echo тест
func startEchoTest(d *Device) (string, error) {

	const ArraySize = 128
	r := rand.New(rand.NewSource(time.Now().UnixNano())) // Инициализируем генератор случайных чисел

	// Заполняем тестовую выборку случайными данными
	testArray := make([]byte, ArraySize)
	for i := range testArray {
		testArray[i] = byte(r.Intn(256))
	}

	// Пишем в порт весь массив
	n, err := d.serialPort.Write(testArray)
	if err != nil {
		d.LastError = err.Error()
		return "", err
	}

	if n != ArraySize {
		d.LastError = "Не все байты записаны в порт"
		return "", fmt.Errorf(d.LastError)
	}

	buf := make([]byte, ArraySize)
	totalRead := 0

	// Читаем из порта столько сколько записали
	for totalRead < ArraySize {
		n, err := d.serialPort.Read(buf[totalRead:])
		if err != nil {
			d.LastError = err.Error()
			return "", err
		}

		if n == 0 {
			break
		}
		totalRead += n
	}

	// Сравниваем
	isOk := true
	for i := 0; i < ArraySize; i++ {
		if buf[i] != testArray[i] {
			isOk = false
			break
		}
	}

	if isOk {
		return "PASS", nil
	} else {
		return "FAIL", nil
	}
}
