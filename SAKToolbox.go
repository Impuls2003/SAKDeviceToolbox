package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
	"golang.org/x/sys/windows"
)

var user32_dll = windows.NewLazyDLL("user32.dll")
var GetKeyState = user32_dll.NewProc("GetKeyState")

func main() {

	var deviceSelect = 1
	var devicePort = "1"

	for {
		//fmt.Print("\033[H\033[2J") // Очистка экрана
		var deviceType = [...]string{
			"Сканер",
			"Весы",
			"Echo тест (необходимо иметь Echo dongle Tx-Rx; Rx-Tx)"}

		// Выводим весь список доступного оборудования
		fmt.Println("Выберите тип тестируемого устройства: ")
		for index, value := range deviceType {
			fmt.Println("\t", strconv.Itoa(index+1)+".", value)
		}

		// Выводим запрос на то, что будем читать
		fmt.Print("Тип тестируемого устройства [1]: ")
		fmt.Scanf("%d\n", &deviceSelect)

		fmt.Print("Номер порта [1]: ")
		fmt.Scanf("%s\n", &devicePort)

		// Тестируем сканер
		switch deviceSelect {
		case 1:
			startScanTest(devicePort)
		case 2:
			startWeightTest(devicePort)
		case 3:
			startEchoTest(devicePort)
		}
	}
}

// У функции единственное предназначение. Она проверяет состояние ESC. Если кнопка нажата вернут true
func ESCIsPressed() bool {

	r1, _, _ := GetKeyState.Call(27) // Читаем состояние кнопки ESC.
	return (r1 > 1)

}

func startScanTest(devicePort string) {

	fmt.Println("Начато чтение данных из: ", "COM"+devicePort, "\n", "ESC для выхода.")

	// Открываем порт
	c := &serial.Config{Name: "COM" + devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)

	if err != nil {
		log.Fatal(err)
	}

	var n int = 0
	buf := make([]byte, 128)

	// Заходим в бесконечный цикл чтения данных из порта. Выйти отсюда можно только через ESC
	for {
		n, err = s.Read(buf) // Прочитали
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < n; i++ { // Посмвольно выводим прочитанное в консоль. Если встретили 13 - переходим на новую строку
			if buf[i] == 13 {
				fmt.Print("\n")
			} else {
				fmt.Print(string(buf[i]))
			}
		}

		if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
			s.Close()
			return
		}
	}
}

func startWeightTest(devicePort string) {

	var deviceSelect = 1
	var deviceType = [...]string{
		"CAS",
		"Massa-K",
	}

	for {
		// Выводим весь список доступного оборудования
		fmt.Println("Выберите модель весов: ")
		for index, value := range deviceType {
			fmt.Println("\t", strconv.Itoa(index+1)+".", value)
		}

		// Выводим запрос на то, что будем читать
		fmt.Print("Модель весов [1]: ")
		fmt.Scanf("%d\n", &deviceSelect)

		fmt.Println("Начато чтение данных из: ", "COM"+devicePort, "\n", "ESC для выхода.")

		switch deviceSelect {
		case 1: // CAS
			readWeightFromCAS(devicePort)
		case 2:
			readWeightFromMassaK(devicePort)
		}

		if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
			return
		}
	}
}

func readWeightFromCAS(devicePort string) {
	// Открываем порт
	c := &serial.Config{Name: "COM" + devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)

	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1)
	var n int
	var weight string

	// Заходим в бесконечный цикл чтения данных из порта. Выйти отсюда можно только через ESC
	for {
		for (len(weight)) < 16 {
			n, err = s.Read(buf) // Прочитали
			if err != nil {
				log.Fatal(err)
			}
			if n != 0 {
				weight += string(buf)
			}
			if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
				s.Close()
				return
			}
		}
		fmt.Print(strings.ReplaceAll(weight, "\n", ""))
		weight = ""

		if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
			s.Close()
			return
		}
	}
}

func readWeightFromMassaK(devicePort string) {

}
func startEchoTest(devicePort string) {

	const ArraySize = 128

	fmt.Println("Начато чтение данных из: ", "COM"+devicePort, "\n", "ESC для выхода.")

	r := rand.New(rand.NewSource(time.Now().UnixNano())) // Инициализируем генератор случайных чисел

	// Открываем порт
	c := &serial.Config{Name: "COM" + devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)

	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1)

	// Заходим в бесконечный цикл чтения данных из порта. Выйти отсюда можно только через ESC
	for {
		isOk := true
		var testArray [1]byte
		// Побайтово пишем в порт и сразу читаем. Далее сравниваем отправленное с прочитанным. Если совпало - хорошо
		for i := 0; i < ArraySize; i++ {
			testArray[0] = byte(r.Intn(255))
			_, err := s.Write(testArray[:])
			if err != nil {
				log.Fatal(err)
			}

			_, err = s.Read(buf)
			if err != nil {
				log.Fatal(err)
			}
			if buf[0] != testArray[0] {
				isOk = false
				break
			}
		}
		if isOk {
			fmt.Println("PASS")
		} else {
			fmt.Println("FAIL")
		}

		if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
			s.Close()
			return
		}
	}
}
