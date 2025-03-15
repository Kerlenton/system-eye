package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"system-eye/internal/system"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds settings that can be loaded from a configuration file.
type Config struct {
	RefreshInterval time.Duration `mapstructure:"refresh_interval"` // e.g. "2s"
	CPUThreshold    float64       `mapstructure:"cpu_threshold"`    // e.g. 90.0 (%)
}

// Global configuration instance.
var config = Config{
	RefreshInterval: 2 * time.Second, // default 2 seconds
	CPUThreshold:    90.0,            // default threshold (percent)
}

// Metric represents system metrics along with a timestamp.
type Metric struct {
	Timestamp time.Time
	CPUUsage  float64
	MemUsage  float64
	DiskUsage float64
}

// MetricPlugin defines an interface for extensible metric plugins.
type MetricPlugin interface {
	// Name returns the plugin's name.
	Name() string
	// Fetch returns the plugin's metric value.
	Fetch() (float64, error)
}

// pluginRegistry holds all registered plugins.
var pluginRegistry []MetricPlugin

// RegisterPlugin adds a new plugin to the registry.
func RegisterPlugin(p MetricPlugin) {
	pluginRegistry = append(pluginRegistry, p)
}

var (
	mode            string
	csvPath         string
	fromTime        string
	toTime          string
	refreshInterval time.Duration // Overrides config.RefreshInterval if set via flag.
	configFile      string        // Path to configuration file.
)

