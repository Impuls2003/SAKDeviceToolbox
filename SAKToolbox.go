package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/tarm/serial"
	"go.bug.st/serial/enumerator"
	"golang.org/x/sys/windows"
)

var user32_dll = windows.NewLazyDLL("user32.dll")
var GetKeyState = user32_dll.NewProc("GetKeyState")

func main() {
	mainMenu()
}

func mainMenu() {
	var devicePort string

	// Бесконечный цикл. Выход только из меню, или закрыв приложение
	for {
		// Если порт не выбран, делаем запрос для выбора порта
		// Если порт выбран предлагаем пользователю его сменить
		if devicePort == "" {
			devicePort = showMenuSelectCOMPort()
		}
		// Очищаем экран и выводим текущий выбранный порт
		clearScreen()
		fmt.Println("Текущий порт: ", devicePort)

		prompt := &survey.Select{
			Message: "Выберите действие:",
			Options: []string{
				"Сменить COM порт",
				"Сканер",
				"Весы",
				"Echo тест",
				"Эмуляция весов CAS",
				"Выход",
			},
		}

		var deviceType string
		// Выводим главное меню
		survey.AskOne(prompt, &deviceType)

		switch deviceType {
		case "Сменить COM порт":
			devicePort = showMenuSelectCOMPort()
			continue
		case "Сканер":
			startScanTest(devicePort)
		case "Весы":
			showWeightMenu(devicePort)
		case "Echo тест":
			startEchoTest(devicePort)
			continue
		case "Эмуляция весов CAS":
			emulateCAS(devicePort)
		case "Выход":
			fmt.Println("Завершение работы.")
			os.Exit(0)
		}
	}
}

// Отображает меню выбора порта
func showMenuSelectCOMPort() string {
	clearScreen()

	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatal(err)
	}

	portNames := []string{}
	for _, p := range ports {
		portNames = append(portNames, p.Name)
	}
	// Особый пункт меню для ввода вручную
	portNames = append(portNames, "Ввести вручную...")

	var selected string

	// меню выбора
	prompt := &survey.Select{
		Message:  "Выберите COM-порт:",
		Options:  portNames,
		PageSize: 10,
	}

	survey.AskOne(prompt, &selected)

	if selected == "Ввести вручную..." {
		survey.AskOne(&survey.Input{Message: "Введите порт вручную:"}, &selected)
		selected = "COM" + selected
	}

	return selected
}

func showWeightMenu(devicePort string) {

	// Бесконечный цикл. Выход из цикла только через меню
	for {
		// Очищаем экран и выводим текущий выбранный порт
		clearScreen()
		fmt.Println("Текущий порт: ", devicePort)

		var weightType string

		prompt := &survey.Select{
			Message: "Выберите тип весов:",
			Options: []string{
				"CAS",
				"CAS по запросу (запрос веса: ASCII - D, HEX - 44, DEC - 68)",
				"Keli",
				"Massa-K",
				"Назад",
			},
		}
		// Выводим главное меню
		survey.AskOne(prompt, &weightType)

		switch weightType {
		case "CAS":
			readWeightFromCAS(devicePort)
		case "CAS по запросу (запрос веса: ASCII - D, HEX - 44, DEC - 68)":
			readWeightFromCASWithRequest(devicePort)
		case "Keli":
			//
		case "Massa-K":
			readWeightFromMassaK(devicePort)
		case "Назад":
			return
		}
	}
}

// Очистка экрана
func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// У функции единственное предназначение. Она проверяет состояние ESC. Если кнопка нажата вернуть true
func ESCIsPressed() bool {

	r1, _, _ := GetKeyState.Call(27) // Читаем состояние кнопки ESC.
	return (r1 > 1)

}

func startScanTest(devicePort string) {

	fmt.Println("Начато чтение данных из: ", devicePort, "\n", "ESC для выхода.")

	// Открываем порт
	c := &serial.Config{Name: devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
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

func readWeightFromCAS(devicePort string) {
	// Открываем порт
	c := &serial.Config{Name: devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
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

func readWeightFromCASWithRequest(devicePort string) {
	// Открываем порт
	c := &serial.Config{Name: devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)

	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1)
	var n int
	var weight string

	// Заходим в бесконечный цикл чтения данных из порта. Выйти отсюда можно только через ESC
	for {
		buf[0] = 68 //D
		_, err := s.Write(buf[:])
		if err != nil {
			log.Fatal(err)
		}

		for (len(weight)) < 22 {
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
		time.Sleep(10)
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

func emulateCAS(devicePort string) {

	var fixedW = "n"
	fmt.Print("Зафиксировать вес? y/n [n]: ")
	fmt.Scanf("%s\n", &fixedW)
	fmt.Println("Начата отправка данных в: ", devicePort, "\n", "ESC для выхода.")

	r := rand.New(rand.NewSource(time.Now().UnixNano())) // Инициализируем генератор случайных чисел

	// Открываем порт
	c := &serial.Config{Name: devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
	s, err := serial.OpenPort(c)

	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 22)
	isFirst := true

	// Заходим в бесконечный цикл записи данных в порт. Выйти отсюда можно только через ESC
	for {
		// Генерируем управляющую строку для передачи веса
		if isFirst || fixedW != "y" {
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

			isFirst = false
		}

		_, err := s.Write(buf[:])
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(10)
		if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
			s.Close()
			return
		}
	}
}

func startEchoTest(devicePort string) {

	const ArraySize = 128

	fmt.Println("Для выхода нажмите ESC.")

	r := rand.New(rand.NewSource(time.Now().UnixNano())) // Инициализируем генератор случайных чисел

	// Открываем порт
	c := &serial.Config{Name: devicePort, Baud: 9600, ReadTimeout: time.Millisecond * 500}
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
