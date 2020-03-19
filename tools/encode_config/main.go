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

	filename := "input.conf"

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	cipher, err := rc4.NewCipher(bytePassword)
	if err != nil {
		return
	}

	dst := make([]byte, len(data))

	cipher.XORKeyStream(dst, data)

	base64Str := base64.StdEncoding.EncodeToString(dst)

	ioutil.WriteFile("output.conf", []byte(base64Str), 0644)
}
