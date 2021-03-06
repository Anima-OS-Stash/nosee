package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type tomlAlert struct {
	Name      string
	Disabled  bool
	Targets   []string
	Command   string
	Arguments []string
	Hours     []string
	Days      []int
}

func alertCheckHour(hour string) ([2]int, error) {
	var err error
	var res [2]int

	parts := strings.Split(hour, ":")
	if len(parts) != 2 {
		return res, fmt.Errorf("invalid format '%s' (ex: '19:30')", hour)
	}
	res[0], err = strconv.Atoi(parts[0])
	if err != nil {
		return res, fmt.Errorf("can't convert '%s' hour to integer: %s", hour, err)
	}
	res[1], err = strconv.Atoi(parts[1])
	if err != nil {
		return res, fmt.Errorf("can't convert '%s' minute to integer: %s", hour, err)
	}

	if res[0] < 0 {
		return res, fmt.Errorf("hour can't be less than 0: %s", hour)
	}
	if res[1] < 0 {
		return res, fmt.Errorf("minute can't be less than 0: %s", hour)
	}
	if res[0] > 23 {
		return res, fmt.Errorf("hour can't more than 23: %s", hour)
	}
	if res[1] > 59 {
		return res, fmt.Errorf("minute can't more than 59: %s", hour)
	}

	return res, nil
}

func alertCheckHours(hours []string) ([]HourRange, error) {
	var hourRanges []HourRange

	for _, hour := range hours {
		var (
			hourRange HourRange
			err       error
		)

		rng := strings.Split(hour, "-")
		if len(rng) != 2 {
			return nil, fmt.Errorf("invalid format '%s' (ex: '8:90 - 19:00')", hour)
		}
		rng[0] = strings.TrimSpace(rng[0])
		rng[1] = strings.TrimSpace(rng[1])

		if hourRange.Start, err = alertCheckHour(rng[0]); err != nil {
			return nil, fmt.Errorf("invalid start hour: %s", err)
		}
		if hourRange.End, err = alertCheckHour(rng[1]); err != nil {
			return nil, fmt.Errorf("invalid end hour: %s", err)
		}

		start := hourRange.Start[0]*60 + hourRange.Start[1]
		end := hourRange.End[0]*60 + hourRange.End[1]
		if start >= end {
			return nil, fmt.Errorf("end of the hour range (%s) is before its start", hour)
		}

		hourRanges = append(hourRanges, hourRange)
	}
	return hourRanges, nil
}

func alertCheckAndCleanDays(days []int) error {
	for key, day := range days {
		if day < 0 {
			return fmt.Errorf("day can't be less than 0: %d", day)
		}
		if day > 7 {
			return fmt.Errorf("day can't be more than 7: %d", day)
		}

		if day == 7 {
			days[key] = 0
		}
	}
	return nil
}

func tomlAlertToAlert(tAlert *tomlAlert, config *Config) (*Alert, error) {
	var alert Alert

	if tAlert.Disabled == true && config.loadDisabled == false {
		return nil, nil
	}

	if tAlert.Name == "" {
		return nil, errors.New("invalid or missing 'name'")
	}
	alert.Name = tAlert.Name

	if tAlert.Command == "" {
		return nil, errors.New("invalid or missing 'command'")
	}

	scriptPath := path.Clean(config.configPath + "/scripts/alerts/" + tAlert.Command)
	stat, err := os.Stat(scriptPath)

	if err == nil {
		if !stat.Mode().IsRegular() {
			return nil, fmt.Errorf("is not a regular 'script' file '%s'", scriptPath)
		}
		tAlert.Command = scriptPath
	} else {
		path, errp := exec.LookPath(tAlert.Command)
		if errp != nil {
			return nil, fmt.Errorf("'%s' command not found in PATH: %s", tAlert.Command, errp)
		}
		tAlert.Command = path
	}

	alert.Command = tAlert.Command

	_, err = ioutil.ReadFile(alert.Command)
	if err != nil {
		return nil, fmt.Errorf("error reading script file '%s': %s", alert.Command, err)
	}

	if tAlert.Targets == nil {
		return nil, errors.New("no valid 'targets' parameter found")
	}

	if len(tAlert.Targets) == 0 {
		return nil, errors.New("empty 'targets'")
	}
	// explode targets on & and check IsValidTokenName
	hasGeneralClass := false
	for _, targets := range tAlert.Targets {
		if targets == "*" || targets == GeneralClass {
			hasGeneralClass = true
			continue
		}
		tokens := strings.Split(targets, "&")
		for _, token := range tokens {
			ttoken := strings.TrimSpace(token)
			if !IsValidTokenName(ttoken) {
				return nil, fmt.Errorf("invalid 'target' class name '%s'", ttoken)
			}
		}
	}
	alert.Targets = tAlert.Targets

	alert.Arguments = tAlert.Arguments

	hours, err := alertCheckHours(tAlert.Hours)
	if err != nil {
		return nil, fmt.Errorf("'hours' parameter: %s", err)
	}
	alert.Hours = hours

	if err := alertCheckAndCleanDays(tAlert.Days); err != nil {
		return nil, fmt.Errorf("'days' parameter: %s", err)
	}
	alert.Days = tAlert.Days

	if hasGeneralClass == true && len(alert.Hours) > 0 && len(alert.Days) > 0 {
		return nil, fmt.Errorf("a 'general' (or '*') alert can't have hours/days restrictions, since you may miss alerts")
	}

	return &alert, nil
}
