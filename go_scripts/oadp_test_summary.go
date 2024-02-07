package main

import (
	"bufio"
	"fmt"
	"net/http"
	"flag"
	"os"
	"strings"
	"regexp"
	"time"
)

type TestData struct {
	StartTime        time.Time
	EndTime          time.Time
	BackupStartTime  time.Time
	BackupEndTime    time.Time
	RestoreStartTime time.Time
	RestoreEndTime   time.Time
}

func generateLogURL(originalURL string) string {
	// Check if the original URL already points to build-log.txt
	if strings.HasSuffix(originalURL, "/build-log.txt") {
		return originalURL
	}
	parts := strings.Split(originalURL, "https://prow.ci.openshift.org/view/gs/")

	re := regexp.MustCompile(`e2e-test-(.*?)/`)
	testType := re.FindString(originalURL)
	logURL := fmt.Sprintf("https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/%s/artifacts/%se2e/build-log.txt", parts[1], testType)
	return logURL
}

func parseLogFile(logFile string) map[string]map[int]*TestData {
	testData := make(map[string]map[int]*TestData)
	var scanner *bufio.Scanner

	if strings.HasPrefix(logFile, "http://") || strings.HasPrefix(logFile, "https://") {
	  logFileURL := generateLogURL(logFile)
    fmt.Println("Using log file", logFileURL)
		resp, err := http.Get(logFileURL)
		if err != nil {
			fmt.Println("Error opening URL:", err)
			return testData
		}
		defer resp.Body.Close()

		scanner = bufio.NewScanner(resp.Body)
	} else {
		file, err := os.Open(logFile)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return testData
		}
		defer file.Close()

		scanner = bufio.NewScanner(file)
	}
	var currentTestName string
	currentRetryCounter := make(map[string]int)

	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Parse start of test
		enterMatch := regexp.MustCompile(`.*Enter \[It\] (.+?) - (.+?) @`).FindStringSubmatch(line)
		if len(enterMatch) > 0 {
			currentRetryCounter[currentTestName] += 1
			currentTestName = fmt.Sprintf("%s - %s", enterMatch[1], enterMatch[2])
			if _, ok := testData[currentTestName]; !ok {
				testData[currentTestName] = make(map[int]*TestData)
			}
			testData[currentTestName][currentRetryCounter[currentTestName]] = &TestData{}
			timestamp := extractTimeFromEnterIt(line)
			testData[currentTestName][currentRetryCounter[currentTestName]].StartTime = timestamp
		}

		// Parse end of test
		exitMatch := regexp.MustCompile(`.*Exit \[It\] (.+?) - (.+?) @`).FindStringSubmatch(line)
		if len(exitMatch) > 0 && currentTestName != "" {
			endTime := extractTimeFromEnterIt(line)
			testData[currentTestName][currentRetryCounter[currentTestName]].EndTime = endTime
		}

		// Parse timestamps for other events
		if currentTestName != "" {
			parseTimestamps(line, testData[currentTestName][currentRetryCounter[currentTestName]])
		}
	}
	
	if lineCount < 100 {
	  fmt.Printf("\nWARNING: The log file is only %d lines long, probably something wrong here with the infra\n\n", lineCount)
  }

	if err := scanner.Err(); err != nil { 
		fmt.Println("Error reading file:", err)
	}

	return testData
}

func extractTimeFromEnterIt(line string) time.Time {
    parts := strings.Split(line, " @ ")
    if len(parts) != 2 {
				fmt.Println("Error invalid format: %s", line)    
        return time.Time{}
    }
    re := regexp.MustCompile(`(\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})(?:\.\d{1,3})?(?:\s+\([\d.]+s\))?`)
    timeStr := strings.TrimSpace(parts[1])
    matches := re.FindStringSubmatch(timeStr)
    if len(matches) != 2 {
				fmt.Println("Error invalid format: %s", line)    
        return time.Time{}
    }
    timeStr = matches[1]

    layout := "01/02/06 15:04:05.999"
    t, err := time.Parse(layout, timeStr)
    if err != nil {
				fmt.Println("Error invalid format: %v", err)        
        return time.Time{}
    }
    return t
}

