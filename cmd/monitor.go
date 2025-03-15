package cmd

import (
	"fmt"
	"time"

	"system-eye/internal/system"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"
)

const (
	maxDataPoints   = 30
	refreshInterval = 2 * time.Second
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start system monitoring",
	Long:  "Collects and displays system metrics (CPU, memory, disk) as live ASCII graphs.",
	Run: func(cmd *cobra.Command, args []string) {
		var cpuData, memData, diskData []float64

		for {
			cpuUsage, err := system.GetCPUUsage()
			if err != nil {
				fmt.Println("Error retrieving CPU usage:", err)
				return
			}
			memUsage, err := system.GetMemoryUsage()
			if err != nil {
				fmt.Println("Error retrieving Memory usage:", err)
				return
			}
			diskUsage, err := system.GetDiskUsage("/")
			if err != nil {
				fmt.Println("Error retrieving Disk usage:", err)
				return
			}

			cpuData = appendValue(cpuData, cpuUsage, maxDataPoints)
			memData = appendValue(memData, memUsage, maxDataPoints)
			diskData = appendValue(diskData, diskUsage, maxDataPoints)

			fmt.Print("\033[H\033[2J")

			cpuGraph := asciigraph.Plot(cpuData,
				asciigraph.Width(50),
				asciigraph.Caption("CPU Usage (%)"))
			memGraph := asciigraph.Plot(memData,
				asciigraph.Width(50),
				asciigraph.Caption("Memory Usage (%)"))
			diskGraph := asciigraph.Plot(diskData,
				asciigraph.Width(50),
				asciigraph.Caption("Disk Usage (%)"))

			fmt.Println(cpuGraph)
			fmt.Println()
			fmt.Println(memGraph)
			fmt.Println()
			fmt.Println(diskGraph)

			time.Sleep(refreshInterval)
		}
	},
}

func appendValue(data []float64, value float64, maxLen int) []float64 {
	if len(data) >= maxLen {
		data = data[1:]
	}
	return append(data, value)
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
