//go:build linux

package battery

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestGetUPSStats_Apcaccess(t *testing.T) {
	oldCommand := execCommand
	oldLookPath := execLookPath
	defer func() {
		execCommand = oldCommand
		execLookPath = oldLookPath
	}()

	execLookPath = func(file string) (string, error) {
		if file == "apcaccess" {
			return "/usr/sbin/apcaccess", nil
		}
		return "", errors.New("not found")
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	percent, state, err := GetUPSStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if percent != 95 {
		t.Errorf("expected percent 95, got %d", percent)
	}
	if state != stateCharging {
		t.Errorf("expected state %d (stateCharging), got %d", stateCharging, state)
	}
}

func TestGetUPSStats_Upsc(t *testing.T) {
	oldCommand := execCommand
	oldLookPath := execLookPath
	defer func() {
		execCommand = oldCommand
		execLookPath = oldLookPath
	}()

	execLookPath = func(file string) (string, error) {
		if file == "upsc" {
			return "/usr/bin/upsc", nil
		}
		return "", errors.New("not found")
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	percent, state, err := GetUPSStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if percent != 80 {
		t.Errorf("expected percent 80, got %d", percent)
	}
	if state != stateDischarging {
		t.Errorf("expected state %d (stateDischarging), got %d", stateDischarging, state)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 2 {
		os.Exit(2)
	}
	cmd := args[1]
	if cmd == "/usr/sbin/apcaccess" {
		os.Stdout.WriteString("BCHARGE  : 95.0 Percent\nSTATUS   : ONLINE CHARGING\n")
		os.Exit(0)
	} else if cmd == "/usr/bin/upsc" {
		// First call with "-l" to list UPS
		if len(args) > 2 && args[2] == "-l" {
			os.Stdout.WriteString("myups\n")
			os.Exit(0)
		}
		// Second call with "myups"
		if len(args) > 2 && args[2] == "myups" {
			os.Stdout.WriteString("battery.charge: 80\nups.status: OB\n")
			os.Exit(0)
		}
	}
	os.Exit(1)
}
