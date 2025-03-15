package cmd

import (
	"fmt"
	"time"

	"system-eye/internal/system"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start system monitoring",
	Long:  "Starts collecting and displaying system metrics (CPU, memory, disk) in real time with beautiful ASCII graphs.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting system monitoring...")
		const maxDataPoints = 30
		var cpuData, memData, diskData []float64

		for {
			cpuUsage, err := system.GetCPUUsage()
			if err != nil {
				fmt.Println("Error getting CPU usage:", err)
				return
			}
			memUsage, err := system.GetMemoryUsage()
			if err != nil {
				fmt.Println("Error getting memory usage:", err)
				return
			}
			diskUsage, err := system.GetDiskUsage("/")
			if err != nil {
				fmt.Println("Error getting disk usage:", err)
				return
			}

			if len(cpuData) >= maxDataPoints {
				cpuData = cpuData[1:]
			}
			if len(memData) >= maxDataPoints {
				memData = memData[1:]
			}
			if len(diskData) >= maxDataPoints {
				diskData = diskData[1:]
			}

			cpuData = append(cpuData, cpuUsage)
			memData = append(memData, memUsage)
			diskData = append(diskData, diskUsage)

			fmt.Print("\033[H\033[2J")

			cpuGraph := asciigraph.Plot(cpuData, asciigraph.Caption("CPU Usage (%)"))
			memGraph := asciigraph.Plot(memData, asciigraph.Caption("Memory Usage (%)"))
			diskGraph := asciigraph.Plot(diskData, asciigraph.Caption("Disk Usage (%)"))

			fmt.Println(cpuGraph)
			fmt.Println()
			fmt.Println(memGraph)
			fmt.Println()
			fmt.Println(diskGraph)

			time.Sleep(time.Microsecond)
		}
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
