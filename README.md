# System-Eye

System-Eye is a CLI utility for monitoring system metrics such as CPU, memory, and disk usage. It provides both a live dashboard and historical data analysis using an interactive terminal UI.

## Features

- **Live Mode**: Displays real-time system metrics with a responsive terminal UI.
- **History Mode**: Loads historical data from a CSV file and displays graphs for a selected time range.
- **Configurable Settings**: Customize refresh interval and CPU usage threshold via an external configuration file.
- **Interactive Controls**:
  - `p`: Toggle pause/resume live updates.
  - `z`: Zoom in (decrease refresh interval).
  - `x`: Zoom out (increase refresh interval).
  - `e`: Export the current metric history to a CSV file.
  - `q` or `Ctrl+C`: Quit the dashboard.

## Installation

1. **Clone the repository:**
   ```sh
   git clone https://github.com/yourusername/system-eye.git
   cd system-eye
   ```
2. **Download dependencies:**
   ```sh
   go mod download
   ```
3. **Build the project:**
   ```sh
   go build -o system-eye
   ```

## Usage

### Live Mode

Launch the dashboard in live mode:
```sh
./system-eye dashboard --mode live --config ./config.yaml --csv path/to/export.csv
```

### History Mode

View historical data from a CSV file:
```sh
./system-eye dashboard --mode history --csv path/to/data.csv --from "2025-01-01T00:00:00Z" --to "2025-01-02T00:00:00Z"
```

## Configuration

The configuration file (e.g., `config.yaml`) lets you set the following parameters:
- **refresh_interval**: The dashboard update interval (e.g., "1s", "2s").
- **cpu_threshold**: The CPU usage percentage threshold for logging warnings.

Example configuration:
```yaml
refresh_interval: "1s"
cpu_threshold: 10.0
```

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests with improvements and bug fixes.
