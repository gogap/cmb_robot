package main

import (
	"crypto/rc4"
	"encoding/base64"
	"fmt"
	"github.com/gogap/logrus_mate"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"sync"
	"syscall"
	"time"

	"github.com/go-akka/configuration"
	"github.com/gogap/cmb_robot/monitor"
	"github.com/gogap/cmb_robot/robot"
	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/gogap/logrus_mate/hooks/bearychat"
	_ "github.com/gogap/logrus_mate/hooks/expander"
)

func main() {

	var err error
	defer func() {
		if err != nil {
			logrus.Errorln(err)
			return
		}
	}()

	logrus_mate.Hijack(logrus.StandardLogger(), logrus_mate.ConfigFile("log.conf"))

	fmt.Print("请输入配置文件密码:")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return
	}

	fmt.Println()

	conf, err := getConfig(bytePassword, "cmb-robot.conf")
	if err != nil {
		logrus.Errorln(err)
		return
	}

	if conf == nil {
		err = fmt.Errorf("加载配置文件失败，请检查密码是否正确")
		return
	}

	wg := sync.WaitGroup{}

	err = startRobot(&wg, conf)
	if err != nil {
		return
	}

	wg.Wait()
}

func getConfig(bytePassword []byte, filename string) (conf *configuration.Config, err error) {
	defer func() {
		recover()
	}()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	src, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return
	}

	cipher, err := rc4.NewCipher(bytePassword)
	if err != nil {
		return
	}

	dst := make([]byte, len(src))

	cipher.XORKeyStream(dst, src)

	conf = configuration.ParseString(string(dst))

	return
}

func startRobot(wg *sync.WaitGroup, conf *configuration.Config) (err error) {
	bot, err := robot.NewRobot(conf)
	if err != nil {
		return
	}

	mon, err := monitor.NewCMBMonitor(conf)
	if err != nil {
		return
	}

	username := conf.GetString("username")

	wg.Add(1)

	go func(bot *robot.Robot, mon *monitor.CMBMonitor) {

		logrus.WithField("username", username).Infoln("开始监控......")

		runMode := robot.RunModeReListen | robot.RunModeReLogin

		pingExceptionCountMax := 3
		pingExceptionCount := 0

		for {

			excepetion := false

			if e := mon.Ping(); e != nil {
				excepetion = true
				pingExceptionCount++
				if pingExceptionCount == 1 {
					logrus.WithField("username", username).WithError(e).Warnln("PING 业务状态开始抖动")
				}
				time.Sleep(time.Second * 10)
			}

			if !excepetion {
				if pingExceptionCount > 0 {
					logrus.WithField("username", username).Infof("PING 业务状态抖动恢复, 抖动次数: %d", pingExceptionCount)
				}
				pingExceptionCount = 0
				time.Sleep(time.Second)
				continue
			}

			if pingExceptionCount < pingExceptionCountMax {
				time.Sleep(time.Second)
				continue
			}

			logrus.WithField("username", username).Errorln("发现异常，即将启动机器人")

			if e := bot.Run(runMode); e != nil {
				if e == robot.ErrWrongLoginPassword || e == robot.ErrWrongUSBKeyPassword {
					logrus.WithField("username", username).WithError(e).Panicln("YOU ENTER THE WRONG PASSWORD!!!!!!!")
				}

				logrus.WithField("username", username).WithError(e).Errorln("机器人执行登录时异常, 30s后将执行应用重启")
				runMode = robot.RunModeRestart | robot.RunModeReListen | robot.RunModeReLogin
			} else {
				logrus.WithField("username", username).Infoln("机器人执行登录成功")
				runMode = robot.RunModeReListen | robot.RunModeReLogin
			}
			time.Sleep(time.Second * 30)
			pingExceptionCount = 0
		}
	}(bot, mon)

	return
}
