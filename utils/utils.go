package utils

import (
	"io"
	"os"
	"fmt"
	"log"
	"flag"
	"sync"
	"time"
	"bufio"
	"regexp"
	"reflect"
	"strconv"
	"strings"
	"encoding/base64"
	"gopkg.in/yaml.v2"
	"golang.org/x/crypto/ssh"
)

type Session struct {
	In      io.WriteCloser
	Out     io.Reader
	Channel chan string
	Signal  chan time.Time
	Timeout int
}

func (s Session) Run(command string) error {
	Log("DEBUG", "Executing command: "+command)
	_, err := fmt.Fprintf(s.In, command+"\n")
	if err != nil {
		Log("ERROR", "Failed to execute command: "+err.Error())
	}
	Log("DEBUG", "Command executed successfully")
	return nil
}

func (s Session) Runner(commands []string) error {
	for _, command := range commands {
		_, err := fmt.Fprintf(s.In, command+"\n")
		if err != nil {
			Log("ERROR", "Failed to execute command: "+err.Error())
		}
	}
	return nil
}

func (s Session) Close() { s.In.Close() }

func (s Session) OutputReader() {
	scanner := bufio.NewScanner(s.Out)
	for scanner.Scan() {
		s.Channel <- scanner.Text()
	}
	close(s.Channel)
}

func (s Session) CheckTimeOut() {
	// t := 20
	for {
		select {
		case <-s.Signal:
			continue
		case <-time.After(time.Duration(s.Timeout) * time.Second):
			// case <-time.After(time.Duration(t) * time.Second):
			Log("INFO", "Session timed out after "+fmt.Sprint(s.Timeout)+" seconds of inactivity")
			s.Close()
			close(s.Channel)
			return
		}
	}
}

func (s Session) ShowOutput() {
	go s.CheckTimeOut()
	for line := range s.Channel {
		s.Signal <- time.Now()
		Log("INFO", line)
	}
}

type LOG struct {
	log_dir   string
	log_file  string
	logMutex  sync.Mutex
	Loglevel  int
	Filelog   string
	Terminal  string
	Showtime  string
	Showlevel string
	Loglevels map[string]int
}

var (
	l              LOG
	defaultciphers = []string{
		"aes128-gcm@openssh.com", "aes256-gcm@openssh.com",
		"chacha20-poly1305@openssh.com",
		"aes128-ctr", "aes192-ctr", "aes256-ctr",
	}
	defc = &defaultciphers
)

func InitLog(c *Config) error {
	lc := c.LogSetup
	if lc["prefix"] != "" {
		lc["prefix"] = lc["prefix"] + "_"
	}
	if lc["name"] == "" {
		lc["name"] = time.Now().Format("2006-01-02_15-04-05")
	}
	if lc["suffix"] == "" {
		lc["suffix"] = ".log"
	}
	if lc["dir"] == "" {
		lc["dir"] = "./logs/"
	}
	if !strings.HasSuffix(lc["dir"], "/") {
		lc["dir"] = lc["dir"] + "/"
	}
	if lc["loglevel"] == "" {
		lc["loglevel"] = "3"
	}
	Loglevels := make(map[string]int)
	Loglevels["TRACE"] = 1
	Loglevels["DEBUG"] = 2
	Loglevels["INFO"] = 3
	Loglevels["WARN"] = 4
	Loglevels["ERROR"] = 5
	Loglevels["FATAL"] = 6
	// level, err := strconv.ParseInt(lc["loglevel"], 10, 64)
	// if err != nil {
	// 	return err
	// }
	// lvl := int(level)
	level, err := strconv.Atoi(lc["loglevel"])
	if err != nil {
		err = fmt.Errorf("ERROR parsing loglevel: %v", err)
		return err
	}

	l = LOG{
		log_dir:   lc["dir"],
		log_file:  lc["prefix"] + lc["name"] + lc["suffix"],
		logMutex:  sync.Mutex{},
		Loglevel:  level,
		Filelog:   lc["logtofile"],
		Terminal:  lc["logtoterminal"],
		Showtime:  lc["showtime"],
		Showlevel: lc["showlevel"],
		Loglevels: Loglevels,
	}
	return nil
}

