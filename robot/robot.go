package robot

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/AllenDang/w32"
	"github.com/go-akka/configuration"
	"github.com/sirupsen/logrus"
)

var (
	ErrWrongLoginPassword      = errors.New("wrong login password")
	ErrWrongUSBKeyPassword     = errors.New("wrong usb key password")
	ErrOpenUSBKeyFailure       = errors.New("open usb key failure")
	ErrEmptyUserNameWhileLogin = errors.New("empty username while login")
	ErrEmptyUserName           = errors.New("empty username")
	ErrBadLoginPasswordLength  = errors.New("login password lenght is not 8")
	ErrBadUSBKeyPasswordLength = errors.New("usb key password lenght is not 8")
	ErrLoginWindowNotFound     = errors.New("login window not found")
	ErrLoginWindowNotCorrect   = errors.New("login window status not correct")
	ErrBadPasswordBoxCount     = errors.New("password box count not correct")
	ErrConfirmLoginFailure     = errors.New("confirm login failure")
	ErrLoginFrmDidNotDismissed = errors.New("login form did not dismissed")
	ErrProcessNotAlive         = errors.New("process not alive")
	ErrRestartFailure          = errors.New("restart failure")
	ErrListenFailure           = errors.New("listen failure")
	ErrLoginTimeout            = errors.New("login timeout")
	ErrLoginFailure            = errors.New("login failure")
	ErrNetworkError            = errors.New("network error")
	ErrCrashWindow             = errors.New("crash window found")
)

type RunMode int

const (
	RunModeRestart  RunMode = 1
	RunModeReListen RunMode = 2
	RunModeReLogin  RunMode = 4
)

var (
	mainFormTitle = syscall.StringToUTF16Ptr("招商银行企业银行直联")
	mainFormClass = syscall.StringToUTF16Ptr("TMainFrm")
)

type Robot struct {
	userName       string
	loginPassword  string
	usbKeyPassword string
	path           string
	listenAddr     string
	filename       string
}

func NewRobot(config *configuration.Config) (robot *Robot, err error) {
	userName := config.GetString("username")
	loginPassword := config.GetString("login-password")
	usbKeyPassword := config.GetString("usbkey-password")
	path := config.GetString("path", "C:\\Program Files\\CMB\\FbSdk\\Bin\\FBSdkManager.exe")
	listenAddr := config.GetString("listen-addr", "127.0.0.1:8080")
	filename := filepath.Base(path)
	cmbVersion := config.GetString("cmb-version", "")

	if len(cmbVersion) > 0 {
		mainFormTitle = syscall.StringToUTF16Ptr("招商银行企业银行直联" + cmbVersion)
	}

	if len(userName) == 0 {
		err = ErrEmptyUserName
		return
	}

	if len(loginPassword) != 8 {
		err = ErrBadLoginPasswordLength
		return
	}

	if len(usbKeyPassword) != 8 {
		err = ErrBadUSBKeyPasswordLength
		return
	}

	return &Robot{
		userName:       userName,
		loginPassword:  loginPassword,
		usbKeyPassword: usbKeyPassword,
		path:           path,
		listenAddr:     listenAddr,
		filename:       filename,
	}, nil
}

func (p *Robot) Logout() (err error) {
	pid := p.getMainProcessPID()
	if pid == 0 {
		err = ErrProcessNotAlive
		return
	}

	hwnd := w32.FindWindowW(mainFormClass, mainFormTitle)

	w32.ShowWindow(hwnd, w32.SW_NORMAL)
	w32.SetForegroundWindow(hwnd)

	time.Sleep(time.Second * 2)

	// o+control
	TapKey(w32.VK_CONTROL, 'O')

	for p.closeMessageBox(hwnd, "#32770", "招商银行企业银行直联系统", "确定", "确定要签退用户") {
		logrus.WithField("username", p.userName).Debugln("捕获签退用户的确认提示框, 准备模拟点击确定")
		time.Sleep(time.Second * 2)
	}

	return
}

