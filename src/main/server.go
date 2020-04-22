package main

import (
	"github.com/goftp/server"
	"myftp/src/dirver"
	"os"
	"fmt"
	"github.com/lunny/log"
	"github.com/Unknwon/goconfig"
	//"dirver"
)

func main() {
	//读取配置文件：
	cfg, _ := goconfig.LoadConfigFile("ftp.conf")
	//端口
	port, _ := cfg.Int("server", "port")
	//使用简单的登录验证，用户名密码写死。
	authUser, _ := cfg.GetValue("auth", "user")
	authPassword, _ := cfg.GetValue("auth", "password")
	//权限：
	permOwner, _ := cfg.GetValue("perm", "owner")
	permGroup, _ := cfg.GetValue("perm", "group")
	//上传目录dir
	rootPath, _ := cfg.GetValue("server", "dir")
	_, err := os.Lstat(rootPath)
	if os.IsNotExist(err) {
		os.MkdirAll(rootPath, os.ModePerm)
	} else if err != nil {
		fmt.Println(err)
		return
	}
	//设置权限。
	factory := &dirver.PutFileOnlyDriverFactory{
		rootPath,
		server.NewSimplePerm(permOwner, permGroup),
	}

	opt := &server.ServerOpts{
		Name:    "go",
		Factory: factory,
		Port:    port,
		Auth:    &server.SimpleAuth{authUser, authPassword},
	}
	// start ftp server
	ftpServer := server.NewServer(opt)
	log.Info("FTP Server", "1.0")
	err = ftpServer.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