func Log(loglevel string, message string) {
	var msg string
	if l.Showlevel == "true" {
		msg = "[" + loglevel + "] " + message
	} else {
		msg = message
	}
	if l.Showtime != "true" {
		log.SetFlags(0)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	if l.Loglevels[loglevel] < l.Loglevel {
		return
	}
	if l.Terminal == "true" {
		log.Println(msg)
	}
	if l.Filelog != "true" {
		return
	}
	if _, err := os.Stat(l.log_dir); os.IsNotExist(err) {
		if err := os.MkdirAll(l.log_dir, 0755); err != nil {
			log.Fatalf("ERROR creating log directory: %v", err)
		}
	}
	l.logMutex.Lock()
	defer l.logMutex.Unlock()
	f, err := os.OpenFile(l.log_dir+l.log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ERROR opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println(msg)
	log.SetOutput(os.Stdout)
}

func TestLOG() {
	Log("FATAL", "---------------------------------- APP STARTED ----------------------------------")
	Log("TRACE", "This is a trace message")
	Log("DEBUG", "This is a debug message")
	Log("INFO", "This is an info message")
	Log("WARN", "This is a warning message")
	Log("ERROR", "This is an error message")
	Log("FATAL", "This is a fatal message")
}

type MachineDetails struct {
	username string
	password string
	hostname string
	port     string
	ciphers  []string
	timeout  int
}

func SSH(username string, password string, address string, port string, ciphers []string, timeout int) MachineDetails {
	Log("TRACE", "Creating SSH server details; function name: SSH()")
	if ciphers == nil {
		ciphers = *defc
		Log("DEBUG", "No ciphers provided, using default ciphers")
	} else {
		Log("DEBUG", "Ciphers was provided")
	}
	Log("TRACE", "Cipher list: "+fmt.Sprint(ciphers))
	return MachineDetails{
		username: username,
		password: password,
		hostname: address,
		port:     port,
		ciphers:  ciphers,
		timeout:  timeout,
	}
}

func (server MachineDetails) Connect() (Session, error) {
	Log("TRACE", "Connecting to SSH server; function name: Connect()")
	ConfigCmd := &ssh.ClientConfig{
		User:            server.username,
		Auth:            []ssh.AuthMethod{ssh.Password(server.password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config:          ssh.Config{Ciphers: server.ciphers},
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	Log("DEBUG", "Connecting to "+server.hostname+":"+server.port)
	client, err := ssh.Dial("tcp", server.hostname+":"+server.port, ConfigCmd)
	if err != nil {
		return Session{}, err
	}
	Log("DEBUG", "Creating SSH session")
	session, err := client.NewSession()
	if err != nil {
		return Session{}, err
	}
	Log("DEBUG", "Requesting pseudo-terminal xterm for SSH session")
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return Session{}, err
	}
	session.SendRequest("shell", true, nil)
	sshIn, _ := session.StdinPipe()
	sshOut, _ := session.StdoutPipe()
	channel := make(chan string)
	signal := make(chan time.Time)

	return Session{
		In:      sshIn,
		Out:     sshOut,
		Channel: channel,
		Signal:  signal,
		Timeout: server.timeout,
	}, nil
}

type Config struct {
	Address           string
	Port              string
	User              string
	Password          string
	Ciphers           []string
	InactivityTimeout int
	MaxRetries        int
	CmdSet            string
	LogSetup          map[string]string
	AppSetup          map[string]string
	Variables         map[string]string
	Commands          map[string][]string
	CmdstoRun         []string
}

func (c *Config) Parse(configFile string) (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return c, err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&c); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) CheckRequired() ([]string, error) {
	Log("TRACE", "Checking for required variables; function name: CheckRequired()")
	required := []string{"Address", "Port", "User", "Password"}
	Log("DEBUG", "Checking for required variables: "+fmt.Sprint(required))
	missing := []string{}
	v := reflect.ValueOf(c).Elem()
	t := v.Type()
	// t := reflect.TypeOf(c)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if v.Field(i).IsZero() && Contains(required, field.Name) {
			missing = append(missing, field.Name)
			Log("DEBUG", "Value for required variable missing: "+field.Name)
		}
	}
	if len(missing) > 0 && c.AppSetup["strictmode"] == "true" {
		return missing, error(fmt.Errorf("Missing required variables: %s", fmt.Sprint(missing)))
	} else if len(missing) > 0 {
		Log("DEBUG", "Entering missing variables")
	} else {
		Log("DEBUG", "No missing variables found")
	}

	if c.CmdstoRun == nil {
		Log("DEBUG", "CmdstoRun not set making empty slice")
		c.CmdstoRun = make([]string, 0)
	}
	// fmt.Println("Map: ", c.AppSetup)
	if c.AppSetup == nil {
		Log("DEBUG", "AppSetup not set, using default values")
		c.AppSetup = make(map[string]string)
	}

	if c.AppSetup["maxretries"] == "" {
		Log("DEBUG", "MaxRetries not set, using default value of 10")
		c.AppSetup["maxretries"] = "10"
	}
	i, err := strconv.Atoi(c.AppSetup["maxretries"])
	if err != nil {
		return missing, err
	}
	c.MaxRetries = i

	if c.AppSetup["inactivitytimeout"] == "" {
		Log("DEBUG", "InactivityTimeout not set, using default value of 100")
		c.AppSetup["inactivitytimeout"] = "100"
	}
	to, err := strconv.Atoi(c.AppSetup["inactivitytimeout"])
	if err != nil {
		return missing, err
	}
	c.InactivityTimeout = to

	return missing, nil
}

func (c *Config) AddRequired(vars map[string]string) {
	Log("TRACE", "Adding required variables; function name: AddRequired()")
	v := reflect.ValueOf(c).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if field.Type.Kind() == reflect.String {
			if newVal, exists := vars[field.Name]; exists {
				fieldValue.SetString(newVal)
			}
		}
	}
}

