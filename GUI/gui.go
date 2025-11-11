package gui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Impuls2003/SAKDeviceToolbox/logic"
	"golang.org/x/sys/windows"
)

var user32_dll = windows.NewLazyDLL("user32.dll")
var GetKeyState = user32_dll.NewProc("GetKeyState")

func Show(device *logic.Device) {
	// Бесконечный цикл. Выход только из меню, или закрыв приложение
	for {
		// Если порт не выбран, делаем запрос для выбора порта
		// Если порт выбран предлагаем пользователю его сменить
		if device.Port == "" {
			showSelectCOMPortMenu(device)
		}
		// Очищаем экран и выводим информационные заголовки
		showHeader(device)
		device.LastError = ""

		prompt := &survey.Select{
			Message: "Выберите действие:",
			Options: []string{
				"Сканер",
				"Весы",
				"Echo тест",
				"Сменить COM порт",
				"Выход",
			},
		}

		var deviceType string
		// Выводим главное меню
		survey.AskOne(prompt, &deviceType)

		switch deviceType {
		case "Сменить COM порт":
			showSelectCOMPortMenu(device)
		case "Сканер":
			showScannerMenu(device)
		case "Весы":
			showWeightMenu(device)
		case "Echo тест":
			showEchoTestMenu(device)
		case "Выход":
			os.Exit(0)
		}
	}
}

// Отображает меню выбора порта
func showSelectCOMPortMenu(device *logic.Device) {

	// Очищаем экран
	clearScreen()

	// Получаем список доступных в системе портов
	portNames := logic.GetAvailablePortList()

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

	device.Port = selected
}

// Отображает меню работы со сканером
func showScannerMenu(device *logic.Device) {
	device.Type = logic.Scanner
	showHeader(device)
	fmt.Println("Начато получение данных от сканера. ESC для выхода.")
	if device.Connect() == nil {
		// Если подключение прошло успешно.
		// Заходим в бесконечный цикл. Выход из цикла по ESC
		for {
			str, err := device.Process()
			// Если была ошибка - выходим из цикла
			if err != nil {
				break
			}

			// Если есть что выводить - выводим
			if str != "" {
				fmt.Println(str)
			}

			// Если нажата ESC - выходим из цикла
			if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
				break
			}
		}

		device.Disconnect()
	}
}

// Отображает меню работы с весами
func showWeightMenu(device *logic.Device) {
	// Бесконечный цикл. Выход из цикла только через меню
	for {
		// Очищаем экран и выводим текущий выбранный порт
		showHeader(device)

		var weightType string

		prompt := &survey.Select{
			Message: "Выберите тип весов:",
			Options: []string{
				"CAS",
				"CAS по запросу (запрос веса: ASCII - D, HEX - 44, DEC - 68)",
				"Keli",
				"Massa-K",
				"Эмуляция весов CAS непрерывно",
				"Эмуляция весов CAS по запросу (HEX - 44)",
				"Назад",
			},
		}
		// Выводим главное меню
		survey.AskOne(prompt, &weightType)

		switch weightType {
		case "CAS":
			device.Type = logic.ScalesCAS
		case "CAS по запросу (запрос веса: ASCII - D, HEX - 44, DEC - 68)":
			device.Type = logic.ScalesCASRequest
		case "Keli":
			device.Type = logic.ScalesKeliRequest
		case "Massa-K":
			device.Type = logic.ScalesMassaKRequest
		case "Эмуляция весов CAS непрерывно":
			device.Type = logic.EmulatorCAS
		case "Эмуляция весов CAS по запросу (HEX - 44)":
			device.Type = logic.EmulatorCASRequest
		case "Назад":
			return
		}

		showHeader(device)
		fmt.Println("Начато получение данных от весов. ESC для выхода.")
		if device.Connect() == nil {
			// Если подключение прошло успешно.
			// Заходим в бесконечный цикл. Выход из цикла по ESC
			for {
				str, err := device.Process()
				// Если была ошибка - выходим из цикла
				if err != nil {
					break
				}

				// Если есть что выводить - выводим
				// Форматируем строку чтобы не было перехода на новую строку
				if str != "" {
					fmt.Printf("\rВес: %-100s", str)
				}

				// Если нажата ESC - выходим из цикла
				if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
					break
				}
			}

			device.Disconnect()
		}
	}
}

// Отображает меню работы со сканером
func showEchoTestMenu(device *logic.Device) {
	device.Type = logic.EchoTest
	showHeader(device)
	fmt.Println("Начато Echo тестирование порта. ESC для выхода.")
	if device.Connect() == nil {
		// Если подключение прошло успешно.
		// Заходим в бесконечный цикл. Выход из цикла по ESC
		for {
			str, err := device.Process()
			// Если была ошибка - выходим из цикла
			if err != nil {
				break
			}

			// Если есть что выводить - выводим
			if str != "" {
				fmt.Println(str)
			}

			// Если нажата ESC - выходим из цикла
			if ESCIsPressed() { // Проверяем состояние ESC. Если нажата - выходим
				break
			}
		}

		device.Disconnect()
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

// Показывает заголовок. Текущий порт, состояние ошибок и др.
func showHeader(device *logic.Device) {

	clearScreen()
	// Зеленым текущий порт
	fmt.Printf("Текущий порт: \033[32m%s\033[0m\n", device.Port)

	// Если в процессе были ошибки - вывести их на экран красным
	if device.LastError != "" {
		fmt.Printf("\033[31m%s\033[0m\n", device.LastError)
		device.LastError = ""
	}
}

// У функции единственное предназначение. Она проверяет состояние ESC. Если кнопка нажата вернуть true
func ESCIsPressed() bool {
	r1, _, _ := GetKeyState.Call(27) // Читаем состояние кнопки ESC.
	return (r1 > 1)
}
