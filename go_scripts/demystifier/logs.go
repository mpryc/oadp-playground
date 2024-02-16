package demystifier

import (
	"os"
	"strings"
)

// func (a *AttemptData) logsPrefixedWithTestName() []string {
// 	var logsWithPrefix []string
// 	for _, log := range a.Logs {
// 		logsWithPrefix = append(logsWithPrefix, a.Name+": "+log)
// 	}
// 	return logsWithPrefix
// }

func (a *AttemptData) DumpLogsToFileWithPrefixes(folder string, prefixes ...string) error {
	// replace / in name
	fileName := strings.ReplaceAll(a.Name, "/", "_")
	file, err := os.Create(folder + "/" + fileName + ".log")
	if err != nil {
		return err
	}
	defer file.Close()
	for i := range a.Logs {
		for j := range prefixes {
			if _, err := file.WriteString(prefixes[j]); err != nil {
				return err
			}
		}
		if _, err := file.WriteString(a.Logs[i] + "\n"); err != nil {
			return err
		}
	}
	return nil
}