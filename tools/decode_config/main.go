package main

import (
	"crypto/rc4"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	var err error
	defer func() {
		if err != nil {
			logrus.Errorln(err)
			return
		}
	}()

	fmt.Print("请输入配置文件密码:")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return
	}

	fmt.Print(len(bytePassword))

	filename := "input.conf"

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

	cipher.XORKeyStream(dst, []byte(src))

	ioutil.WriteFile("output.conf", dst, 0644)
}