func (c *Config) CheckVariables() ([]string, error) {
	Log("TRACE", "Checking for variables; function name: CheckVariables()")
	loopcount := make(map[string]int)
	temp := make(map[string]bool)
	res := []string{}
	for i := range c.CmdstoRun {
		k := &c.CmdstoRun[i]
		for _, v := range c.CheckVar(k, &loopcount) {
			temp[v] = true
		}
	}
	for k := range temp {
		res = append(res, k)
	}

	if len(res) > 0 && c.AppSetup["strictmode"] == "true" {
		return res, error(fmt.Errorf("Missing variables: %s", fmt.Sprint(res)))
	} else if len(res) > 0 {
		Log("DEBUG", "Entering missing variables")
	} else {
		Log("DEBUG", "No missing variables found")
	}

	return res, nil
}

func (c Config) ParseVar(vars map[string]string) {
	Log("TRACE", "Parsing variables; function name: ParseVar()")
	for cmd := range c.CmdstoRun {
		Log("DEBUG", "Parsing command: "+c.CmdstoRun[cmd])
		for k, v := range vars {
			Log("DEBUG", "Replacing variable: "+k+" with value: "+v)
			c.CmdstoRun[cmd] = strings.Replace(c.CmdstoRun[cmd], "<"+k+">", v, -1)
		}
	}
}

func (c *Config) CheckVar(str *string, loopcount *map[string]int) []string {
	Log("TRACE", "Checking for variables in: "+*str+"; function name: CheckVar()")
	temp := []string{}
	re := regexp.MustCompile(`<([^<>]+)>`)
	matches := re.FindAllStringSubmatch(*str, -1)
	for _, match := range matches {
		// Log("TRACE", "Found variable: "+match[1])
		if _, ok := c.Variables[match[1]]; ok {
			vari := c.Variables[match[1]]
			Log("DEBUG", "Variable found: "+match[1]+" = "+vari)
			(*loopcount)[vari]++
			if (*loopcount)[vari] > c.MaxRetries {
				Log("FATAL", "Detected potential infinite loop with variable '"+match[1]+"' please check your .yaml configuration maxretries value was set to: "+fmt.Sprint(c.MaxRetries))
				os.Exit(1)
			}
			check1 := c.CheckVar(&vari, loopcount)
			if len(check1) > 0 {
				temp = append(temp, check1...)
			}
			*str = strings.Replace(*str, "<"+match[1]+">", vari, -1)
		} else {
			Log("DEBUG", "Variable not found: "+match[1])
			temp = append(temp, match[1])
		}
	}
	return temp
}


