package demystifier

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func GetRunDataFromLog(logFile string) (*TestRunData, error) {
	var testRunData TestRunData

	var data []byte

	if strings.HasPrefix(logFile, "http://") || strings.HasPrefix(logFile, "https://") {
		log.WithFields(log.Fields{
			"log location": logFile,
		}).Debug("Using log from URL")
		resp, err := http.Get(logFile)
		if err != nil {
			return nil, fmt.Errorf("error opening URL: %v", err)
		}
		defer resp.Body.Close()

		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading HTTP response body: %v", err)
		}
	} else {
		log.WithFields(log.Fields{
			"log location": logFile,
		}).Debug("Using log from file")
		file, err := os.Open(logFile)
		if err != nil {
			return nil, fmt.Errorf("errosr opening file: %v", err)
		}
		defer file.Close()

		data, err = ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("error reading file: %v", err)
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(data)) // Create scanner from data

	var fullLogs strings.Builder
	for scanner.Scan() {
		fullLogs.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	testRunData.FullLogs = fullLogs.String()

	return &testRunData, nil
}

// GenerateLogURL generates a URL for the log file.
// This function may be replaced with your actual URL generation logic.
func GenerateLogURL(originalURL string) string {
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

// SetIndividualTestsFromLog processes the log data and updates the test run data accordingly.
// It sets individual test runs and their corresponding attempts based on the provided log data.
// If the provided testRunData is nil or if the logs are empty, it returns an error.
// The anchorTag parameter specifies the tag used to identify the start and end of individual tests.
//
// Parameters:
//   - testRunData: A pointer to TestRunData struct representing the test run data to be updated.
//   - anchorTag: A string indicating the tag used to identify the start and end of individual tests.
//
// Returns:
//   - An error if any issue occurs during processing, or nil if the processing is successful.
func SetIndividualTestsFromLog(testRunData *TestRunData, anchorTag string) error {
	if testRunData == nil {
		return errors.New("testRunData is nil")
	}

	if testRunData.FullLogs == "" {
		return errors.New("Logs were not provided")
	}

	attempts := make(map[string]int)

	lines := strings.Split(testRunData.FullLogs, "\n")
	startRegex := regexp.MustCompile(fmt.Sprintf(`> Enter \[%s\] (.+) - (.+) @ (.+)`, anchorTag))
	endRegex := regexp.MustCompile(fmt.Sprintf(`< Exit \[%s\] (.+?) - .+ @ (.+) \(.+\)`, anchorTag))
	failureRegex := regexp.MustCompile(`^[\t ]*\[FAILED\].*`)

	var currentAttempt *AttemptData
	var currentTestRun *IndividualTestRunData

	for _, line := range lines {
		if matches := startRegex.FindStringSubmatch(line); matches != nil {
			handleStartTag(line, matches, &currentTestRun, &currentAttempt, attempts, &testRunData)
		} else if matches := endRegex.FindStringSubmatch(line); matches != nil {
			handleEndTag(line, matches, &currentAttempt)
		} else if matches := failureRegex.FindStringSubmatch(line); matches != nil {
			log.WithFields(log.Fields{
				"Line":       currentAttempt.Name,
				"Attempt no": currentAttempt.AttemptNo,
			}).Debug("Marking attempt FAILED")
			currentAttempt.Status = EventStatus{Status: Failed}
		} else if currentAttempt != nil {
			handleLogs(line, currentAttempt)
		}
	}

	return nil
}

func handleStartTag(line string, matches []string, currentTestRunPtr **IndividualTestRunData, currentAttemptPtr **AttemptData, attempts map[string]int, testRunsPtr **TestRunData) {
	eventName := matches[2]
	shortEventName := matches[1]
	log.WithFields(log.Fields{
		"Line":       line,
		"Attempt no": attempts[eventName],
	}).Debug("Found new Attempt")

	*currentTestRunPtr = getOrAddTestRun(testRunsPtr, eventName, shortEventName)

	// Create a new instance of AttemptData
	newAttempt := &AttemptData{
		AttemptNo: attempts[eventName],
		Name:      eventName,
	}

	*currentAttemptPtr = newAttempt
	attempts[eventName]++

	(*currentTestRunPtr).Attempt = append((*currentTestRunPtr).Attempt, newAttempt)

	// Add logs to the new attempt
	newAttempt.Logs = append(newAttempt.Logs, line)

	// Parse and set the start time for the new attempt
	parsedTime, err := parseGingkoTime(matches[3])
	if err != nil {
		log.Error("Error parsing time:", err)
		return
	}
	newAttempt.StartTime = parsedTime

	log.WithFields(log.Fields{
		"Test":         (*currentTestRunPtr).ShortName,
		"Attempt no":   newAttempt.AttemptNo,
		"Attempt name": newAttempt.Name,
		"Start Time":   newAttempt.StartTime,
	}).Debug("Created New Attempt")
}

func getOrAddTestRun(testRunsPtr **TestRunData, eventName string, shortEventName string) *IndividualTestRunData {
	// Ensure that the TestRun slice is initialized
	if (*testRunsPtr).TestRun == nil {
		(*testRunsPtr).TestRun = []*IndividualTestRunData{}
	}

	// Iterate through existing IndividualTestRunData instances
	for _, testRun := range (*testRunsPtr).TestRun {
		if testRun.Name == eventName {
			// If an IndividualTestRunData with the same eventName exists, return a pointer to it
			return testRun
		}
	}

	// If no matching IndividualTestRunData was found, create a new one
	newTestRun := &IndividualTestRunData{Name: eventName, ShortName: shortEventName}
	// Append it to the TestRun slice
	(*testRunsPtr).TestRun = append((*testRunsPtr).TestRun, newTestRun)
	// Return a pointer to the newly created IndividualTestRunData
	return newTestRun
}

func handleEndTag(line string, matches []string, currentAttemptPtr **AttemptData) {
	currentAttempt := *currentAttemptPtr // Dereference the pointer to get the AttemptData

	log.WithFields(log.Fields{
		"Line":       line,
		"Attempt no": currentAttempt.AttemptNo,
	}).Debug("Found end Attempt")

	if currentAttempt != nil {
		endTime, err := parseGingkoTime(matches[2])
		if err != nil {
			log.Error("Error parsing end time:", err)
			return
		}
		currentAttempt.EndTime = endTime
		currentAttempt.Duration = endTime.Sub(currentAttempt.StartTime)
		log.WithFields(log.Fields{
			"StartTime": currentAttempt.StartTime,
			"EndTime":   currentAttempt.EndTime,
			"Duration":  currentAttempt.Duration,
		}).Debug("Attempt times")
		currentAttempt.Logs = append(currentAttempt.Logs, line)
	}
}

func handleLogs(line string, currentAttempt *AttemptData) {
	currentAttempt.Logs = append(currentAttempt.Logs, line)
}

// parseDuration parses a duration string like "16.65s" into a time.Duration.
func parseDuration(durationStr string) (time.Duration, error) {
	return time.ParseDuration(durationStr)
}

func parseGingkoTime(timeStr string) (time.Time, error) {
	formats := []string{
		"01/02/06 15:04:05.000",
		"01/02/06 15:04:05.00",
		"01/02/06 15:04:05.0",
		"01/02/06 15:04:05",
	}
	var parsedTime time.Time
	var err error
	for _, format := range formats {
		parsedTime, err = time.Parse(format, timeStr)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}
