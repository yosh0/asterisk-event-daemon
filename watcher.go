package main

import (
  	"bufio"
  	"bytes"
  	"fmt"
  	"net"
  	"time"
	"github.com/takama/daemon"
	"os"
  	"log"
	"flag"
	"encoding/json"
)

const (
    _DAEMON_NAME  = "daemon"
    _DAEMON_DESC  = "WatchHedgeHog"
    timeout       = 10 * time.Second
	_LINE_TERM    = "\r\n"            // packet line separator
	_KEY_VAL_TERM = ":"               // header value separator
	_READ_BUF     = 512               // buffer size for socket reader
	_CMD_END      = "--END COMMAND--" // Asterisk command data end
)

var (
	_PT_BYTES = []byte(_LINE_TERM + _LINE_TERM) // packet separator
    stdlog,
	errlog *log.Logger
	AMIhost, AMIuser, AMIpassword, AMIport string
)

type Config struct  {
	Ami Ami
}

type Ami struct {
	RemotePort string
	RemoteHost string
	Username   string
	Password   string
}

type Message map[string]string

type Service struct {
    daemon.Daemon
}

func (service *Service) Manage() (string, error) {
    usage := "Usage: myservice install | remove | start | stop | status\nconfig 'asterGo.json' should be placed in /etc/asterisk"
    if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
	    	return service.Remove()
		case "start":
	    	return service.Start()
		case "stop":
	    	return service.Stop()
		case "status":
	    	return service.Status()
		default:
	    	return usage, nil
		}
    }
	eventGet()
    return usage, nil
}

func eventGet() {
	conn, _ := net.Dial("tcp", AMIhost + ":" + AMIport)
	fmt.Fprintf(conn, "Action: Login\r\n")
	fmt.Fprintf(conn, "Username:" + AMIuser + "\r\n")
	fmt.Fprintf(conn, "Secret:" + AMIpassword + "\r\n\r\n")
	r := bufio.NewReader(conn)
	pbuf := bytes.NewBufferString("")
	buf := make([]byte, _READ_BUF)
	for {
		rc, err := r.Read(buf)
		if err != nil {
				continue
		}
		wb, err := pbuf.Write(buf[:rc])
		if err != nil || wb != rc { // can't write to data buffer, just skip
				continue
		}
		for pos := bytes.Index(pbuf.Bytes(), _PT_BYTES); pos != -1; pos = bytes.Index(pbuf.Bytes(), _PT_BYTES) {
			bp := make([]byte, pos + len(_PT_BYTES))
			r, err := pbuf.Read(bp)                    // reading packet to separate puffer
			if err != nil || r != pos + len(_PT_BYTES) { // reading problems, just skip
				continue
			}
			m := make(Message)
			for _, line := range bytes.Split(bp, []byte(_LINE_TERM)) {
				if len(line) == 0 {
					continue
				}
				kvl := bytes.Split(line, []byte(_KEY_VAL_TERM))
				if len(kvl) == 1 {
					if string(line) != _CMD_END {
						m["CmdData"] += string(line)
					}
					continue
				}
				k := bytes.TrimSpace(kvl[0])
				v := bytes.TrimSpace(kvl[1])
				m[string(k)] = string(v)
			}
			eventHandler(m)
		}
	}
}

func eventHandler(E map[string]string) {
  	if (E["Event"] == "UserEvent") {
		// Some Action
	}
}

func init() {
	file, e1 := os.Open("/etc/asterisk/asterGo.json")
	if e1 != nil {
		fmt.Println("Error:	", e1)
	}
	decoder := json.NewDecoder(file)
	conf := Config{}
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("Error: ", err)
	}

	flag.StringVar(&AMIport, "port", conf.Ami.RemotePort, "AMI port")
	flag.StringVar(&AMIhost, "host", conf.Ami.RemoteHost, "AMI host")
	flag.StringVar(&AMIuser, "user", conf.Ami.Username, "AMI user")
	flag.StringVar(&AMIpassword, "password", conf.Ami.Password, "AMI secret")
	flag.Parse()
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
    srv, err := daemon.New(_DAEMON_NAME, _DAEMON_DESC)
    if err != nil {
		errlog.Println("Error 1: ", err)
		os.Exit(1)
    }
    service := &Service{srv}
    status, err := service.Manage()
    if err != nil {
		errlog.Println(status, "\nError 2: ", err)
		os.Exit(1)
    }
    fmt.Println(status)
}

func Logger(s map[string]string) {
  	tf := timeFormat()
  	f, _ := os.OpenFile("/var/log/asterisk/custom.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
 	log.SetOutput(f)
  	log.Print(tf)
  	log.Print(s)
  	fmt.Println(s)
}

func timeFormat() (string) {
	t := time.Now()
  	tf := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d\n", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
  	return tf
}