func (config *Config) MergeArgs(cli *CLIargs) {
	fmt.Println("--------- MERGING ARGS ---------")
	fmt.Println("--------- CLI ARGS ---------")
	if cli.Address != "" {
		fmt.Println("Address:", cli.Address)
	}
	if cli.Port != "" {
		fmt.Println("Port:", cli.Port)
	}
	if cli.CMD != nil {
		fmt.Println("CMDs:", cli.CMD)
	} else {
		fmt.Println("CMDs: nil")
	}
	
	if cli.Ciphers != nil {
		fmt.Println("Ciphers:", cli.Ciphers)
	} else {
		fmt.Println("Ciphers: nil")
	}
	// fmt.Println("Address:", cli.Address)
	// fmt.Println("Port:", cli.Port)
	// fmt.Println("User:", cli.User)
	


	// if Address != "" {
	// 	config.Address = Address
	// }
	// if Port != "22" {
	// 	config.Port = Port
	// }
	// if User != "" {
	// 	config.User = User
	// }
	// if Password != "" {
	// 	config.Password = Password
	// }
	// if LogDir != "./logs" {
	// 	config.LogSetup["dir"] = LogDir
	// }
	// if LogPrefix != "" {
	// 	config.LogSetup["prefix"] = LogPrefix
	// }
	// if LogName != "" {
	// 	config.LogSetup["name"] = LogName
	// }
	// if LogSuffix != ".log" {
	// 	config.LogSetup["suffix"] = LogSuffix
	// }
	// if InactivityTimeout != 10 {
	// 	config.InactivityTimeout = InactivityTimeout
	// }
	// if CmdSet != "RM" {
	// 	config.CmdSet = CmdSet
	// }
	// if Ciphers != "aes256-cbc aes192-cbc aes128-cbc 3des-cbc" {
	// 	config.Ciphers = strings.Fields(Ciphers)
	// }

}

func LoadAppConfig() (*Config, error) {
	cliargs := &CLIargs{}
	cliargs.ParseArgs()
	// cliargs.ShowVariables()

	// fmt.Println("--------- ARGS ---------")
	// // ShowVariables(*args)
	// args.MergeArgs()
	// args.ShowVariables()

	cfg := Config{}

	config := &cfg


	// config.MergeArgs(cliargs)



	// if args.Noyaml == "true" {
	// 	fmt.Println("Skip yaml config")
	// 	// args.MergeArgs(config)
	// 	return config, nil
	// }
	
	// os.Exit(0)

	CurrentDir, err := os.Executable()
	if err != nil {
		return config, err
	}
	ConfigFile := strings.TrimSuffix(CurrentDir, ".exe") + ".yaml"
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		return config, err
	}
	config, err = config.Parse(ConfigFile)
	return config, err
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ShowVariables(vars interface{}) {
	v := reflect.ValueOf(vars)
	t := reflect.TypeOf(vars)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// print field name and value
		// fmt.Println(field.Name, v.Field(i).Interface())
		//  check if field type is a map
		if field.Type.Kind() == reflect.Map {
			for _, key := range v.Field(i).MapKeys() {
				Log("DEBUG", "Field: "+field.Name+" Key: "+fmt.Sprint(key.Interface())+" Value: "+fmt.Sprint(v.Field(i).MapIndex(key).Interface()))
			}
		} else if field.Type.Kind() == reflect.Slice {
			for j := 0; j < v.Field(i).Len(); j++ {
				Log("DEBUG", "Field: "+field.Name+" Value: "+fmt.Sprint(v.Field(i).Index(j).Interface()))
			}
		} else {
			Log("DEBUG", "Field: "+field.Name+" Value: "+fmt.Sprint(v.Field(i).Interface()))
		}
	}
}

type CLIargs struct {
	Address  string
	Port     string
	User     string
	Password string
	// Cph 	 string
	Ciphers  []string


	Ls_loglevel      string
	Ls_showtime      string
	Ls_logtofile     string
	Ls_logtoterminal string
	Ls_showlevel     string
	Ls_dir           string
	Ls_prefix        string
	Ls_name          string
	Ls_suffix        string

	As_strictmode        string
	As_inactivitytimeout string
	As_maxretries        string

	Noyaml string
	CMD    []string
}

func (c *CLIargs) ShowVariables() {
	fmt.Println("--------- CLI ARGS ---------")
	if c.Address != "" {
		fmt.Println("Address:", c.Address)
	}
	if c.Port != "" {
		fmt.Println("Port:", c.Port)
	}
	if c.CMD != nil {
		fmt.Println("CMDs:", c.CMD)
	} else {
		fmt.Println("CMDs: nil")
	}
	
	if c.Ciphers != nil {
		fmt.Println("Ciphers:", c.Ciphers)
	} else {
		fmt.Println("Ciphers: nil")
	}

	// fmt.Println("Cipher:", c.Ciphers)

	// fmt.Println("Address:", c.Address)
	// fmt.Println("Port:", c.Port)
	// fmt.Println("User:", c.User)
	// fmt.Println("Password:", c.Password)
	// fmt.Println("Ciphers:", strings.Fields(c.Ciphers))
	// fmt.Println("Ls_loglevel:", c.Ls_loglevel)
	// fmt.Println("Ls_showtime:", c.Ls_showtime)
	// fmt.Println("Ls_logtofile:", c.Ls_logtofile)
	// fmt.Println("Ls_logtoterminal:", c.Ls_logtoterminal)
	// fmt.Println("Ls_showlevel:", c.Ls_showlevel)
	// fmt.Println("Ls_dir:", c.Ls_dir)
	// fmt.Println("Ls_prefix:", c.Ls_prefix)
	// fmt.Println("Ls_name:", c.Ls_name)
	// fmt.Println("Ls_suffix:", c.Ls_suffix)
	// fmt.Println("As_strictmode:", c.As_strictmode)
	// fmt.Println("As_inactivitytimeout:", c.As_inactivitytimeout)
	// fmt.Println("As_maxretries:", c.As_maxretries)
	// fmt.Println("Noyaml:", c.Noyaml)
	// fmt.Println("CMDs:", c.CMD)
}

