// FTP Client for Google Go language.
// Author: smallfish <smallfish.xy@gmail.com>

package ftp

import (
  "os"
	"fmt"
	"net"
	"strconv"
	"strings"
  "crypto/tls"
  "bytes"
)

type FTP struct {
	host    string
	port    int
	user    string
	passwd  string
	pasv    int
	cmd     string
	Code    int
	Message string
	Debug   bool
	stream  []byte
	conn    net.Conn
	Error   error
}

func (ftp *FTP) debugInfo(s string) {
	if ftp.Debug {
		fmt.Println(s)
	}
}

func (ftp *FTP) Connect(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	ftp.conn, ftp.Error = net.Dial("tcp", addr)
	ftp.Response()
	ftp.host = host
	ftp.port = port
}

func (ftp *FTP) Login(user, passwd string) {
	ftp.Request("USER " + user)
	ftp.Request("PASS " + passwd)
	ftp.user = user
	ftp.passwd = passwd
}

func (ftp *FTP) Auth() {
  ftp.Request("AUTH TLS")
  conn := tls.Client(ftp.conn, &tls.Config{InsecureSkipVerify: true})
  ftp.Error = conn.Handshake()
  if ftp.Error != nil {
    fmt.Printf("tls error: %s\n", ftp.Error)
    os.Exit(1)
  }
  ftp.conn = conn
}

func (ftp *FTP) Response() (code int, message string) {
  message = ""
  code = 0
  var buffer bytes.Buffer
  Again:
  	ret := make([]byte, 1024)
	  n, _ := ftp.conn.Read(ret)
  	msg := string(ret[:n])
    if len(msg) == 0 {
      goto Again
    }
	  code, _ = strconv.Atoi(msg[:3])
  	buffer.WriteString(msg[4 : len(msg)-2])
    tmp := strings.Split(msg, "\n")
    if tmp[len(tmp)-2][3] == 45 {
      goto Again
    }
  if code == 0 {
    buffer.WriteString("\n")
    goto Again
  }
  message = buffer.String()
	ftp.debugInfo("<*cmd*> " + ftp.cmd)
	ftp.debugInfo(fmt.Sprintf("<*code*> %d", code))
	ftp.debugInfo("<*message*> " + message)
	return
}

func (ftp *FTP) RawRequest(cmd string) {
  ftp.conn.Write([]byte(cmd + "\r\n"))
  ftp.cmd = cmd
}

func (ftp *FTP) Request(cmd string) {
  ftp.RawRequest(cmd)
	ftp.Code, ftp.Message = ftp.Response()
	if cmd == "PASV" || cmd == "CPSV" {
		start, end := strings.Index(ftp.Message, "("), strings.Index(ftp.Message, ")")
		s := strings.Split(ftp.Message[start:end], ",")
		l1, _ := strconv.Atoi(s[len(s)-2])
		l2, _ := strconv.Atoi(s[len(s)-1])
		ftp.pasv = l1*256 + l2
	}
	if (cmd != "PASV") && (ftp.pasv > 0) {
		ftp.Message = newRequest(ftp.host, ftp.pasv, ftp.stream)
		ftp.pasv = 0
		ftp.stream = nil
		ftp.Code, _ = ftp.Response()
	}
}

func (ftp *FTP) Pasv() {
	ftp.Request("PASV")
}

func (ftp *FTP) Cpsv() {
  ftp.Request("CPSV")
}

func (ftp *FTP) Pwd() {
	ftp.Request("PWD")
}

func (ftp *FTP) Cwd(path string) {
	ftp.Request("CWD " + path)
}

func (ftp *FTP) Mkd(path string) {
	ftp.Request("MKD " + path)
}

func (ftp *FTP) Size(path string) (size int) {
	ftp.Request("SIZE " + path)
	size, _ = strconv.Atoi(ftp.Message)
	return
}

func (ftp *FTP) List() {
	ftp.Pasv()
	ftp.Request("LIST")
}

func (ftp *FTP) Statl(path string) {
  var buffer bytes.Buffer
  buffer.WriteString("STAT -l ")
  buffer.WriteString(path)
  ftp.Request(buffer.String())
}

func (ftp *FTP) Stor(file string, data []byte) {
	ftp.Pasv()
	if data != nil {
		ftp.stream = data
	}
	ftp.Request("STOR " + file)
}

func (ftp *FTP) Quit() {
	ftp.Request("QUIT")
	ftp.conn.Close()
}

// new connect to FTP pasv port, return data
func newRequest(host string, port int, b []byte) string {
	conn, _ := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	defer conn.Close()
	if b != nil {
		conn.Write(b)
		return "OK"
	}
	ret := make([]byte, 4096)
	n, _ := conn.Read(ret)
	return string(ret[:n])
}
