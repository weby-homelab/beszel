//go:build linux

package battery

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

var execCommand = exec.Command
var execLookPath = exec.LookPath

// GetUPSStats returns battery percent and charge state from apcaccess or upsc.
func GetUPSStats() (percent uint8, state uint8, err error) {
	if percent, state, err = getApcaccessStats(); err == nil {
		return percent, state, nil
	}
	if percent, state, err = getUpscStats(); err == nil {
		return percent, state, nil
	}
	return 0, 0, errors.New("no UPS found or unable to read stats")
}

func getApcaccessStats() (uint8, uint8, error) {
	path, err := execLookPath("apcaccess")
	if err != nil {
		return 0, 0, err
	}
	cmd := execCommand(path, "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, 0, err
	}

	var percent float64 = -1
	status := ""

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "BCHARGE":
			valParts := strings.Fields(val)
			if len(valParts) > 0 {
				if pct, err := strconv.ParseFloat(valParts[0], 64); err == nil {
					percent = pct
				}
			}
		case "STATUS":
			status = strings.ToUpper(val)
		}
	}

	if percent < 0 || status == "" {
		return 0, 0, errors.New("invalid apcaccess output")
	}

	var batState uint8 = stateUnknown
	if strings.Contains(status, "ONBATT") {
		batState = stateDischarging
	} else if strings.Contains(status, "CHARGING") {
		batState = stateCharging
	} else if strings.Contains(status, "ONLINE") {
		if percent >= 99 {
			batState = stateFull
		} else {
			batState = stateCharging
		}
	} else {
		batState = stateIdle
	}

	return uint8(percent), batState, nil
}

func getUpscStats() (uint8, uint8, error) {
	path, err := execLookPath("upsc")
	if err != nil {
		return 0, 0, err
	}

	listCmd := execCommand(path, "-l")
	var listOut bytes.Buffer
	listCmd.Stdout = &listOut
	if err := listCmd.Run(); err != nil {
		return 0, 0, err
	}

	upsName := ""
	scanner := bufio.NewScanner(&listOut)
	if scanner.Scan() {
		upsName = strings.TrimSpace(scanner.Text())
	}
	if upsName == "" {
		return 0, 0, errors.New("no UPS defined in upsc")
	}

	cmd := execCommand(path, upsName)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, 0, err
	}

	var percent int = -1
	status := ""

	scanner = bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "battery.charge":
			if pct, err := strconv.Atoi(val); err == nil {
				percent = pct
			}
		case "ups.status":
			status = strings.ToUpper(val)
		}
	}

	if percent < 0 || status == "" {
		return 0, 0, errors.New("invalid upsc output")
	}

	var batState uint8 = stateUnknown
	if strings.Contains(status, "OB") {
		batState = stateDischarging
	} else if strings.Contains(status, "CHG") {
		batState = stateCharging
	} else if strings.Contains(status, "OL") {
		if percent >= 99 {
			batState = stateFull
		} else {
			batState = stateCharging
		}
	} else {
		batState = stateIdle
	}

	return uint8(percent), batState, nil
}
