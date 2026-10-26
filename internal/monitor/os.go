package monitor

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func parseOSRelease(reader io.Reader) (string, error) {
	values := map[string]string{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return "", fmt.Errorf("invalid os-release assignment %q", line)
		}
		if key != "PRETTY_NAME" && key != "NAME" && key != "VERSION_ID" {
			continue
		}
		parsed, err := parseReleaseValue(value)
		if err != nil {
			return "", err
		}
		values[key] = parsed
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if values["PRETTY_NAME"] != "" {
		return values["PRETTY_NAME"], nil
	}
	name := values["NAME"]
	if name == "" {
		name = "Linux"
	}
	return strings.TrimSpace(name + " " + values["VERSION_ID"]), nil
}

func parseReleaseValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	quote := byte(0)
	if value[0] == '\'' || value[0] == '"' {
		quote = value[0]
		if len(value) < 2 || value[len(value)-1] != quote {
			return "", fmt.Errorf("unclosed os-release value %q", value)
		}
		value = value[1 : len(value)-1]
	}
	if quote == '\'' {
		return value, nil
	}
	var result strings.Builder
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == '\\' {
			if index+1 >= len(value) {
				return "", fmt.Errorf("unfinished os-release escape %q", value)
			}
			next := value[index+1]
			if quote == 0 || strings.ContainsRune("$`\"\\", rune(next)) {
				result.WriteByte(next)
				index++
				continue
			}
		}
		result.WriteByte(character)
	}
	return result.String(), nil
}
