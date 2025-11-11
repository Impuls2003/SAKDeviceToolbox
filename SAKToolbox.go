package main

import (
	gui "github.com/Impuls2003/SAKDeviceToolbox/GUI"
	"github.com/Impuls2003/SAKDeviceToolbox/logic"
)

func main() {
	device := &logic.Device{}
	gui.Show(device)
}
