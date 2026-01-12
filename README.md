# FIO-QA - Disk Performance Testing Tool

A Golang tool for automated disk performance testing using FIO.

## Requirements

- Go 1.21+
- FIO 3.38+
- Linux (recommended for libaio support)

## Installation

```bash
# Install FIO
sudo apt-get install fio  # Ubuntu/Debian
sudo yum install fio      # RHEL/CentOS

# Build the tool
go mod download
go build -o fio-qa main.go
```

## Usage

```bash
./fio-qa
```

The tool will:
1. Load test cases from `fio-testcases.json`
2. Run each test sequentially
3. Display detailed results in formatted tables
4. Show overall summary at the end
5. Save complete results to `test_results-<timestamp>.json`

## Test Cases

All tests run for 10 seconds each using libaio engine with direct I/O:

1. **Random Read IOPS** - 4k blocks, iodepth=256, numjobs=4
2. **Sequential Read IOPS** - 4k blocks, iodepth=256, numjobs=4
3. **Random Read/Write IOPS** - 4k blocks, iodepth=256, numjobs=4, mixed workload
4. **Random Read Latency** - 4k blocks, iodepth=1, numjobs=1, single thread
5. **Random Read/Write Latency** - 4k blocks, iodepth=1, numjobs=1, single thread, mixed workload
6. **Random Read Throughput** - 64k blocks, iodepth=64, numjobs=4
7. **Random Read/Write Throughput** - 64k blocks, iodepth=64, numjobs=4, mixed workload
8. **Sequential Read Throughput** - 64k blocks, iodepth=64, numjobs=4

## Output

### Terminal Output

Each test displays detailed tables with:
- **Test Information**: Status, name, description, duration
- **IOPS Statistics**: Read/Write IOPS with min, max, avg, stddev
- **Bandwidth Statistics**: Read/Write bandwidth (MB/s) with min, max, avg
- **Latency Statistics**:
  - Submission latency (slat)
  - Completion latency (clat)
  - Total latency
  - All in microseconds with min, max, avg, stddev
- **Latency Percentiles**: p1, p5, p10, p20, p30, p40, p50, p60, p70, p80, p90, p95, p99, p99.5, p99.9, p99.95, p99.99
- **CPU Usage**: User/System CPU %, context switches, faults
- **Disk Utilization**: Device stats, read/write IOs, sectors, utilization %

Final summary includes:
- Total tests passed/failed
- Performance comparison table
- Performance highlights (highest IOPS, highest bandwidth, lowest latency)

All tables are perfectly aligned for easy reading.

### JSON Output

Results are automatically saved to `test_results-YYYY-MM-DD-HHMMSS.json` with complete data:

```json
{
  "summary": {
    "total_tests": 8,
    "passed": 8,
    "failed": 0,
    "total_duration": "1m23s"
  },
  "test_results": [
    {
      "test_name": "iops_and_bw_for_rand_reads",
      "description": "IOPS and Bandwidth for Random Reads",
      "status": "PASSED",
      "duration": "10s",
      "iops": 738406,
      "bandwidth_mbps": 2884.40,
      "latency_us": 1385.97,
      "iops_stats": {
        "read": {
          "iops": 738406,
          "min": 734254,
          "max": 740376,
          "avg": 738182,
          "stddev": 59
        },
        "write": { ... },
        "total": 738406
      },
      "bandwidth_stats": { ... },
      "latency_stats": {
        "read": {
          "submission_latency_us": { "min": 5.91, "max": 4997.38, "avg": 11.89, "stddev": 22.44 },
          "completion_latency_us": { "min": 575.26, "max": 13717.06, "avg": 4627.78, "stddev": 744.46 },
          "total_latency_us": { ... }
        },
        "write": { ... }
      },
      "latency_percentiles": {
        "p1": 3883.01,
        "p50": 4423.68,
        "p95": 6586.37,
        "p99": 7897.09,
        "p99_9": 8716.29,
        "p99_99": 10420.22
      },
      "cpu_usage": {
        "user_cpu_percent": 3.93,
        "system_cpu_percent": 19.36,
        "context_switches": 413843,
        "major_faults": 0,
        "minor_faults": 4141
      },
      "disk_utilization": [
        {
          "device": "nvme0n1",
          "read_ios": 545964,
          "write_ios": 73,
          "read_sectors": 69858816,
          "write_sectors": 2008,
          "utilization_percent": 70.39
        }
      ]
    }
  ],
  "performance_highlights": {
    "highest_iops": {
      "test_name": "iops_and_bw_for_seq_reads",
      "value": 757426,
      "unit": "iops"
    },
    "highest_bandwidth": {
      "test_name": "throughput_for_seq_reads",
      "value": 3446.81,
      "unit": "MB/s"
    },
    "lowest_latency": {
      "test_name": "latency_for_random_reads_and_writes",
      "value": 53.29,
      "unit": "μs"
    }
  }
}
```

Each test run creates a new timestamped JSON file, allowing you to track performance over time.

## Configuration

Edit `fio-testcases.json` to customize tests:

```json
{
  "tests": [
    {
      "name": "test_name",
      "description": "Test Description",
      "size": "1G",
      "rw": "randread",
      "bs": "4k",
      "runtime": 120
    }
  ]
}
```

## Cleanup

```bash
# Remove test files and logs
rm -f *.log fio.*.log

# Remove JSON results (optional - you may want to keep these for analysis)
rm -f test_results-*.json
```

## Features

- Comprehensive performance testing with 8 predefined test cases
- Detailed metrics: IOPS, bandwidth, latency (submission, completion, total)
- Latency percentile analysis (p1 to p99.99)
- CPU usage and disk utilization tracking
- Perfectly aligned table output for easy reading
- Automated JSON export with timestamps for historical tracking
- Customizable test configurations via JSON

## License

MIT
