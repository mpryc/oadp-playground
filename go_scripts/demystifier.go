package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"test_demystifier/demystifier"
	"time"

	log "github.com/sirupsen/logrus"
)

func parseLogFile(logFile string) (*demystifier.TestRunData, error) {
	testRunDataPtr, err := demystifier.GetRunDataFromLog(logFile)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error")
	}

	err = demystifier.SetIndividualTestsFromLog(testRunDataPtr, "It")

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Error")
	}

	return testRunDataPtr, nil
}

func DumpTestsToFolder(testData *demystifier.TestRunData, folder string) {
	mkdirErr := os.MkdirAll(folder, 0755)
	if mkdirErr != nil {
		log.WithFields(log.Fields{
			"error": mkdirErr,
		}).Fatal("Error")
	}
	for i := range testData.TestRun {
		thisRun := &testData.TestRun[i]
		for j := range thisRun.Attempt {
			thisAttempt := &thisRun.Attempt[j]
			err := thisAttempt.DumpLogsToFileWithPrefixes(folder, thisAttempt.Name, ": ")
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Fatal("Error")
			}
		}
	}
}

func PrintTestSummary(testData *demystifier.TestRunData) {
	// Define a struct to hold the summary data
	type TestSummary struct {
		Name           string
		NumAttempts    int
		NumFailed      int
		TotalRunTime   time.Duration
		NumOver1Second int
		AverageRunTime time.Duration
	}

	// Initialize a slice to hold the summary data for each test run
	var summaries []TestSummary

	// Loop through each test run to collect summary data
	for i := range testData.TestRun {
		var numAttempts, failedAttempts, numOver1Second int
		totalRunTime := time.Duration(0)
		thisTest := &testData.TestRun[i]
		for j := range thisTest.Attempt {
			// Increment the number of attempts
			numAttempts++
			thisAttempt := &thisTest.Attempt[j]
			// If the attempt failed, increment the failed attempts counter
			if thisAttempt.Status.Status == "FAILED" {
				failedAttempts++
			}

			// If the duration is greater than 1 second, increment the counter
			if thisAttempt.Duration > time.Second {
				numOver1Second++
			}

			// Add the duration to the total run time
			totalRunTime += thisAttempt.Duration
		}

		// Calculate the average run time based on durations over 1 second
		var averageRunTime time.Duration
		if numOver1Second > 0 {
			averageRunTime = totalRunTime / time.Duration(numOver1Second)
		}

		// Append the summary data to the slice
		summaries = append(summaries, TestSummary{
			Name:           thisTest.ShortName,
			NumAttempts:    numAttempts,
			NumFailed:      failedAttempts,
			TotalRunTime:   totalRunTime,
			NumOver1Second: numOver1Second,
			AverageRunTime: averageRunTime,
		})
	}

	// Sort by avg time
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].AverageRunTime < summaries[j].AverageRunTime
	})

	// Print the summary table
	fmt.Println("Test Summary Table:")
	headerStr := "---------------------------------------------------------------------------------------------------"
	fmt.Println(headerStr)
	fmt.Printf("| %-40s | %-15s | %-11s | %-20s |\n", "Test Name", "Num Attempts", "Num Failed", "Average Run Time")
	fmt.Println(headerStr)
	for _, summary := range summaries {
		fmt.Printf("| %-40s | %-15d | %-11d | %-20s |\n", summary.Name, summary.NumAttempts, summary.NumFailed, summary.AverageRunTime)
	}
	fmt.Println(headerStr)
}

func main() {
	log.SetLevel(log.InfoLevel)

	log.WithFields(log.Fields{
		">>> start_demystifier_timestamp": time.Now().Unix(),
	}).Info("Test Demystifier starts its journey")

	var (
		logLocation string
		showPassing bool
		timeStamps  bool
		debugMode   bool
		dumpLogsToFolder string
	)

	flag.BoolVar(&timeStamps, "t", false, "whether to include timestamps in the output (shorthand)")
	flag.BoolVar(&showPassing, "s", false, "show all tests even those passing")
	flag.BoolVar(&debugMode, "d", false, "debug mode")
	flag.StringVar(&dumpLogsToFolder, "f", "", "dump logs to folder")

	flag.Parse()

	if debugMode {
		log.SetLevel(log.DebugLevel)
	}

	if len(flag.Args()) > 0 {
		logLocation = demystifier.GenerateLogURL(flag.Arg(0))
	}

	log.WithFields(log.Fields{
		">>> location": logLocation,
	}).Info("Using log from")

	testData, _ := parseLogFile(logLocation)

	for i := range testData.TestRun {
		failedAttempts := 0 // Initialize counter for failed attempts in this test run
		thisTest := &testData.TestRun[i]
		for j := range thisTest.Attempt {
			thisAttempt := &thisTest.Attempt[j]
			fields := log.Fields{
				"Name": thisTest.ShortName,
				"No":   thisAttempt.AttemptNo,
				"Time": thisAttempt.Duration,
			}

			// If the attempt failed or showPassing is true, log the attempt
			if thisAttempt.Status.Status == demystifier.Failed {
				log.WithFields(fields).Error("Failed attempt run")
				// Increment the counter if the attempt failed
				if thisAttempt.Status.Status == demystifier.Failed {
					failedAttempts++
				}
			} else if showPassing {
				log.WithFields(fields).Info("Pass attempt run")
			}
		}

		// Summary for this test run
		if failedAttempts > 0 {
			log.WithFields(log.Fields{
				"Name":   thisTest.Name,
				"Failed": failedAttempts,
			}).Info("Test Summary")
		}
	}
	if dumpLogsToFolder != "" {
		DumpTestsToFolder(testData, dumpLogsToFolder)
		os.Exit(0)
	}
	PrintTestSummary(testData)

	log.WithFields(log.Fields{
		">>> end_demystifier_timestamp": time.Now().Unix(),
	}).Info("Test Demystifier finishes its journey")
}
