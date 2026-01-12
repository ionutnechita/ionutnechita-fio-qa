package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

// FioTest represents a single fio test configuration
type FioTest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Filename       string `json:"filename"`
	Size           string `json:"size"`
	Direct         int    `json:"direct"`
	RW             string `json:"rw"`
	BS             string `json:"bs"`
	IOEngine       string `json:"ioengine"`
	IODepth        int    `json:"iodepth"`
	NumJobs        int    `json:"numjobs"`
	TimeBased      bool   `json:"time_based"`
	GroupReporting bool   `json:"group_reporting"`
	Runtime        int    `json:"runtime"`
	EtaNewline     int    `json:"eta_newline"`
}

// TestCases represents the structure of the JSON file
type TestCases struct {
	Tests []FioTest `json:"tests"`
}

// FioJobResult represents the result of a single fio job
type FioJobResult struct {
	JobName   string     `json:"jobname"`
	Read      FioIO      `json:"read"`
	Write     FioIO      `json:"write"`
	Sync      FioSync    `json:"sync"`
	UsrCPU    float64    `json:"usr_cpu"`
	SysCPU    float64    `json:"sys_cpu"`
	Ctx       int64      `json:"ctx"`
	MajF      int64      `json:"majf"`
	MinF      int64      `json:"minf"`
	IODepths  map[string]float64 `json:"iodepth_level"`
	LatBins   map[string]float64 `json:"latency_ns"`
}

// FioIO represents read or write statistics
type FioIO struct {
	IOPS          float64   `json:"iops"`
	BWBytes       float64   `json:"bw_bytes"`      // Bandwidth in bytes/sec
	BWMean        float64   `json:"bw_mean"`       // Bandwidth mean in KiB/s
	BWMin         float64   `json:"bw_min"`        // Bandwidth min in KiB/s
	BWMax         float64   `json:"bw_max"`        // Bandwidth max in KiB/s
	BWDev         float64   `json:"bw_dev"`        // Bandwidth deviation in KiB/s
	IOKBytes      float64   `json:"io_kbytes"`
	Runtime       float64   `json:"runtime"`
	Slat          FioLatNs  `json:"slat_ns"`
	Clat          FioClat   `json:"clat_ns"`
	LatNs         FioLatNs  `json:"lat_ns"`
	IOPSMin       float64   `json:"iops_min"`
	IOPSMax       float64   `json:"iops_max"`
	IOPSMean      float64   `json:"iops_mean"`
	IOPSStddev    float64   `json:"iops_stddev"`
}

// FioLatNs represents latency in nanoseconds
type FioLatNs struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Mean       float64 `json:"mean"`
	Stddev     float64 `json:"stddev"`
	Percentile map[string]float64 `json:"percentile"`
}

// FioClat represents completion latency
type FioClat struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Mean       float64 `json:"mean"`
	Stddev     float64 `json:"stddev"`
	Percentile map[string]float64 `json:"percentile"`
}

// FioSync represents sync statistics
type FioSync struct {
	LatNs FioLatNs `json:"lat_ns"`
}

// FioDiskUtil represents disk utilization statistics
type FioDiskUtil struct {
	Name        string  `json:"name"`
	ReadIOs     int64   `json:"read_ios"`
	WriteIOs    int64   `json:"write_ios"`
	ReadSectors int64   `json:"read_sectors"`
	WriteSectors int64  `json:"write_sectors"`
	ReadMerges  int64   `json:"read_merges"`
	WriteMerges int64   `json:"write_merges"`
	ReadTicks   int64   `json:"read_ticks"`
	WriteTicks  int64   `json:"write_ticks"`
	InQueue     int64   `json:"in_queue"`
	Util        float64 `json:"util"`
}

// FioOutput represents the complete fio JSON output
type FioOutput struct {
	FioVersion string         `json:"fio version"`
	Jobs       []FioJobResult `json:"jobs"`
	DiskUtil   []FioDiskUtil  `json:"disk_util"`
}

// TestResult stores the parsed results from a test
type TestResult struct {
	TestName       string
	Description    string
	ReadIOPS       float64
	WriteIOPS      float64
	TotalIOPS      float64
	ReadBWMBps     float64
	WriteBWMBps    float64
	TotalBWMBps    float64
	ReadLatencyUs  float64
	WriteLatencyUs float64
	AvgLatencyUs   float64
	Duration       time.Duration
	Status         string
	Error          error
	FioJob         *FioJobResult
	DiskUtil       []FioDiskUtil
}

