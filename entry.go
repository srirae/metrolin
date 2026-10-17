package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

const GB = 1024 * 1024 * 1024

func checkWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func DiskPath() string {
	switch runtime.GOOS {
	case "windows":
		if drive := os.Getenv("SystemDrive"); drive != "" {
			return drive + "\\"
		}
		return "C:\\"
	case "linux":
		if checkWSL() == true {
			return "/mnt/c"
		}
		return "/"
	default:
		return "/"
	}
}

func allocRAM() string {
	var MaxRam int
	fmt.Print("Enter how much RAM you would like to allocate for metrolin")
	fmt.Scan(&MaxRam)
	for MaxRam >= (ramCheck.Total - ramCheck.Used) {
		fmt.Print("Please re-enter")

	}

}
func main() {
	ramCheck, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("RAM is running into an error:", err)
		return
	}
	path := DiskPath()
	diskCheck, err := disk.Usage(path)
	if err != nil {
		fmt.Println("Disk is running into an error", err)
		return
	}
	if checkWSL() == true {
		fmt.Println("Your current operating system is WSL")
	} else {
		fmt.Println("Your current operating system is", runtime.GOOS)
	}
	fmt.Printf("RAM %.1f GB total, %.1f GB used (%.1f%%)\n",
		float64(ramCheck.Total)/GB, float64(ramCheck.Used)/GB, ramCheck.UsedPercent)
	fmt.Printf("Disk %.1f GB total, %.1f GB used (%.1f%%)\n",
		float64(diskCheck.Total)/GB, float64(diskCheck.Used)/GB, diskCheck.UsedPercent)
}
