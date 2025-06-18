package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
        "nextflow-go/pkg/utils"
)

var refPattern = regexp.MustCompile(`\$\{k8s\.([a-zA-Z0-9_]+)\}`)

func NormalizeK8sConfig(config map[string]string) (map[string]string, error) {
        normalized := make(map[string]string)

	for key, value := range config {
		matches := refPattern.FindAllStringSubmatch(value, -1)
		newValue := value

		for _, match := range matches {
			refKey := match[1]

			// Check for self-reference
			if refKey == key {
				return nil, fmt.Errorf("self-reference detected in key: %s", key)
			}

			refValue, ok := config[refKey]
			if !ok {
				return nil, fmt.Errorf("reference to undefined key: %s", refKey)
			}
                        refValue = utils.Stripped(refValue)

			// Replace all occurrences of the reference
			newValue = strings.ReplaceAll(newValue, match[0], refValue)
		}

		normalized[key] = newValue
	}
        
	return normalized, nil
}

func ReadNextflowConfig(filename string) (map[string]string, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var (
		insideK8s        bool
		k8sBlock         []string
		remainingLines   []string
		nestingLevel     int
		accumulator      string
		k8sStartRegex    = regexp.MustCompile(`^\s*k8s\s*\{.*$`)
	)

	for scanner.Scan() {
		line := accumulator + scanner.Text()
                inString := false
                var quote rune
                for i := 0; i < len(line); i++ {
                        if !inString && strings.HasPrefix(line[i:], "//") {
                                line = line[:i]
                                break
                        }
                        if (line[i] == '\'' || line[i] == '"') && (i == 0 || line[i-1] != '\\') {
                                if !inString {
                                        inString, quote = true, rune(line[i])
                                } else if rune(line[i]) == quote {
                                        inString = false
                                }
                        }
                }
		if !insideK8s {
			if k8sStartRegex.MatchString(line) {
				insideK8s = true
				nestingLevel = 1
				idx := strings.Index(line, "{")
				k8sBlock = append(k8sBlock, line[idx+1:])
				accumulator = ""
			} else {
				accumulator += line
			}
			continue
		} 
		nestingLevel += strings.Count(line, "{") - strings.Count(line, "}")
		k8sBlock = append(k8sBlock, line)
		if nestingLevel == 0 {
			break
		}
	}
	for scanner.Scan() {
		remainingLines = append(remainingLines, scanner.Text())
	}
	if !insideK8s {
		return nil, "", fmt.Errorf("k8s block not found in config file %s", filename)
	}
	return parseK8sBlock(k8sBlock), strings.Join(remainingLines, "\n"), nil
}

func parseK8sBlock(lines []string) map[string]string {
	config := make(map[string]string)
	var (
		key string
		val strings.Builder
		multiLine bool
		delimiter rune
		nesting   int
	)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !multiLine {
			parts := splitLine(line)
			if len(parts) != 2 {
				continue
			}
			key = strings.TrimSpace(parts[0])
			val.Reset()
			val.WriteString(strings.TrimSpace(parts[1]))
			if isMultilineStart(parts[1]) {
				multiLine = true
				delimiter, nesting = getDelimiterAndNesting(parts[1])
				continue
			}
			config[key] = val.String()
		} else {
			val.WriteString(" " + line)
			multiLine, nesting = handleMultiline(line, delimiter, nesting, &val)
			if !multiLine {
				config[key] = strings.TrimSpace(val.String())
			}
		}
	}
	return config
}

func splitLine(line string) []string {
	re := regexp.MustCompile(`(?m)(?P<key>[^=]+)\s*=\s*(?P<value>.*)`)
	match := re.FindStringSubmatch(line)
	if len(match) == 3 {
		return match[1:]
	}
	return []string{}
}

func isMultilineStart(value string) bool {
	return strings.Count(value, "[") != strings.Count(value, "]")
}

func getDelimiterAndNesting(value string) (rune, int) {
	if strings.HasPrefix(strings.TrimSpace(value), "[") {
		return ']', strings.Count(value, "[") - strings.Count(value, "]")
	}
	return 0, 0
}

func handleMultiline(line string, delimiter rune, nesting int, builder *strings.Builder) (bool, int) {
	for _, r := range line {
		switch r {
		case delimiter:
			nesting--
			if nesting == 0 {
				return false, nesting
			}
		case '[':
			nesting++
		}
	}
	return true, nesting
}

