package main

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jlaffaye/ftp"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var fileLog *os.File
var debuglog *log.Logger

var ftpServerList []Server

type Server struct {
	Ftpserver string
	User      string
	Password  string
}
type MyConfig struct {
	Dir        string
	Serverlist []Server
}

type Commfun struct {
	nowfile string
}

func NewCommfun() (*Commfun, error) {
	cf := &Commfun{}
	return cf, nil
}

/**
获取系统当前时间字符串
*/
func getSystemTimeStr(formatTimeStr string) string {
	timeUnix := time.Now().Unix() //已知的时间戳
	systemTimeStr := time.Unix(timeUnix, 0).Format(formatTimeStr)
	return systemTimeStr
}

func PrintLog(err error) {
	if fileLog == nil {
		logfilename, _ := os.Getwd()
		logfilename = "./opt.log"
		log.Println(logfilename)
		if _, err := os.Stat(logfilename); err == nil {
			//err = os.Rename(logfilename, logfilename+"."+getSystemTimeStr("20060102"))
			err = os.Rename(logfilename, logfilename+"bak")
			if err != nil {
				err = os.Remove(logfilename)
				if err != nil {
					log.Println("删除成功")
				}
			}
		}
		fileLog, err = os.Create(logfilename)
		if err != nil {
			log.Println("日志创建出错：", err)
		}
		debuglog = log.New(fileLog, "", log.LstdFlags)
	}
	if nil != err {
		debuglog.Println("日志：", err)
	}
}

func (g *Commfun) Monitor(fullPath string) {
	start := time.Now()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		debuglog.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					debuglog.Println("Write file:", event.Name)
					processFile(event.Name)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					debuglog.Println("Create file:", event.Name)
					//if IsFile(event.Name) {
					//	go processFile(event.Name)
					//} else {
					//	debuglog.Println(event.Name, "是文件夹")
					//}

				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					debuglog.Println("Rename file:", event.Name)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					debuglog.Println("Remove file:", event.Name)
				} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					debuglog.Println("Chmod file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				debuglog.Println("error:", err)
			}

		}
	}()

	fullPath, _ = filepath.Abs(fullPath)
	err = watcher.Add(fullPath)
	//filepath.Walk(fullPath, func(path string, fi os.FileInfo, err error) error {
	//
	//	if nil == fi {
	//		PrintLog(err)
	//		return err
	//	}
	//	if fi.IsDir() {
	//
	//		err = watcher.Add(strings.Replace(path, "\\", "\\\\", -1))
	//		if err != nil {
	//			debuglog.Println(err)
	//		}
	//		return err
	//	}
	//	return nil
	//})
	log.Println("监控文件处理监听处理时间约%f秒", time.Now().Sub(start).Seconds())
	if err != nil {
		debuglog.Println(err)
	}
	<-done
}

func IsFile(f string) bool {
	fi, e := os.Stat(f)
	if e != nil {
		return false
	}
	return !fi.IsDir()
}

func sendFTP(filename string, server Server) {
	debuglog.Println("sendFTP.....", filename)

	ftpserver := server.Ftpserver
	ftpuser := server.User
	pw := server.Password
	//ftpserver := "192.168.75.140:22221"
	//ftpuser := "ftpuser"
	//pw := "ftpuser"

	localFile := filename
	remoteSavePath := "test"
	saveName := filepath.Base(localFile)

	ftp, err := ftp.Connect(ftpserver)
	if err != nil { // 校验如果连接失败，不往下执行
		debuglog.Println(err)
		return
	}
	var loginErr error
	loginErr = ftp.Login(ftpuser, pw)
	if loginErr != nil {
		debuglog.Println(err)
	}
	//注意是 pub/log，不能带“/”开头
	ftp.ChangeDir("/")
	dir, err := ftp.CurrentDir()
	debuglog.Println(dir)
	ftp.MakeDir(remoteSavePath)
	ftp.ChangeDir(remoteSavePath)
	dir, _ = ftp.CurrentDir()
	debuglog.Println(dir)
	file, err := os.Open(localFile)
	if err != nil {
		debuglog.Println(err)

	}
	defer file.Close()
	err = ftp.Stor(saveName, file)
	if err != nil {
		debuglog.Println(err)

	}
	defer ftp.Logout()
	defer ftp.Quit()
	debuglog.Println("success upload file:", localFile)
}

func processFile(filename string) {
	for {
		filetemp, errtemp := os.Open(filename)
		filetemp.Close()
		if errtemp != nil {
			//time.Sleep(1 * time.Second)
			log.Println("文件被占用", filename)
		} else {
			for _, value := range ftpServerList {
				sendFTP(filename, value)
				//go sendFTP(filename, value)// 不能使用协程，抢占文件资源，导致死锁
			}
			break
		}
		defer filetemp.Close()
	}

	debuglog.Println("processFile......end")

}

func loadConfig() MyConfig {
	data, err := ioutil.ReadFile("monitorconfig.json")
	if err != nil {
		log.Println(err)
	}
	//读取的数据为json格式，需要进行解码
	//var v []Server
	var v MyConfig
	err = json.Unmarshal(data, &v)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("v==", v)
	//for _, value := range v {
	//	fmt.Println(value.Ftpserver, value.User, value.Password)
	//}
	return v
}

func main() {
	myconfig := loadConfig()
	ftpServerList = myconfig.Serverlist

	done := make(chan bool)
	cn, err := NewCommfun()
	PrintLog(err)
	go cn.Monitor(myconfig.Dir)
	<-done

}