func (p *Robot) IsLoggedIn() bool {
	hwnd := w32.FindWindowW(mainFormClass, mainFormTitle)

	lvs := listViews(hwnd)

	if len(lvs) != 2 {
		logrus.WithField("username", p.userName).Errorln("ListView数量不等于2")
		return false
	}

	loginName := getLVItem(lvs[IDLV_LOGIN], 0, 0)
	if loginName == p.userName {
		return true
	}
	return false
}

func (p *Robot) Login() (alreadyLoggedin bool, err error) {
	pid := p.getMainProcessPID()
	if pid == 0 {
		err = ErrProcessNotAlive
		return
	}

	hwnd := w32.FindWindowW(mainFormClass, mainFormTitle)

	w32.ShowWindow(hwnd, w32.SW_NORMAL)
	w32.SetForegroundWindow(hwnd)

	time.Sleep(time.Second * 3)

	alreadyLoggedin, err = p.login(hwnd)

	return
}

func (p *Robot) Listen() bool {
	if p.IsListening() {
		return true
	}

	hwnd := w32.FindWindowW(mainFormClass, mainFormTitle)

	w32.ShowWindow(hwnd, w32.SW_NORMAL)
	w32.SetForegroundWindow(hwnd)

	time.Sleep(time.Second * 2)

	// control+b
	TapKey(w32.VK_CONTROL, 'B')

	for i := 0; i < 5; i++ { // 多次尝试关闭。。。。
		p.closeMessageBox(hwnd, "#32770", "招商银行企业银行直联系统", "确定", "HTTP服务已启动")
		time.Sleep(time.Second)
	}

	return p.IsListening()
}

func (p *Robot) StopListen() bool {
	if !p.IsListening() {
		return true
	}

	hwnd := w32.FindWindowW(mainFormClass, mainFormTitle)

	w32.ShowWindow(hwnd, w32.SW_NORMAL)
	w32.SetForegroundWindow(hwnd)

	time.Sleep(time.Second * 2)

	// control+e
	TapKey(w32.VK_CONTROL, 'E')

	for i := 0; i < 5; i++ { // 多次尝试关闭弹窗。。。。
		p.closeMessageBox(hwnd, "#32770", "招商银行企业银行直联系统", "确定", "停止HTTP")
		time.Sleep(time.Second)
	}

	return !p.IsListening()
}

func (p *Robot) RestartProcess() (err error) {

	oldPid := p.getMainProcessPID()
	if oldPid != 0 {
		var proc *os.Process
		proc, err = os.FindProcess(oldPid)
		if err != nil {
			return
		}

		logrus.WithField("username", p.userName).WithField("old_pid", oldPid).Debugln("发现旧的程序")

		if err = proc.Kill(); err != nil {
			return
		}

		for i := 0; i < 30; i++ {
			logrus.WithField("username", p.userName).WithField("old_pid", oldPid).Debugln("等待窗口释放...")
			h := w32.FindWindowW(mainFormClass, mainFormTitle)
			if h == 0 {
				break
			}
			time.Sleep(time.Second)
		}

		logrus.WithField("username", p.userName).WithField("old_pid", oldPid).Debugln("已经关闭旧的程序")
	}

	time.Sleep(time.Second * 3)

	attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	newProc, err := os.StartProcess(p.path, nil, attr)
	if err != nil {
		return
	}

	time.Sleep(time.Second)
	logrus.WithField("username", p.userName).WithField("proc_pid", newProc.Pid).Debugln("新的程序已经启动")

	return
}

func (p *Robot) Run(mode RunMode) (err error) {

reRun:

	if p.getMainProcessPID() == 0 {
		mode = mode | RunModeRestart // 追加启动进程
	}

	if !p.IsListening() {
		mode = mode | RunModeReListen // 如果没有监听，追加监听
	}

	if mode&RunModeRestart == RunModeRestart {
		err = p.RestartProcess()
		if err != nil {
			return
		}
	}

	if mode&RunModeReListen == RunModeReListen {
		if p.IsListening() {
			p.StopListen()
		}

		if !p.Listen() {
			logrus.WithField("username", p.userName).Errorln("进行监听失败")
			err = ErrListenFailure
			return
		}
	}

	if mode&RunModeReLogin == RunModeReLogin {
		if p.IsLoggedIn() {
			err = p.Logout()
		}

		if err == ErrProcessNotAlive {
			err = nil
			goto reRun
		}

		if err != nil {
			return
		}

		_, err = p.Login()
		if err == ErrProcessNotAlive {
			err = nil
			goto reRun
		}

		return
	}

	return
}