// dashboardCmd defines the "dashboard" command.
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch interactive TUI dashboard for system monitoring",
	Long: `Launches an interactive TUI dashboard that shows live system metrics or historical data from a CSV file.

Modes:
  live    - Displays live system metrics in real time.
  history - Reads data from a CSV file and displays historical metrics.

Flags:
  -m, --mode       Mode of dashboard: "live" or "history" (default: "live").
  -c, --csv        CSV file path for exporting (live mode) or reading (history mode).
      --from       Start time for historical data (RFC3339 format).
      --to         End time for historical data (RFC3339 format).
  -i, --interval   Refresh interval for live mode (default: 2s).
      --config     Path to configuration file (YAML, JSON, etc).

Interactive Controls:
  p               Toggle pause/resume live updates.
  z               Zoom in (decrease refresh interval for more detail).
  x               Zoom out (increase refresh interval for a broader view).
  e               Export current live metrics to CSV.
  q, Ctrl+C       Quit the dashboard.
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration file if provided.
		if configFile != "" {
			if err := loadConfig(configFile); err != nil {
				log.Printf("Failed to load config file: %v", err)
			} else {
				log.Printf("Loaded configuration from %s", configFile)
			}
		}
		// If the refreshInterval flag is set, override the config.
		if refreshInterval != 0 {
			config.RefreshInterval = refreshInterval
		}
		// Run the appropriate mode.
		switch mode {
		case "live":
			runLiveDashboard()
		case "history":
			runHistoryDashboard()
		default:
			fmt.Println("Invalid mode. Use 'live' or 'history'.")
		}
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Flags().StringVarP(&mode, "mode", "m", "live", "Mode of dashboard: live or history")
	dashboardCmd.Flags().StringVarP(&csvPath, "csv", "c", "", "CSV file path for exporting (live mode) or reading (history mode)")
	dashboardCmd.Flags().StringVar(&fromTime, "from", "", "Start time for historical data (RFC3339 format)")
	dashboardCmd.Flags().StringVar(&toTime, "to", "", "End time for historical data (RFC3339 format)")
	dashboardCmd.Flags().DurationVarP(&refreshInterval, "interval", "i", 0, "Refresh interval for live mode (overrides config file)")
	dashboardCmd.Flags().StringVar(&configFile, "config", "", "Path to configuration file (YAML, JSON, etc)")
}

// loadConfig loads configuration values from the specified file using Viper.
func loadConfig(path string) error {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(&config)
}

// ensureMinData ensures that the slice has at least 2 data points to avoid termui errors.
func ensureMinData(data []float64) []float64 {
	if len(data) < 2 {
		if len(data) == 0 {
			return []float64{0, 0}
		}
		return []float64{data[0], data[0]}
	}
	return data
}

// createPlot creates and configures a Plot widget for termui.
func createPlot(title string, data []float64, x1, y1, x2, y2 int, lineColor ui.Color) *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = title
	plot.Data = [][]float64{ensureMinData(data)}
	plot.SetRect(x1, y1, x2, y2)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = lineColor
	return plot
}

// updateWidgetRects updates the rectangles for all widgets based on current terminal dimensions.
// It divides the terminal into a chart area (top) and a log area (bottom, fixed height).
func updateWidgetRects(cpuChart, memChart, diskChart *widgets.Plot, logWidget *widgets.List) {
	w, h := ui.TerminalDimensions()
	logHeight := 5 // Reserve 5 lines for logs.
	// CPU and Memory charts occupy the top half.
	cpuChart.SetRect(0, 0, w/2, h/2)
	memChart.SetRect(w/2, 0, w, h/2)
	// Disk chart occupies the area from half height to above the log area.
	diskChart.SetRect(0, h/2, w, h-logHeight)
	// Log widget occupies the bottom logHeight lines.
	logWidget.SetRect(0, h-logHeight, w, h)
}

// runLiveDashboard launches the interactive live dashboard with a dedicated log widget.
// It handles dynamic terminal resizing and interactive key events.
func runLiveDashboard() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Create a context for graceful shutdown using OS signals.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	var history []Metric
	var cpuData, memData, diskData []float64
	var logMessages []string // Holds recent log messages.

	// Create log widget.
	logWidget := widgets.NewList()
	logWidget.Title = "Logs"
	logWidget.Rows = []string{}
	logWidget.TextStyle.Fg = ui.ColorWhite

	// Helper function to append a log message and update the log widget.
	addLog := func(msg string) {
		// Append and keep only the last 10 messages.
		logMessages = append(logMessages, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
		if len(logMessages) > 10 {
			logMessages = logMessages[len(logMessages)-10:]
		}
		logWidget.Rows = logMessages
	}

	// Create plot widgets for charts.
	cpuChart := createPlot("CPU Usage (%)", cpuData, 0, 0, 50, 12, ui.ColorGreen)
	memChart := createPlot("Memory Usage (%)", memData, 50, 0, 100, 12, ui.ColorYellow)
	diskChart := createPlot("Disk Usage (%)", diskData, 0, 12, 100, 24, ui.ColorCyan)

	// Set initial widget sizes based on terminal dimensions.
	updateWidgetRects(cpuChart, memChart, diskChart, logWidget)

	// Use refresh interval from config.
	ticker := time.NewTicker(config.RefreshInterval)
	defer ticker.Stop()

	uiEvents := ui.PollEvents()
	paused := false

loop:
	for {
		select {
		case <-ctx.Done():
			addLog("Shutdown signal received. Exiting live dashboard.")
			break loop
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				addLog("Quit event received.")
				break loop
			case "p":
				paused = !paused
				if paused {
					addLog("Live updates paused.")
				} else {
					addLog("Live updates resumed.")
				}
			case "z":
				// Zoom in: decrease refresh interval to show more detail.
				config.RefreshInterval = config.RefreshInterval / 2
				addLog(fmt.Sprintf("Zoom in: new refresh interval: %v", config.RefreshInterval))
				ticker.Reset(config.RefreshInterval)
			case "x":
				// Zoom out: increase refresh interval for a broader view.
				config.RefreshInterval = config.RefreshInterval * 2
				addLog(fmt.Sprintf("Zoom out: new refresh interval: %v", config.RefreshInterval))
				ticker.Reset(config.RefreshInterval)
			case "e":
				// Export CSV immediately when 'e' is pressed.
				if csvPath != "" {
					if err := exportHistoryToCSV(csvPath, history); err != nil {
						addLog(fmt.Sprintf("Failed to export CSV: %v", err))
					} else {
						addLog(fmt.Sprintf("Exported history to %s", csvPath))
					}
				}
			case "<Resize>":
				updateWidgetRects(cpuChart, memChart, diskChart, logWidget)
			}
		case <-ticker.C:
			if paused {
				continue
			}
			// Retrieve metrics concurrently.
			var wg sync.WaitGroup
			var cpuUsage, memUsage, diskUsage float64
			var errCPU, errMem, errDisk error

			wg.Add(3)
			go func() {
				defer wg.Done()
				cpuUsage, errCPU = system.GetCPUUsage()
			}()
			go func() {
				defer wg.Done()
				memUsage, errMem = system.GetMemoryUsage()
			}()
			go func() {
				defer wg.Done()
				diskUsage, errDisk = system.GetDiskUsage("/")
			}()
			wg.Wait()

			// Handle any errors from metric collection.
			if errCPU != nil {
				addLog(fmt.Sprintf("Error retrieving CPU usage: %v", errCPU))
				continue
			}
			if errMem != nil {
				addLog(fmt.Sprintf("Error retrieving Memory usage: %v", errMem))
				continue
			}
			if errDisk != nil {
				addLog(fmt.Sprintf("Error retrieving Disk usage: %v", errDisk))
				continue
			}

			// Log notification if CPU usage exceeds threshold.
			if cpuUsage > config.CPUThreshold {
				addLog(fmt.Sprintf("High CPU usage detected: %.2f%%", cpuUsage))
				// (Optional: integrate with a notification system here)
			}

			now := time.Now()
			history = append(history, Metric{
				Timestamp: now,
				CPUUsage:  cpuUsage,
				MemUsage:  memUsage,
				DiskUsage: diskUsage,
			})

			// Maintain only the latest 30 data points.
			cpuData = appendValue(cpuData, cpuUsage, 30)
			memData = appendValue(memData, memUsage, 30)
			diskData = appendValue(diskData, diskUsage, 30)

			// Update chart data.
			cpuChart.Data = [][]float64{ensureMinData(cpuData)}
			memChart.Data = [][]float64{ensureMinData(memData)}
			diskChart.Data = [][]float64{ensureMinData(diskData)}

			// Render all widgets (charts and logs).
			ui.Render(cpuChart, memChart, diskChart, logWidget)
		}
	}

	// On exit, export CSV if the path is provided.
	if csvPath != "" {
		if err := exportHistoryToCSV(csvPath, history); err != nil {
			addLog(fmt.Sprintf("Failed to export CSV on shutdown: %v", err))
		} else {
			addLog(fmt.Sprintf("Exported history to %s", csvPath))
		}
		// Render final state to show the export message.
		ui.Render(logWidget)
	}
}

// runHistoryDashboard reads a CSV file and displays charts for the specified time range.
func runHistoryDashboard() {
	if csvPath == "" {
		fmt.Println("CSV file path must be provided in history mode.")
		return
	}
	var start, end time.Time
	var err error
	if fromTime != "" {
		start, err = time.Parse(time.RFC3339, fromTime)
		if err != nil {
			fmt.Println("Invalid 'from' time:", err)
			return
		}
	}
	if toTime != "" {
		end, err = time.Parse(time.RFC3339, toTime)
		if err != nil {
			fmt.Println("Invalid 'to' time:", err)
			return
		}
	}

	history, err := readHistoryFromCSV(csvPath)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	// Filter data based on the specified time range.
	var filtered []Metric
	for _, m := range history {
		if (!start.IsZero() && m.Timestamp.Before(start)) || (!end.IsZero() && m.Timestamp.After(end)) {
			continue
		}
		filtered = append(filtered, m)
	}
	if len(filtered) == 0 {
		fmt.Println("No data available for the specified time range.")
		return
	}

	var cpuData, memData, diskData []float64
	for _, m := range filtered {
		cpuData = append(cpuData, m.CPUUsage)
		memData = append(memData, m.MemUsage)
		diskData = append(diskData, m.DiskUsage)
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	cpuChart := createPlot(fmt.Sprintf("CPU Usage (%%) [%s - %s]", fromTime, toTime), cpuData, 0, 0, 50, 12, ui.ColorGreen)
	memChart := createPlot(fmt.Sprintf("Memory Usage (%%) [%s - %s]", fromTime, toTime), memData, 50, 0, 100, 12, ui.ColorYellow)
	diskChart := createPlot(fmt.Sprintf("Disk Usage (%%) [%s - %s]", fromTime, toTime), diskData, 0, 12, 100, 24, ui.ColorCyan)

	ui.Render(cpuChart, memChart, diskChart)

	// Wait for the user to quit (q or Ctrl+C).
	for e := range ui.PollEvents() {
		if e.ID == "q" || e.ID == "<C-c>" {
			break
		}
	}
}

// appendValue adds a new value to the slice, maintaining a maximum length of maxLen.
func appendValue(data []float64, value float64, maxLen int) []float64 {
	if len(data) >= maxLen {
		data = data[1:]
	}
	return append(data, value)
}

// exportHistoryToCSV writes the historical metrics to a CSV file.
// Data is first written to a temporary file and then atomically renamed to reduce data corruption risk.
func exportHistoryToCSV(filename string, history []Metric) error {
	dir, file := filepath.Split(filename)
	tmpFile, err := os.CreateTemp(dir, file)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(tmpFile)
	if err := writer.Write([]string{"timestamp", "cpu_usage", "mem_usage", "disk_usage"}); err != nil {
		tmpFile.Close()
		return err
	}
	for _, m := range history {
		record := []string{
			m.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.2f", m.CPUUsage),
			fmt.Sprintf("%.2f", m.MemUsage),
			fmt.Sprintf("%.2f", m.DiskUsage),
		}
		if err := writer.Write(record); err != nil {
			tmpFile.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()
	// Atomically move the temporary file.
	return os.Rename(tmpFile.Name(), filename)
}

// readHistoryFromCSV reads metrics history from a CSV file.
func readHistoryFromCSV(filename string) ([]Metric, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var history []Metric
	reader := csv.NewReader(file)
	// Skip the header.
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 4 {
			continue
		}
		t, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			continue
		}
		cpuVal, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}
		memVal, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}
		diskVal, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			continue
		}
		history = append(history, Metric{
			Timestamp: t,
			CPUUsage:  cpuVal,
			MemUsage:  memVal,
			DiskUsage: diskVal,
		})
	}
	return history, nil
}
