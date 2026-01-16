# SSH TUI Automation Framework

## Progress:
- [x] Run everything inside one session (pseudo terminal)
- [x] Logging system
    - [x] Expand the loging method and made it customisable
- [x] User interface TUI (Text User Interface)
    - [x] User will/will not be prompted to enter variable based on the configuration
- [x] Configuration
    - [x] YAML
    - [x] CLI (terminal)
    - [x] Dinamicly alocated variables inside YAML 
    - [x] Dinamicly alocated variables inside CLI (terminal)
- [x] Strict mode to prevent further running if there was unknown variables
- [ ] Merging CLI and YAML arguments
- [ ] Fixing recursion in the CheckVar function 

## Table of Contents
- [Overview](#overview)
- [Usage](#usage)
- [Setup](#setup)
- [Configuration](#configuration)
  - [CLI Arguments](#using-cli-arguments)
  - [YAML Arguments](#using-yaml-arguments)

### Overview
This framework is built for simple SSH automation, entirely coded in Go. 
It leverages pseudo-terminal technology to maintain persistent sessions. 
Users can either input dynamic variables during execution or configure them in a config file. 
The framework allows easy creation of automation commands that are executed automatically during the process. 
It also enables dynamic generation of text input fields for variables, prompting users to provide values. 
For repetitive settings, a **.yaml** configuration file can be defined. 
A command-line interface (CLI) is provided for interaction.

Before you proceed, make sure that you have basic knowledge of [SSH](https://en.wikipedia.org/wiki/Secure_Shell), [Terminal](https://www.zentyal.com/news/linux-commands/) or [CMD](https://myelo.elotouch.com/support/s/article/Common-Windows-Command-Prompt-CMD-Commands)

### Usage
In your terminal, run `main.exe`, and the app will attempt to read the **main.yaml** configuration file. Alternatively, you can pass inline arguments like this:
```
main.exe -a myserverhostname -p 12345
```
You can also use both the **main.yaml** file and comand line arguments at the same time, with comand line arguments overriding the YAML values if provided.

##### Setup
To use the app, you need to pass certain arguments. There are three options:

###### 1. YAML
Place the **main.yaml** configuration file in the same folder as the application, where you can define all the necessary rules for the app to run.

###### 2. CLI
Pass arguments via the command line, which take priority over the values in the **main.yaml** file.

###### 3. YAML + CLI
Combine both solutions, where the command-line arguments will override the YAML file values.

### Configuration
##### Using CLI Arguments

Example of **CLI** arguments

> main.exe -a myserverhostname -p 12345 -un username -pw supersecretpassword \
 -ciphers "aes256-cbc aes192-cbc aes128-cbc 3des-cbc" -cmd "`ip a` `help`" \
 -loglevel 3 -showtime true -logtofile false -noyaml true



**list of all available arguments for CLI**

``` cli
address
port
user
password
ciphers
logsetup
  loglevel
  showtime
  logtofile
  logtoterminal
  showlevel
  dir
  prefix
  name
  suffix
appsetup
  noyaml
  strictmode
  inactivitytimeout
  maxretries
commands
```

##### Using YAML Arguments
You can create configuration (main.yaml) file and use it to save your configuration.
Example of **.YAML** configuration

```yaml
address: hostname # This is required, 
                   # but can also be entered during runtime 
                   # (e.g., ip/hostname). 
                   # Default values can be "10.23.44.241" or "myserver".
port: 22 # This is required, but can also be entered during runtime. 
         # Default port is 22.
user: root # This is required, but can also be entered during runtime.
password: supersecretpassword # This is required, but can also be entered during runtime.
ciphers: # This is optional but cannot be entered during runtime. 
         # If no ciphers are specified, the default value will be: 
         # "aes128-gcm@openssh.com", "aes256-gcm@openssh.com", 
         # "chacha20-poly1305@openssh.com", "aes256-ctr", 
         # "aes192-ctr", "aes256-ctr".
  - aes256-ctr
  - aes256-ctr
logsetup: # Log setup, where you can modify how logs should appear.
  loglevel: 1 # TRACE:1, DEBUG:2, INFO:3, WARN:4, ERROR:5, FATAL:6. 
              # If loglevel is 5, you will only see ERROR and FATAL. 
              # If set to 1, all messages will be visible. Default log level is 3.
  # loglevel: 3
  showtime: true # If true, the timestamp will be included in the log entry: 
                 # "2025/02/07 14:38:17 [DEBUG] This is a debug message". 
                 # If false, the message will be shorter: "[DEBUG] This is a debug message".
  logtofile: true # If true, log output will be written to a file. 
                  # Set to false to disable logging to a file.
  logtoterminal: true # If true, log output will be displayed in the terminal. 
                      # Set to false to disable logging to terminal.
  showlevel: true # If true, the log level will be included in the log entry: 
                  # "2025/02/07 14:38:17 [DEBUG] This is a debug message". 
                  # If false, the message will be shorter: "2025/02/07 14:38:17 This is a debug message".
  dir: ./logs # Directory where logs should be stored.
  prefix: dev # Prefix for your log filename, e.g., dev_gfdfhsg.log.
  name: test # Name for your log file, e.g., gfdfhsg_name.log.
  suffix: .log # Suffix for your log file, e.g., dev_name.log.
appsetup:
  # Between configuration file and command-line arguments, command-line arguments will take precedence.

  # If strictmode is enabled, the app will exit if required arguments are missing. 
  # If strictmode is disabled, the app will prompt the user to enter the missing information.

  # noyaml: true  # This argument should only be available via command-line arguments.
  strictmode: true # Prevents the app from running if unknown variables are detected. 
                   # This minimizes user interaction with the app.
  inactivitytimeout: 15 # Prevents infinite recursion. The maximum recursion depth is 15.
  maxretries: 10 # Maximum retry attempts allowed.

cmdset: Scan Linux # This is an optional value. If set, the user will not be prompted 
                # to select the menu item during runtime. The value must exist 
                # in the commands section.
variables:  # This is the variable section, where each variable can be defined along with 
           # its corresponding value. For example, the key "sm" should have the value 
           # "sh manager".
  myvariable: |-
    help
    ls -al
    pwd
commands: # This is the command section, where each section contains a list of commands 
          # that can be executed.
  Scan Linux:
    - help
  Scan PC:
    - <myvariable>
    - cd /home/user
    - cd <unknownvar>

```
