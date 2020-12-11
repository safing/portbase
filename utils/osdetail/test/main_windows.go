package main

import (
	"fmt"

	"github.com/safing/portbase/utils/osdetail"
)

func main() {
	fmt.Println("Binary Names:")
	printBinaryName("openvpn-gui.exe", `C:\Program Files\OpenVPN\bin\openvpn-gui.exe`)
	printBinaryName("firefox.exe", `C:\Program Files (x86)\Mozilla Firefox\firefox.exe`)
	printBinaryName("powershell.exe", `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`)
	printBinaryName("explorer.exe", `C:\Windows\explorer.exe`)
	printBinaryName("svchost.exe", `C:\Windows\System32\svchost.exe`)

	fmt.Println("\n\nBinary Icons:")
	printBinaryIcon("openvpn-gui.exe", `C:\Program Files\OpenVPN\bin\openvpn-gui.exe`)
	printBinaryIcon("firefox.exe", `C:\Program Files (x86)\Mozilla Firefox\firefox.exe`)
	printBinaryIcon("powershell.exe", `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`)
	printBinaryIcon("explorer.exe", `C:\Windows\explorer.exe`)
	printBinaryIcon("svchost.exe", `C:\Windows\System32\svchost.exe`)

	fmt.Println("\n\nSvcHost Service Names:")
	names, err := osdetail.GetAllServiceNames()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", names)
}

func printBinaryName(name, path string) {
	binName, err := osdetail.GetBinaryNameFromSystem(path)
	if err != nil {
		fmt.Printf("%s: ERROR: %s\n", name, err)
	} else {
		fmt.Printf("%s: %s\n", name, binName)
	}
}

func printBinaryIcon(name, path string) {
	binIcon, err := osdetail.GetBinaryIconFromSystem(path)
	if err != nil {
		fmt.Printf("%s: ERROR: %s\n", name, err)
	} else {
		fmt.Printf("%s: %s\n", name, binIcon)
	}
}