func parseTimestamp(line string) time.Time {
	// Define regular expressions for different timestamp formats
	regexFormats := []string{
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d{3}`,
		`\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d{3}`,
		`\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
	}

	// Iterate over regular expressions and attempt to match each one
	for _, format := range regexFormats {
		re := regexp.MustCompile(format)
		match := re.FindString(line)
		if match != "" {
			layout := "06/01/02 15:04:05"
			if len(match) > 19 {
				layout += ".000"
			}
			timestamp, err := time.Parse(layout, match)
			if err != nil {
				fmt.Println("Error parsing timestamp:", err)
				return time.Time{}
			}
			return timestamp
		}
	}

	fmt.Println("No matching timestamp format found in line:", line)
	return time.Time{}
}


func parseTimestamps(line string, data *TestData) {
  if backupStartMatch := regexp.MustCompile(`(\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Creating backup`).FindStringSubmatch(line); len(backupStartMatch) > 0 {
		data.BackupStartTime = parseTimestamp(line)
	} else if backupEndMatch := regexp.MustCompile(`(\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Backup for case (.+) succeeded`).FindStringSubmatch(line); len(backupEndMatch) > 0 {
		data.BackupEndTime = parseTimestamp(line)
	} else if restoreStartMatch := regexp.MustCompile(`(\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Creating restore`).FindStringSubmatch(line); len(restoreStartMatch) > 0 {
		data.RestoreStartTime = parseTimestamp(line)
	} else if restoreEndMatch := regexp.MustCompile(`(\d{2}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}).*Post backup and restore state:  passed`).FindStringSubmatch(line); len(restoreEndMatch) > 0 {
		data.RestoreEndTime = parseTimestamp(line)
	}
}

func printTestData(testData map[string]map[int]*TestData, timestamps bool) {
	for testName, attempts := range testData {
		fmt.Printf("\n> %s\n", testName)
		for attempt, data := range attempts {
			fmt.Printf("\tAttempt %d:\n", attempt)
			if timestamps {
				fmt.Printf("\t\tStart_Time: %s\n", data.StartTime)
				fmt.Printf("\t\tEnd_Time: %s\n", data.EndTime)
			}
			fmt.Printf("\t\tAttempt_Time: %s\n", data.EndTime.Sub(data.StartTime))
			if !data.BackupStartTime.IsZero() {
				if timestamps {
					fmt.Printf("\t\tBackup_Start_Time: %s\n", data.BackupStartTime)
				}
				if !data.BackupEndTime.IsZero() {
					if timestamps {
						fmt.Printf("\t\tBackup_End_Time: %s\n", data.BackupEndTime)
					}
					fmt.Printf("\t\tTotal_Backup_Time: %s\n", data.BackupEndTime.Sub(data.BackupStartTime))
				} else {
						fmt.Printf("\t\tBackup_End_Time End Time: %s\n", "FAILURE")
				}
			}
			if !data.RestoreStartTime.IsZero() {
				if timestamps {
					fmt.Printf("\t\tRestore Start Time: %s\n", data.RestoreStartTime)
				}
				if !data.RestoreEndTime.IsZero() {
					if timestamps {
						fmt.Printf("\t\tRestore End Time: %s\n", data.RestoreEndTime)
					}
					fmt.Printf("\t\tTotal_Restore_Time: %s\n", data.RestoreEndTime.Sub(data.RestoreStartTime))
				} else {
						fmt.Printf("\t\tRestore End Time: %s\n", "FAILURE")
				}
			}
		}
	}
}


func main() {
	timestamps := flag.Bool("timestamps", false, "whether to include timestamps in the output")
	flag.BoolVar(timestamps, "t", false, "whether to include timestamps in the output (shorthand)")

	flag.Parse()

	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Println("Usage: go run oadp_test_summary.go <path_or_url_to_log_file> [--timestamps]")
		os.Exit(1)
	}

	if len(os.Args) == 3 && (os.Args[2] == "-t" || os.Args[2] == "--timestamps") {
		*timestamps = true
	}

	logFilePath := os.Args[1]
	testData := parseLogFile(logFilePath)

	printTestData(testData, *timestamps)
}
