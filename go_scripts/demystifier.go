package main

import (
	"flag"
	"fmt"
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

		for j := range testData.TestRun[i].Attempt {
			// Increment the number of attempts
			numAttempts++

			// If the attempt failed, increment the failed attempts counter
			if testData.TestRun[i].Attempt[j].Status.Status == "FAILED" {
				failedAttempts++
			}

			// If the duration is greater than 1 second, increment the counter
			if testData.TestRun[i].Attempt[j].Duration > time.Second {
				numOver1Second++
			}

			// Add the duration to the total run time
			totalRunTime += testData.TestRun[i].Attempt[j].Duration
		}

		// Calculate the average run time based on durations over 1 second
		var averageRunTime time.Duration
		if numOver1Second > 0 {
			averageRunTime = totalRunTime / time.Duration(numOver1Second)
		}

		// Append the summary data to the slice
		summaries = append(summaries, TestSummary{
			Name:           testData.TestRun[i].ShortName,
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
	)

	flag.BoolVar(&timeStamps, "t", false, "whether to include timestamps in the output (shorthand)")
	flag.BoolVar(&showPassing, "s", false, "show all tests even those passing")
	flag.BoolVar(&debugMode, "d", false, "debug mode")

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
		for j := range testData.TestRun[i].Attempt {
			fields := log.Fields{
				"Name": testData.TestRun[i].ShortName,
				"No":   testData.TestRun[i].Attempt[j].AttemptNo,
				"Time": testData.TestRun[i].Attempt[j].Duration,
			}

			// If the attempt failed or showPassing is true, log the attempt
			if testData.TestRun[i].Attempt[j].Status.Status == demystifier.Failed {
				log.WithFields(fields).Error("Failed attempt run")
				// Increment the counter if the attempt failed
				if testData.TestRun[i].Attempt[j].Status.Status == demystifier.Failed {
					failedAttempts++
				}
			} else if showPassing {
				log.WithFields(fields).Info("Pass attempt run")
			}
		}

		// Summary for this test run
		if failedAttempts > 0 {
			log.WithFields(log.Fields{
				"Name":   testData.TestRun[i].Name,
				"Failed": failedAttempts,
			}).Info("Test Summary")
		}
	}

	PrintTestSummary(testData)

	log.WithFields(log.Fields{
		">>> end_demystifier_timestamp": time.Now().Unix(),
	}).Info("Test Demystifier finishes its journey")
}
