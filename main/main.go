package main

import (
	. "FPGA_gotool/tui"
	. "FPGA_gotool/utils"
	"fmt"
)

func main() {
	config, err := LoadAppConfig()
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	err = InitLog(config)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	// TestLOG()
	// ShowVariables(config)

	req, err := config.CheckRequired()
	if err != nil {
		Log("FATAL", err.Error())
		return
	} else if len(req) > 0 {
		z := TextInputTUI(req)
		Log("DEBUG", "User input: "+fmt.Sprint(z))
		config.AddRequired(z)
	}

	keys := make([]string, 0, len(config.Commands))
	Log("DEBUG", fmt.Sprintf("Available command sets"))
	for i := range config.Commands {
		Log("DEBUG", fmt.Sprintf("Set %s", i))
		keys = append(keys, i)
	}

	if len(config.CmdstoRun) == 0 {
		if !Contains(keys, config.CmdSet) {
			Log("DEBUG", fmt.Sprintf("Command set not found %s", config.CmdSet))
			config.CmdSet = ListMenuTUI(keys)
			// temp = append(temp, check1...)

		} else {
			Log("DEBUG", fmt.Sprintf("Command set found %s", config.CmdSet))
			config.CmdstoRun = config.Commands[config.CmdSet]
		}
	} else {
		Log("DEBUG", fmt.Sprintf("Command set found %s", config.CmdSet))
	}

	vars, err := config.CheckVariables()
	if err != nil {
		Log("FATAL", err.Error())
		return
	} else if len(vars) > 0 {
		res := TextInputTUI(vars)
		config.ParseVar(res)
	}

	// fmt.Println(config.CmdstoRun)

	server := SSH(
		config.User,
		config.Password,
		config.Address,
		config.Port,
		config.Ciphers,
		config.InactivityTimeout,
	)

	Session, err := server.Connect()
	if err != nil {
		Log("FATAL", "Failed to create session: "+err.Error())
		return
	}
	defer Session.Close()
	// return

	Log("DEBUG", "Session created successfully")
	go Session.OutputReader()
	Session.Runner(config.CmdstoRun)
	Session.ShowOutput()
}