func (c *CLIargs) ParseArgs() {
	flag.StringVar(&c.Address, "a", "", "Server address")
	flag.StringVar(&c.Port, "p", "", "Server port")
	flag.StringVar(&c.User, "un", "", "Username")
	flag.StringVar(&c.Password, "pw", "", "Password")
	// flag.StringVar(&c.Cph, "ciphers", "", "Space-separated list of ciphers")
	flag.Func("ciphers", "Space-separated list of ciphers", func(s string) error {
		c.Ciphers = strings.Fields(s)
		return nil
	})
	flag.StringVar(&c.Ls_loglevel, "loglevel", "", "Log level")
	flag.StringVar(&c.Ls_showtime, "showtime", "", "Show time")
	flag.StringVar(&c.Ls_logtofile, "logtofile", "", "Log to file")
	flag.StringVar(&c.Ls_logtoterminal, "logtoterminal", "", "Log to terminal")
	flag.StringVar(&c.Ls_showlevel, "showlevel", "", "Show level")
	flag.StringVar(&c.Ls_dir, "dir", "", "Where to store logs")
	flag.StringVar(&c.Ls_prefix, "prefix", "", "Log prefix")
	flag.StringVar(&c.Ls_name, "name", "", "Log name")
	flag.StringVar(&c.Ls_suffix, "suffix", "", "Log suffix")
	flag.StringVar(&c.As_strictmode, "strictmode", "", "Strict mode")
	flag.StringVar(&c.As_inactivitytimeout, "inactivitytimeout", "", "Inactivity timeout")
	flag.StringVar(&c.As_maxretries, "maxretries", "", "Max retries")
	flag.StringVar(&c.Noyaml, "noyaml", "", "No yaml")
	flag.Func("cmd", "Command list (supports quoted arguments) each command must be enclosed in backticks (`) and separated by spaces", func(s string) error {
		c.ParseCMDs(s)
		return nil
	})

	err := flag.Parse()
	if err == flag.ErrHelp {
		// Encdocs()
		// Decdocs()
		fmt.Println("If you need aditional help, please check the help-readme.md file")
		os.Exit(0)
	}

}

func (c *CLIargs) ParseCMDs(input string) {
	var current strings.Builder
	inQuotes := false
	for _, char := range input {
		switch char {
		case '`':
			inQuotes = !inQuotes // Toggle quote state
		case ' ':
			if inQuotes {
				current.WriteRune(char) // Keep spaces inside quotes
			} else if current.Len() > 0 {
				c.CMD = append(c.CMD, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	if current.Len() > 0 {
		c.CMD = append(c.CMD, current.String())
	}
}

// var (
// 	readmeMD   = ``
// 	readmeHTML = ``
// )

func readFile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func writeToFile(filename, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		panic(err)
	}
}

func decodeString(encoded string) string {
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic(err)
	}
	return string(decodedBytes)
}

// func Encdocs() {
// 	MDcontent := readFile("README.MD")
// 	encoded1 := base64.StdEncoding.EncodeToString([]byte(MDcontent))
// 	writeToFile("encoded.md", encoded1)
// 	fmt.Println("Encoded content saved to encoded.md")

// 	HTMLcontent := readFile("README.HTML")
// 	encoded2 := base64.StdEncoding.EncodeToString([]byte(HTMLcontent))
// 	writeToFile("encoded.html", encoded2)
// 	fmt.Println("Encoded content saved to encoded.html")
// }

// func Decdocs() {
// 	readme := decodeString(readmeMD)
// 	writeToFile("help-readme.md", readme)
// 	html := decodeString(readmeHTML)
// 	writeToFile("help-readme.html", html)
// }