func (p *Robot) closeMessageBox(hwnd w32.HWND, windowClassName, windowTitleName, buttonName string, msgContent string) (ret bool) {
	_, pid := w32.GetWindowThreadProcessId(hwnd)
	if pid == 0 {
		return false
	}

	var dlgBoxHwnd []w32.HWND

	fnOfEnumDlg := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT { //HWND hwnd, LPARAM lParam
		className := w32.GetClassNameW(childHwnd)
		title := w32.GetWindowText(childHwnd)

		if windowClassName == className && windowTitleName == title {
			_, pidDlg := w32.GetWindowThreadProcessId(childHwnd)
			if pidDlg != pid {
				return 1
			}

			logrus.WithField("username", p.userName).WithField("HWND", childHwnd).WithField("title", windowTitleName).WithField("class", windowClassName).Debugln("找到消息框")
			dlgBoxHwnd = append(dlgBoxHwnd, childHwnd)
		}
		return 1
	}

	w32.EnumChildWindows(0, fnOfEnumDlg, 0)

	for i := 0; i < len(dlgBoxHwnd); i++ {
		var btnHwnds []w32.HWND
		messageContentFound := false

		fnOfEnumChild := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT {
			windowName := w32.GetWindowText(childHwnd)
			if windowName == buttonName {
				logrus.WithField("username", p.userName).WithField("button", windowName).Debugln("找到消息框按钮")
				btnHwnds = append(btnHwnds, childHwnd)
			}

			if len(msgContent) > 0 {
				if strings.Contains(windowName, msgContent) {
					logrus.WithField("username", p.userName).WithField("message", windowName).WithField("match_message", msgContent).Debugln("消息框内容匹配成功")
					messageContentFound = true
				}
			}

			return 1
		}

		w32.EnumChildWindows(dlgBoxHwnd[i], fnOfEnumChild, 0)

		w32.SetForegroundWindow(dlgBoxHwnd[i])

		if messageContentFound == true || len(msgContent) == 0 {
			logrus.WithField("username", p.userName).Debugln("向按钮发送点击事件")
			for _, btnHwnd := range btnHwnds {
				logrus.WithField("username", p.userName).WithField("parent_hwnd", dlgBoxHwnd[i]).WithField("HWND", btnHwnd).WithField("button", buttonName).Debugln("向消息框按钮发送点击事件")
				w32.SendMessage(btnHwnd, w32.BM_CLICK, 0, 0)
				ret = true
			}
		}
	}

	return

}

func (p *Robot) checkIsLoginWindowUsingUSBKey(hwnd w32.HWND) bool {

	userNameFound := false

	fnOfEnumLoginChild := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT {
		className := w32.GetClassNameW(childHwnd)

		txtUserName := make([]uint16, 255)

		w32.SendMessage(childHwnd, w32.WM_GETTEXT, 255, uintptr(unsafe.Pointer(&txtUserName[0])))

		strUserName := syscall.UTF16ToString(txtUserName)

		if "Edit" == className && p.userName == strUserName {
			logrus.WithField("username", p.userName).WithField("HWND", childHwnd).Debugln("用户名已成功在列表中加载")
			userNameFound = true
			return 0
		}
		return 1
	}

	w32.EnumChildWindows(hwnd, fnOfEnumLoginChild, 0)

	if !userNameFound {
		logrus.WithField("username", p.userName).WithField("HWND", hwnd).Debugln("用户名未加载")
		return false
	}

	totalAltItems := 0
	VisibleAltItems := 0

	fnOfEnumAltTxt := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT {
		className := w32.GetClassNameW(childHwnd)
		if strings.HasPrefix(className, "ATL:") {
			totalAltItems++
			if w32.IsWindowVisible(childHwnd) {
				VisibleAltItems++
			}
		}
		return 1
	}

	w32.EnumChildWindows(hwnd, fnOfEnumAltTxt, 0)

	if totalAltItems != VisibleAltItems {
		logrus.WithField("username", p.userName).WithField("HWND", hwnd).WithField("total", totalAltItems).WithField("visible", VisibleAltItems).Debugln("密码框数量不匹配")
		return false
	}

	return true
}