func main() {
	fmt.Println("=== FIO Disk Performance Testing Tool ===")
	fmt.Println()

	// Check if fio is installed
	if !checkFioInstalled() {
		fmt.Println("Error: fio is not installed or not in PATH")
		fmt.Println("Please install fio before running this tool")
		os.Exit(1)
	}

	// Load test cases
	testCases, err := loadTestCases("fio-testcases.json")
	if err != nil {
		fmt.Printf("Error loading test cases: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d test cases\n", len(testCases.Tests))
	fmt.Println()

	// Run all tests and collect results
	var results []TestResult
	for i, test := range testCases.Tests {
		fmt.Printf("[%d/%d] Running test: %s\n", i+1, len(testCases.Tests), test.Description)
		fmt.Println(strings.Repeat("=", 80))

		result := runTest(test)
		results = append(results, result)

		// Display individual test result
		displayTestResult(result)
		fmt.Println()
	}

	// Display summary of all tests
	displaySummary(results)

	// Save results to JSON file with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("test_results-%s.json", timestamp)
	err = saveResultsToJSON(results, filename)
	if err != nil {
		fmt.Printf("Warning: Failed to save results to JSON: %v\n", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", filename)
	}
}

func checkFioInstalled() bool {
	cmd := exec.Command("fio", "--version")
	err := cmd.Run()
	return err == nil
}

func loadTestCases(filename string) (*TestCases, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var testCases TestCases
	err = json.Unmarshal(data, &testCases)
	if err != nil {
		return nil, err
	}

	return &testCases, nil
}

func runTest(test FioTest) TestResult {
	result := TestResult{
		TestName:    test.Name,
		Description: test.Description,
		Status:      "FAILED",
	}

	start := time.Now()

	// Build fio command
	args := buildFioCommand(test)

	// Create temporary file for JSON output
	tmpFile := fmt.Sprintf("/tmp/fio_output_%s_%d.json", test.Name, time.Now().Unix())
	args = append(args, "--output-format=json", fmt.Sprintf("--output=%s", tmpFile))

	// Run fio command
	cmd := exec.Command("fio", args...)
	output, err := cmd.CombinedOutput()

	result.Duration = time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("fio command failed: %v\nOutput: %s", err, string(output))
		return result
	}

	// Parse JSON output
	fioOutput, err := parseFioOutput(tmpFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse fio output: %v", err)
		return result
	}

	// Extract metrics
	if len(fioOutput.Jobs) > 0 {
		job := fioOutput.Jobs[0]

		result.ReadIOPS = job.Read.IOPS
		result.WriteIOPS = job.Write.IOPS
		result.TotalIOPS = result.ReadIOPS + result.WriteIOPS

		result.ReadBWMBps = float64(job.Read.BWBytes) / 1024 / 1024
		result.WriteBWMBps = float64(job.Write.BWBytes) / 1024 / 1024
		result.TotalBWMBps = result.ReadBWMBps + result.WriteBWMBps

		// Convert latency from ns to us
		result.ReadLatencyUs = job.Read.LatNs.Mean / 1000
		result.WriteLatencyUs = job.Write.LatNs.Mean / 1000

		if result.ReadIOPS > 0 && result.WriteIOPS > 0 {
			result.AvgLatencyUs = (result.ReadLatencyUs + result.WriteLatencyUs) / 2
		} else if result.ReadIOPS > 0 {
			result.AvgLatencyUs = result.ReadLatencyUs
		} else {
			result.AvgLatencyUs = result.WriteLatencyUs
		}

		// Store full job result and disk util
		result.FioJob = &job
		result.DiskUtil = fioOutput.DiskUtil

		result.Status = "PASSED"
	}

	// Clean up temp file
	os.Remove(tmpFile)

	return result
}

func buildFioCommand(test FioTest) []string {
	args := []string{
		fmt.Sprintf("--filename=%s", test.Filename),
		fmt.Sprintf("--size=%s", test.Size),
		fmt.Sprintf("--direct=%d", test.Direct),
		fmt.Sprintf("--rw=%s", test.RW),
		fmt.Sprintf("--bs=%s", test.BS),
		fmt.Sprintf("--ioengine=%s", test.IOEngine),
		fmt.Sprintf("--iodepth=%d", test.IODepth),
		fmt.Sprintf("--numjobs=%d", test.NumJobs),
		fmt.Sprintf("--name=%s", test.Name),
		fmt.Sprintf("--runtime=%d", test.Runtime),
		fmt.Sprintf("--eta-newline=%d", test.EtaNewline),
	}

	if test.TimeBased {
		args = append(args, "--time_based")
	}

	if test.GroupReporting {
		args = append(args, "--group_reporting")
	}

	return args
}

func parseFioOutput(filename string) (*FioOutput, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var fioOutput FioOutput
	err = json.Unmarshal(data, &fioOutput)
	if err != nil {
		return nil, err
	}

	return &fioOutput, nil
}

func displayTestResult(result TestResult) {
	if result.Status == "FAILED" {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Metric", "Value"})
		configureTable(table, 2)
		table.Append([]string{"Status", fmt.Sprintf("❌ %s", result.Status)})
		if result.Error != nil {
			table.Append([]string{"Error", result.Error.Error()})
		}
		table.Render()
		return
	}

	job := result.FioJob

	// Test Info
	fmt.Println("Test Information")
	infoTable := tablewriter.NewWriter(os.Stdout)
	infoTable.SetHeader([]string{"Metric", "Value"})
	configureTable(infoTable, 2)
	infoTable.Append([]string{"Status", "✅ PASSED"})
	infoTable.Append([]string{"Test Name", result.TestName})
	infoTable.Append([]string{"Description", result.Description})
	infoTable.Append([]string{"Duration", result.Duration.Round(time.Second).String()})
	infoTable.Render()
	fmt.Println()

	// IOPS Statistics
	fmt.Println("IOPS Statistics")
	iopsTable := tablewriter.NewWriter(os.Stdout)
	iopsTable.SetHeader([]string{"", "Read", "Write", "Total"})
	configureTable(iopsTable, 4)
	iopsTable.Append([]string{"IOPS", fmt.Sprintf("%.0f", result.ReadIOPS), fmt.Sprintf("%.0f", result.WriteIOPS), fmt.Sprintf("%.0f", result.TotalIOPS)})
	if job != nil {
		iopsTable.Append([]string{"IOPS Min", fmt.Sprintf("%.0f", job.Read.IOPSMin), fmt.Sprintf("%.0f", job.Write.IOPSMin), "-"})
		iopsTable.Append([]string{"IOPS Max", fmt.Sprintf("%.0f", job.Read.IOPSMax), fmt.Sprintf("%.0f", job.Write.IOPSMax), "-"})
		iopsTable.Append([]string{"IOPS Avg", fmt.Sprintf("%.0f", job.Read.IOPSMean), fmt.Sprintf("%.0f", job.Write.IOPSMean), "-"})
		iopsTable.Append([]string{"IOPS StdDev", fmt.Sprintf("%.0f", job.Read.IOPSStddev), fmt.Sprintf("%.0f", job.Write.IOPSStddev), "-"})
	}
	iopsTable.Render()
	fmt.Println()

	// Bandwidth Statistics
	fmt.Println("Bandwidth Statistics")
	bwTable := tablewriter.NewWriter(os.Stdout)
	bwTable.SetHeader([]string{"", "Read (MB/s)", "Write (MB/s)", "Total (MB/s)"})
	configureTable(bwTable, 4)
	bwTable.Append([]string{"Bandwidth", fmt.Sprintf("%.2f", result.ReadBWMBps), fmt.Sprintf("%.2f", result.WriteBWMBps), fmt.Sprintf("%.2f", result.TotalBWMBps)})
	if job != nil {
		bwTable.Append([]string{"BW Min", fmt.Sprintf("%.2f", job.Read.BWMin/1024), fmt.Sprintf("%.2f", job.Write.BWMin/1024), "-"})
		bwTable.Append([]string{"BW Max", fmt.Sprintf("%.2f", job.Read.BWMax/1024), fmt.Sprintf("%.2f", job.Write.BWMax/1024), "-"})
		bwTable.Append([]string{"BW Avg", fmt.Sprintf("%.2f", job.Read.BWMean/1024), fmt.Sprintf("%.2f", job.Write.BWMean/1024), "-"})
	}
	bwTable.Render()
	fmt.Println()

	// Latency Statistics
	fmt.Println("Latency Statistics (microseconds)")
	latTable := tablewriter.NewWriter(os.Stdout)
	latTable.SetHeader([]string{"", "Read", "Write"})
	configureTable(latTable, 3)
	if job != nil {
		latTable.Append([]string{"Submission Lat (slat) Min", fmt.Sprintf("%.2f", job.Read.Slat.Min/1000), fmt.Sprintf("%.2f", job.Write.Slat.Min/1000)})
		latTable.Append([]string{"Submission Lat (slat) Max", fmt.Sprintf("%.2f", job.Read.Slat.Max/1000), fmt.Sprintf("%.2f", job.Write.Slat.Max/1000)})
		latTable.Append([]string{"Submission Lat (slat) Avg", fmt.Sprintf("%.2f", job.Read.Slat.Mean/1000), fmt.Sprintf("%.2f", job.Write.Slat.Mean/1000)})
		latTable.Append([]string{"Submission Lat (slat) StdDev", fmt.Sprintf("%.2f", job.Read.Slat.Stddev/1000), fmt.Sprintf("%.2f", job.Write.Slat.Stddev/1000)})
		latTable.Append([]string{"", "", ""})
		latTable.Append([]string{"Completion Lat (clat) Min", fmt.Sprintf("%.2f", job.Read.Clat.Min/1000), fmt.Sprintf("%.2f", job.Write.Clat.Min/1000)})
		latTable.Append([]string{"Completion Lat (clat) Max", fmt.Sprintf("%.2f", job.Read.Clat.Max/1000), fmt.Sprintf("%.2f", job.Write.Clat.Max/1000)})
		latTable.Append([]string{"Completion Lat (clat) Avg", fmt.Sprintf("%.2f", job.Read.Clat.Mean/1000), fmt.Sprintf("%.2f", job.Write.Clat.Mean/1000)})
		latTable.Append([]string{"Completion Lat (clat) StdDev", fmt.Sprintf("%.2f", job.Read.Clat.Stddev/1000), fmt.Sprintf("%.2f", job.Write.Clat.Stddev/1000)})
		latTable.Append([]string{"", "", ""})
		latTable.Append([]string{"Total Lat Min", fmt.Sprintf("%.2f", job.Read.LatNs.Min/1000), fmt.Sprintf("%.2f", job.Write.LatNs.Min/1000)})
		latTable.Append([]string{"Total Lat Max", fmt.Sprintf("%.2f", job.Read.LatNs.Max/1000), fmt.Sprintf("%.2f", job.Write.LatNs.Max/1000)})
		latTable.Append([]string{"Total Lat Avg", fmt.Sprintf("%.2f", job.Read.LatNs.Mean/1000), fmt.Sprintf("%.2f", job.Write.LatNs.Mean/1000)})
		latTable.Append([]string{"Total Lat StdDev", fmt.Sprintf("%.2f", job.Read.LatNs.Stddev/1000), fmt.Sprintf("%.2f", job.Write.LatNs.Stddev/1000)})
	}
	latTable.Render()
	fmt.Println()

	// Completion Latency Percentiles
	if job != nil && len(job.Read.Clat.Percentile) > 0 {
		fmt.Println("Completion Latency Percentiles (microseconds) - Read")
		percTable := tablewriter.NewWriter(os.Stdout)
		percTable.SetHeader([]string{"Percentile", "Latency (μs)"})
		configureTable(percTable, 2)

		// Sort and display key percentiles
		percentiles := []string{"1.000000", "5.000000", "10.000000", "20.000000", "30.000000", "40.000000",
		                       "50.000000", "60.000000", "70.000000", "80.000000", "90.000000", "95.000000",
		                       "99.000000", "99.500000", "99.900000", "99.950000", "99.990000"}
		for _, p := range percentiles {
			if val, ok := job.Read.Clat.Percentile[p]; ok {
				percTable.Append([]string{fmt.Sprintf("p%.2f", parseFloat(p)), fmt.Sprintf("%.2f", val/1000)})
			}
		}
		percTable.Render()
		fmt.Println()
	}

	// CPU Usage
	if job != nil {
		fmt.Println("CPU Usage")
		cpuTable := tablewriter.NewWriter(os.Stdout)
		cpuTable.SetHeader([]string{"Metric", "Value"})
		configureTable(cpuTable, 2)
		cpuTable.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
		cpuTable.Append([]string{"User CPU", fmt.Sprintf("%.2f%%", job.UsrCPU)})
		cpuTable.Append([]string{"System CPU", fmt.Sprintf("%.2f%%", job.SysCPU)})
		cpuTable.Append([]string{"Context Switches", fmt.Sprintf("%d", job.Ctx)})
		cpuTable.Append([]string{"Major Faults", fmt.Sprintf("%d", job.MajF)})
		cpuTable.Append([]string{"Minor Faults", fmt.Sprintf("%d", job.MinF)})
		cpuTable.Render()
		fmt.Println()
	}

	// Disk Utilization
	if len(result.DiskUtil) > 0 {
		fmt.Println("Disk Utilization")
		diskTable := tablewriter.NewWriter(os.Stdout)
		diskTable.SetHeader([]string{"Device", "Rd IOPS", "Wr IOPS", "Rd Sectors", "Wr Sectors", "Util%"})
		configureTable(diskTable, 6)
		diskTable.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
		for _, disk := range result.DiskUtil {
			diskTable.Append([]string{
				disk.Name,
				fmt.Sprintf("%d", disk.ReadIOs),
				fmt.Sprintf("%d", disk.WriteIOs),
				fmt.Sprintf("%d", disk.ReadSectors),
				fmt.Sprintf("%d", disk.WriteSectors),
				fmt.Sprintf("%.2f%%", disk.Util),
			})
		}
		diskTable.Render()
		fmt.Println()
	}
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func getPercentile(percentiles map[string]float64, key string) float64 {
	if val, ok := percentiles[key]; ok {
		return val
	}
	return 0
}

// configureTable sets uniform table styling with fixed width
// All tables will have the same total width for perfect alignment
func configureTable(table *tablewriter.Table, colCount int) {
	table.SetBorder(true)
	table.SetRowLine(true)
	table.SetAutoWrapText(false)
	table.SetColumnSeparator("│")
	table.SetCenterSeparator("┼")
	table.SetRowSeparator("─")

	// Set fixed column widths to ensure all tables have identical total width
	// Target: all tables aligned at ~115 characters total width
	// Each cell format: │ space content space │
	switch colCount {
	case 2: // Test Info, CPU Usage, Percentiles
		table.SetColMinWidth(0, 40)
		table.SetColMinWidth(1, 66)
		// Total: 40 + 66 + 9 separators ≈ 115 chars
	case 3: // Latency (3 cols), Highlights
		table.SetColMinWidth(0, 40)
		table.SetColMinWidth(1, 32)
		table.SetColMinWidth(2, 31)
		// Total: 40 + 32 + 31 + 4 separators = 113 chars (aligned with 2-col tables)
	case 4: // IOPS, Bandwidth
		table.SetColMinWidth(0, 40)
		table.SetColMinWidth(1, 20)
		table.SetColMinWidth(2, 20)
		table.SetColMinWidth(3, 20)
		// Total: 40 + 20*3 + 15 separators ≈ 115 chars
	case 6: // Details, Disk Utilization
		table.SetColMinWidth(0, 40)
		table.SetColMinWidth(1, 11)
		table.SetColMinWidth(2, 11)
		table.SetColMinWidth(3, 11)
		table.SetColMinWidth(4, 11)
		table.SetColMinWidth(5, 10)
		// Total: 40 + 11*4 + 10 + 21 separators ≈ 115 chars (aligned with CPU table)
	}
}

// JSONResults represents the complete test results in JSON format
type JSONResults struct {
	Summary            JSONSummary            `json:"summary"`
	TestResults        []JSONTestResult       `json:"test_results"`
	PerformanceHighlights JSONPerformanceHighlights `json:"performance_highlights"`
}

// JSONSummary represents the overall summary statistics
type JSONSummary struct {
	TotalTests    int    `json:"total_tests"`
	Passed        int    `json:"passed"`
	Failed        int    `json:"failed"`
	TotalDuration string `json:"total_duration"`
}

// JSONTestResult represents a single test result for JSON output
type JSONTestResult struct {
	TestName       string                `json:"test_name"`
	Description    string                `json:"description"`
	Status         string                `json:"status"`
	Duration       string                `json:"duration"`
	IOPS           float64               `json:"iops"`
	BandwidthMBps  float64               `json:"bandwidth_mbps"`
	LatencyUs      float64               `json:"latency_us"`
	IOPSStats      JSONIOPSStats         `json:"iops_stats"`
	BandwidthStats JSONBandwidthStats    `json:"bandwidth_stats"`
	LatencyStats   JSONLatencyStats      `json:"latency_stats"`
	Percentiles    JSONPercentiles       `json:"latency_percentiles,omitempty"`
	CPUUsage       JSONCPUUsage          `json:"cpu_usage,omitempty"`
	DiskUtil       []JSONDiskUtil        `json:"disk_utilization,omitempty"`
	Error          string                `json:"error,omitempty"`
}

// JSONIOPSStats represents IOPS statistics
type JSONIOPSStats struct {
	Read  JSONIOPSDetail `json:"read"`
	Write JSONIOPSDetail `json:"write"`
	Total float64        `json:"total"`
}

// JSONIOPSDetail represents detailed IOPS metrics
type JSONIOPSDetail struct {
	IOPS   float64 `json:"iops"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	StdDev float64 `json:"stddev"`
}

// JSONBandwidthStats represents bandwidth statistics
type JSONBandwidthStats struct {
	Read  JSONBandwidthDetail `json:"read"`
	Write JSONBandwidthDetail `json:"write"`
	Total float64             `json:"total_mbps"`
}

// JSONBandwidthDetail represents detailed bandwidth metrics
type JSONBandwidthDetail struct {
	BandwidthMBps float64 `json:"bandwidth_mbps"`
	Min           float64 `json:"min_mbps"`
	Max           float64 `json:"max_mbps"`
	Avg           float64 `json:"avg_mbps"`
}

// JSONLatencyStats represents latency statistics
type JSONLatencyStats struct {
	Read  JSONLatencyDetail `json:"read"`
	Write JSONLatencyDetail `json:"write"`
}

// JSONLatencyDetail represents detailed latency metrics
type JSONLatencyDetail struct {
	SubmissionLat JSONLatencyMetric `json:"submission_latency_us"`
	CompletionLat JSONLatencyMetric `json:"completion_latency_us"`
	TotalLat      JSONLatencyMetric `json:"total_latency_us"`
}

// JSONLatencyMetric represents latency metric values
type JSONLatencyMetric struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	StdDev float64 `json:"stddev"`
}

// JSONPercentiles represents latency percentiles
type JSONPercentiles struct {
	P1    float64 `json:"p1"`
	P5    float64 `json:"p5"`
	P10   float64 `json:"p10"`
	P20   float64 `json:"p20"`
	P30   float64 `json:"p30"`
	P40   float64 `json:"p40"`
	P50   float64 `json:"p50"`
	P60   float64 `json:"p60"`
	P70   float64 `json:"p70"`
	P80   float64 `json:"p80"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	P99_5 float64 `json:"p99_5"`
	P99_9 float64 `json:"p99_9"`
	P99_95 float64 `json:"p99_95"`
	P99_99 float64 `json:"p99_99"`
}

// JSONCPUUsage represents CPU usage statistics
type JSONCPUUsage struct {
	UserCPU        float64 `json:"user_cpu_percent"`
	SystemCPU      float64 `json:"system_cpu_percent"`
	ContextSwitches int64  `json:"context_switches"`
	MajorFaults    int64   `json:"major_faults"`
	MinorFaults    int64   `json:"minor_faults"`
}

// JSONDiskUtil represents disk utilization
type JSONDiskUtil struct {
	Device       string  `json:"device"`
	ReadIOs      int64   `json:"read_ios"`
	WriteIOs     int64   `json:"write_ios"`
	ReadSectors  int64   `json:"read_sectors"`
	WriteSectors int64   `json:"write_sectors"`
	Utilization  float64 `json:"utilization_percent"`
}

// JSONPerformanceHighlights represents the performance highlights
type JSONPerformanceHighlights struct {
	HighestIOPS      JSONHighlight `json:"highest_iops"`
	HighestBandwidth JSONHighlight `json:"highest_bandwidth"`
	LowestLatency    JSONHighlight `json:"lowest_latency"`
}

// JSONHighlight represents a single performance highlight
type JSONHighlight struct {
	TestName string  `json:"test_name"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
}

func saveResultsToJSON(results []TestResult, filename string) error {
	// Calculate summary statistics
	passed := 0
	failed := 0
	var totalDuration time.Duration

	for _, r := range results {
		if r.Status == "PASSED" {
			passed++
		} else {
			failed++
		}
		totalDuration += r.Duration
	}

	// Find performance highlights
	var maxIOPS, maxBW, minLatency TestResult
	maxIOPS.TotalIOPS = 0
	maxBW.TotalBWMBps = 0
	minLatency.AvgLatencyUs = 999999999

	for _, r := range results {
		if r.Status == "PASSED" {
			if r.TotalIOPS > maxIOPS.TotalIOPS {
				maxIOPS = r
			}
			if r.TotalBWMBps > maxBW.TotalBWMBps {
				maxBW = r
			}
			if r.AvgLatencyUs < minLatency.AvgLatencyUs && r.AvgLatencyUs > 0 {
				minLatency = r
			}
		}
	}

	// Build JSON structure
	jsonResults := JSONResults{
		Summary: JSONSummary{
			TotalTests:    len(results),
			Passed:        passed,
			Failed:        failed,
			TotalDuration: totalDuration.String(),
		},
		TestResults: make([]JSONTestResult, 0, len(results)),
		PerformanceHighlights: JSONPerformanceHighlights{
			HighestIOPS: JSONHighlight{
				TestName: maxIOPS.TestName,
				Value:    maxIOPS.TotalIOPS,
				Unit:     "iops",
			},
			HighestBandwidth: JSONHighlight{
				TestName: maxBW.TestName,
				Value:    maxBW.TotalBWMBps,
				Unit:     "MB/s",
			},
			LowestLatency: JSONHighlight{
				TestName: minLatency.TestName,
				Value:    minLatency.AvgLatencyUs,
				Unit:     "μs",
			},
		},
	}

	// Add test results
	for _, r := range results {
		testResult := JSONTestResult{
			TestName:      r.TestName,
			Description:   r.Description,
			Status:        r.Status,
			Duration:      r.Duration.Round(time.Second).String(),
			IOPS:          r.TotalIOPS,
			BandwidthMBps: r.TotalBWMBps,
			LatencyUs:     r.AvgLatencyUs,
		}

		// Populate IOPS stats
		if r.FioJob != nil {
			testResult.IOPSStats = JSONIOPSStats{
				Read: JSONIOPSDetail{
					IOPS:   r.ReadIOPS,
					Min:    r.FioJob.Read.IOPSMin,
					Max:    r.FioJob.Read.IOPSMax,
					Avg:    r.FioJob.Read.IOPSMean,
					StdDev: r.FioJob.Read.IOPSStddev,
				},
				Write: JSONIOPSDetail{
					IOPS:   r.WriteIOPS,
					Min:    r.FioJob.Write.IOPSMin,
					Max:    r.FioJob.Write.IOPSMax,
					Avg:    r.FioJob.Write.IOPSMean,
					StdDev: r.FioJob.Write.IOPSStddev,
				},
				Total: r.TotalIOPS,
			}

			// Populate Bandwidth stats
			testResult.BandwidthStats = JSONBandwidthStats{
				Read: JSONBandwidthDetail{
					BandwidthMBps: r.ReadBWMBps,
					Min:           r.FioJob.Read.BWMin / 1024,
					Max:           r.FioJob.Read.BWMax / 1024,
					Avg:           r.FioJob.Read.BWMean / 1024,
				},
				Write: JSONBandwidthDetail{
					BandwidthMBps: r.WriteBWMBps,
					Min:           r.FioJob.Write.BWMin / 1024,
					Max:           r.FioJob.Write.BWMax / 1024,
					Avg:           r.FioJob.Write.BWMean / 1024,
				},
				Total: r.TotalBWMBps,
			}

			// Populate Latency stats (convert from ns to us)
			testResult.LatencyStats = JSONLatencyStats{
				Read: JSONLatencyDetail{
					SubmissionLat: JSONLatencyMetric{
						Min:    r.FioJob.Read.Slat.Min / 1000,
						Max:    r.FioJob.Read.Slat.Max / 1000,
						Avg:    r.FioJob.Read.Slat.Mean / 1000,
						StdDev: r.FioJob.Read.Slat.Stddev / 1000,
					},
					CompletionLat: JSONLatencyMetric{
						Min:    r.FioJob.Read.Clat.Min / 1000,
						Max:    r.FioJob.Read.Clat.Max / 1000,
						Avg:    r.FioJob.Read.Clat.Mean / 1000,
						StdDev: r.FioJob.Read.Clat.Stddev / 1000,
					},
					TotalLat: JSONLatencyMetric{
						Min:    r.FioJob.Read.LatNs.Min / 1000,
						Max:    r.FioJob.Read.LatNs.Max / 1000,
						Avg:    r.FioJob.Read.LatNs.Mean / 1000,
						StdDev: r.FioJob.Read.LatNs.Stddev / 1000,
					},
				},
				Write: JSONLatencyDetail{
					SubmissionLat: JSONLatencyMetric{
						Min:    r.FioJob.Write.Slat.Min / 1000,
						Max:    r.FioJob.Write.Slat.Max / 1000,
						Avg:    r.FioJob.Write.Slat.Mean / 1000,
						StdDev: r.FioJob.Write.Slat.Stddev / 1000,
					},
					CompletionLat: JSONLatencyMetric{
						Min:    r.FioJob.Write.Clat.Min / 1000,
						Max:    r.FioJob.Write.Clat.Max / 1000,
						Avg:    r.FioJob.Write.Clat.Mean / 1000,
						StdDev: r.FioJob.Write.Clat.Stddev / 1000,
					},
					TotalLat: JSONLatencyMetric{
						Min:    r.FioJob.Write.LatNs.Min / 1000,
						Max:    r.FioJob.Write.LatNs.Max / 1000,
						Avg:    r.FioJob.Write.LatNs.Mean / 1000,
						StdDev: r.FioJob.Write.LatNs.Stddev / 1000,
					},
				},
			}

			// Populate percentiles (convert from ns to us)
			if len(r.FioJob.Read.Clat.Percentile) > 0 {
				testResult.Percentiles = JSONPercentiles{
					P1:     getPercentile(r.FioJob.Read.Clat.Percentile, "1.000000") / 1000,
					P5:     getPercentile(r.FioJob.Read.Clat.Percentile, "5.000000") / 1000,
					P10:    getPercentile(r.FioJob.Read.Clat.Percentile, "10.000000") / 1000,
					P20:    getPercentile(r.FioJob.Read.Clat.Percentile, "20.000000") / 1000,
					P30:    getPercentile(r.FioJob.Read.Clat.Percentile, "30.000000") / 1000,
					P40:    getPercentile(r.FioJob.Read.Clat.Percentile, "40.000000") / 1000,
					P50:    getPercentile(r.FioJob.Read.Clat.Percentile, "50.000000") / 1000,
					P60:    getPercentile(r.FioJob.Read.Clat.Percentile, "60.000000") / 1000,
					P70:    getPercentile(r.FioJob.Read.Clat.Percentile, "70.000000") / 1000,
					P80:    getPercentile(r.FioJob.Read.Clat.Percentile, "80.000000") / 1000,
					P90:    getPercentile(r.FioJob.Read.Clat.Percentile, "90.000000") / 1000,
					P95:    getPercentile(r.FioJob.Read.Clat.Percentile, "95.000000") / 1000,
					P99:    getPercentile(r.FioJob.Read.Clat.Percentile, "99.000000") / 1000,
					P99_5:  getPercentile(r.FioJob.Read.Clat.Percentile, "99.500000") / 1000,
					P99_9:  getPercentile(r.FioJob.Read.Clat.Percentile, "99.900000") / 1000,
					P99_95: getPercentile(r.FioJob.Read.Clat.Percentile, "99.950000") / 1000,
					P99_99: getPercentile(r.FioJob.Read.Clat.Percentile, "99.990000") / 1000,
				}
			}

			// Populate CPU usage
			testResult.CPUUsage = JSONCPUUsage{
				UserCPU:         r.FioJob.UsrCPU,
				SystemCPU:       r.FioJob.SysCPU,
				ContextSwitches: r.FioJob.Ctx,
				MajorFaults:     r.FioJob.MajF,
				MinorFaults:     r.FioJob.MinF,
			}
		}

		// Populate disk utilization
		if len(r.DiskUtil) > 0 {
			testResult.DiskUtil = make([]JSONDiskUtil, 0, len(r.DiskUtil))
			for _, disk := range r.DiskUtil {
				testResult.DiskUtil = append(testResult.DiskUtil, JSONDiskUtil{
					Device:       disk.Name,
					ReadIOs:      disk.ReadIOs,
					WriteIOs:     disk.WriteIOs,
					ReadSectors:  disk.ReadSectors,
					WriteSectors: disk.WriteSectors,
					Utilization:  disk.Util,
				})
			}
		}

		if r.Error != nil {
			testResult.Error = r.Error.Error()
		}

		jsonResults.TestResults = append(jsonResults.TestResults, testResult)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Write to file
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

func displaySummary(results []TestResult) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("=== OVERALL SUMMARY ===")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Summary statistics
	passed := 0
	failed := 0
	var totalDuration time.Duration

	for _, r := range results {
		if r.Status == "PASSED" {
			passed++
		} else {
			failed++
		}
		totalDuration += r.Duration
	}

	// Display statistics
	statsTable := tablewriter.NewWriter(os.Stdout)
	statsTable.SetHeader([]string{"Metric", "Value"})
	configureTable(statsTable, 2)
	statsTable.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	statsTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
	)
	statsTable.Append([]string{"Total Tests", strconv.Itoa(len(results))})
	statsTable.Append([]string{"Passed", strconv.Itoa(passed)})
	statsTable.Append([]string{"Failed", strconv.Itoa(failed)})
	statsTable.Append([]string{"Total Duration", totalDuration.String()})
	statsTable.Render()

	fmt.Println()

	// Detailed results table
	detailsTable := tablewriter.NewWriter(os.Stdout)
	detailsTable.SetHeader([]string{
		"Test Name",
		"Status",
		"IOPS",
		"BW (MB/s)",
		"Lat (μs)",
		"Duration",
	})
	// Configure manually instead of using configureTable to have different widths than Disk Utilization
	detailsTable.SetBorder(true)
	detailsTable.SetRowLine(true)
	detailsTable.SetAutoWrapText(false)
	detailsTable.SetColumnSeparator("│")
	detailsTable.SetCenterSeparator("┼")
	detailsTable.SetRowSeparator("─")
	detailsTable.SetColMinWidth(0, 40)
	detailsTable.SetColMinWidth(1, 11)
	detailsTable.SetColMinWidth(2, 11)
	detailsTable.SetColMinWidth(3, 11)
	detailsTable.SetColMinWidth(4, 11)
	detailsTable.SetColMinWidth(5, 10)
	detailsTable.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_CENTER, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT})
	detailsTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
	)

	for _, r := range results {
		status := "❌"
		if r.Status == "PASSED" {
			status = "✅"
		}

		iops := "-"
		bw := "-"
		lat := "-"

		if r.Status == "PASSED" {
			iops = fmt.Sprintf("%.0f", r.TotalIOPS)
			bw = fmt.Sprintf("%.2f", r.TotalBWMBps)
			lat = fmt.Sprintf("%.2f", r.AvgLatencyUs)
		}

		detailsTable.Append([]string{
			r.TestName,
			status,
			iops,
			bw,
			lat,
			r.Duration.Round(time.Second).String(),
		})
	}

	detailsTable.Render()

	// Performance summary
	fmt.Println()
	fmt.Println("=== Performance Highlights ===")

	var maxIOPS, maxBW, minLatency TestResult
	maxIOPS.TotalIOPS = 0
	maxBW.TotalBWMBps = 0
	minLatency.AvgLatencyUs = 999999999

	for _, r := range results {
		if r.Status == "PASSED" {
			if r.TotalIOPS > maxIOPS.TotalIOPS {
				maxIOPS = r
			}
			if r.TotalBWMBps > maxBW.TotalBWMBps {
				maxBW = r
			}
			if r.AvgLatencyUs < minLatency.AvgLatencyUs && r.AvgLatencyUs > 0 {
				minLatency = r
			}
		}
	}

	highlightsTable := tablewriter.NewWriter(os.Stdout)
	highlightsTable.SetHeader([]string{"Category", "Test", "Value"})
	configureTable(highlightsTable, 3)
	highlightsTable.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})
	highlightsTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgMagentaColor},
	)

	if maxIOPS.TotalIOPS > 0 {
		testName := maxIOPS.TestName
		if len(testName) > 32 {
			testName = testName[:32]
		}
		highlightsTable.Append([]string{
			"Highest IOPS",
			testName,
			fmt.Sprintf("%.0f", maxIOPS.TotalIOPS),
		})
	}

	if maxBW.TotalBWMBps > 0 {
		testName := maxBW.TestName
		if len(testName) > 32 {
			testName = testName[:32]
		}
		highlightsTable.Append([]string{
			"Highest Bandwidth",
			testName,
			fmt.Sprintf("%.2f MB/s", maxBW.TotalBWMBps),
		})
	}

	if minLatency.AvgLatencyUs < 999999999 {
		testName := minLatency.TestName
		if len(testName) > 32 {
			testName = testName[:32]
		}
		highlightsTable.Append([]string{
			"Lowest Latency",
			testName,
			fmt.Sprintf("%.2f μs", minLatency.AvgLatencyUs),
		})
	}

	highlightsTable.Render()

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Testing completed!")
	fmt.Println(strings.Repeat("=", 80))
}