func (p *Robot) login(mainHwnd w32.HWND) (alreadyLogin bool, err error) {

	// 1. start login window
	logrus.WithField("username", p.userName).Infoln("开始登录")

	classOfLogin := syscall.StringToUTF16Ptr("TOnlineLoginFrm")
	titleOfLogin := syscall.StringToUTF16Ptr("联机登录 (110100)")

relogin:
	// close all old login window
	for {
		oldhwndLogin := w32.FindWindowW(classOfLogin, titleOfLogin)
		if oldhwndLogin != 0 {
			logrus.WithField("username", p.userName).WithField("HWND", oldhwndLogin).Debugln("找到了已经开启的登陆窗口，已将其关闭")
			w32.PostMessage(oldhwndLogin, w32.WM_SYSKEYDOWN, 'X', 1<<29)
			time.Sleep(time.Second * 2)
			continue
		}
		break
	}

	w32.SetForegroundWindow(mainHwnd)
	// control+i
	TapKey(w32.VK_CONTROL, 'I')

	var hwndLogin w32.HWND

	for i := 0; i < 30; i++ {

		logrus.WithField("username", p.userName).Debugln("查找登陆窗口中")

		hwndLogin = w32.FindWindowW(classOfLogin, titleOfLogin)

		if hwndLogin != 0 {
			logrus.WithField("username", p.userName).WithField("HWND", hwndLogin).Debugln("找到登录窗口")
			break
		}

		time.Sleep(time.Second)
	}

	if p.closeMessageBox(mainHwnd, "#32770", "招商银行企业银行直联系统", "确定", "HTTP") {
		time.Sleep(time.Second * 2)
		goto relogin
	}

	if hwndLogin == 0 {
		err = ErrLoginWindowNotFound
		return
	}

	time.Sleep(time.Second * 2)

	// 2. validate window status
	loginFrmCorrect := false
	for i := 0; i < 30; i++ {
		logrus.WithField("username", p.userName).Debugln("正在验证登录窗口的正确性...")
		loginFrmCorrect = p.checkIsLoginWindowUsingUSBKey(hwndLogin)
		if loginFrmCorrect {
			break
		}

		if p.closeMessageBox(mainHwnd, "#32770", "Microsoft Visual C++ Runtime Library", "确定", "") {
			logrus.WithField("username", p.userName).Errorln("发现VC++崩溃窗口")
			err = ErrCrashWindow
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "Abnormal program termination", "确定", "") {
			logrus.WithField("username", p.userName).Errorln("发现异常退出窗口")
			err = ErrCrashWindow
			return
		}

		logrus.WithField("username", p.userName).Debugf("第%d次尝试失败，请确认USBkey已经生效.", i+1)
		time.Sleep(time.Second)
	}

	if !loginFrmCorrect {
		err = ErrLoginWindowNotCorrect
		return
	}

	// 3. send password
	logrus.WithField("username", p.userName).Debugln("准备输入密码")
	var txtHwnds []w32.HWND

	fn := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT { //HWND hwnd, LPARAM lParam
		className := w32.GetClassNameW(childHwnd)

		if strings.HasPrefix(className, "ATL:") {
			txtHwnds = append(txtHwnds, childHwnd)
		}

		return 1
	}

	w32.EnumChildWindows(hwndLogin, fn, 0)

	if len(txtHwnds) != 2 {
		err = ErrBadPasswordBoxCount
		return
	}

	passwords := []string{p.usbKeyPassword, p.loginPassword}

	for i := 0; i < len(txtHwnds); i++ {

		w32.SendMessage(txtHwnds[i], w32.WM_SETFOCUS, 0, 0)
		pwd := passwords[i]

		for _, c := range pwd {
			w32.SetForegroundWindow(hwndLogin)
			TapKey(uint16(c))
		}

		time.Sleep(time.Second * 2)
	}

	time.Sleep(time.Second * 2)

	lvHwnds := listViews(mainHwnd)
	logsCount := getLVItemRowCount(lvHwnds[IDLV_LOGS])

	logrus.WithField("username", p.userName).Debugln("已开始登录")
	// 4. focus on editbox
	if !p.confirmOnLoginWindow(hwndLogin) {
		err = ErrConfirmLoginFailure
		return
	}

	// 5. waiting
	loginFrmDismissed := false
	for i := 0; i < 30; i++ {

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "打开移动证书失败") {
			err = ErrOpenUSBKeyFailure
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "证书密码错") {
			err = ErrWrongUSBKeyPassword
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "登录密码错") {
			err = ErrWrongLoginPassword
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "用户登录名不能为空") {
			err = ErrEmptyUserNameWhileLogin
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "通讯故障") {
			err = ErrNetworkError
			return
		}

		if p.closeMessageBox(mainHwnd, "#32770", "", "确定", "取字段定义表文件失败") {
			err = ErrNetworkError
			return
		}

		oldhwndLogin := w32.FindWindowW(classOfLogin, titleOfLogin)
		if oldhwndLogin == 0 {
			loginFrmDismissed = true
			break
		}

		logrus.WithField("username", p.userName).WithField("HWND", oldhwndLogin).Debugln("登录中......")
		time.Sleep(time.Second)
	}

	if !loginFrmDismissed {
		err = ErrLoginFrmDidNotDismissed
		return
	}

	time.Sleep(time.Second * 2)

	if p.closeMessageBox(mainHwnd, "#32770", "招商银行企业银行直联系统", "确定", "已经登录") {
		logrus.WithField("username", p.userName).Debugln("用户已经登录")
		alreadyLogin = true
		return
	}

	for i := 0; i < 120; i++ {
		logsCountAfter := getLVItemRowCount(lvHwnds[IDLV_LOGS])
		if logsCountAfter > logsCount {
			title := getLVItem(lvHwnds[IDLV_LOGS], 0, 0)
			if title == "错误" {
				message := getLVItem(lvHwnds[IDLV_LOGS], 0, 2)
				time := getLVItem(lvHwnds[IDLV_LOGS], 0, 1)
				logrus.WithField("username", p.userName).WithField("title", title).WithField("time", time).Debugln(message)
				err = ErrLoginFailure
				return
			} else if title == "信息" {
				message := getLVItem(lvHwnds[IDLV_LOGS], 0, 2)
				time := getLVItem(lvHwnds[IDLV_LOGS], 0, 1)
				logrus.WithField("username", p.userName).WithField("title", title).WithField("time", time).Debugln(message)
			}
		}

		if p.IsLoggedIn() {
			return
		}

		logrus.WithField("username", p.userName).Debugln("等待登录列表中显示登录信息......")
		time.Sleep(time.Second)
	}

	err = ErrLoginTimeout

	return
}

func (p *Robot) confirmOnLoginWindow(hwnd w32.HWND) bool {
	confirmd := false
	fnOfEnumLoginChild := func(childHwnd w32.HWND, LPARAM w32.LPARAM) w32.LRESULT {
		className := w32.GetClassNameW(childHwnd)
		title := w32.GetWindowText(childHwnd)

		if "TFBSpeedButton" == className && title == "登录[&L]" {
			confirmd = true
			w32.PostMessage(childHwnd, w32.BM_CLICK, 0, 0)
			return 0
		}
		return 1
	}

	w32.SetForegroundWindow(hwnd)
	w32.EnumChildWindows(hwnd, fnOfEnumLoginChild, 0)

	return confirmd
}

func (p *Robot) getMainProcessPID() int {
	return int(findProcess(p.filename))
}

func (p *Robot) IsListening() bool {
	conn, err := net.Dial("tcp", p.listenAddr)
	if err != nil {
		return false
	}

	conn.Close()

	return true
}
