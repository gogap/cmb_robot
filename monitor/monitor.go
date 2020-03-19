package monitor

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-akka/configuration"
	"github.com/gogap/cmb_robot/monitor/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	GBKEncoder transform.Transformer = simplifiedchinese.GBK.NewEncoder()
	GBKDecoder transform.Transformer = simplifiedchinese.GBK.NewDecoder()
)

var (
	ErrBadTXCount        = errors.New("bad tx resp count")
	ErrBadRespTXStatus   = errors.New("bad cmb response tx status")
	ErrBadRespTXAmount   = errors.New("bad response tx amount")
	ErrMonitorURLIsEmpty = errors.New("monitor url is empty")
	ErrUsernameIsEmpty   = errors.New("username is empty")
	ErrSystemSNIsEmpty   = errors.New("system sn is empty")
	ErrChannelSNIsEmpty  = errors.New("channel sn is empty")
	ErrAmountIsZero      = errors.New("excpet amount could not be zero")
	ErrStatusIsEmpty     = errors.New("status is empty")
	ErrDateIsEmpty       = errors.New("date is empty")
)

const CMBRespErrCodeSuc = "SUC0000"

var httpCli http.Client = http.Client{
	Timeout: 29 * time.Second,
}

type CMBMonitor struct {
	url       string
	username  string
	systemSN  string
	channelSN string
	amount    int64
	status    string
	date      string

	networkCheckedTimes int64
}

func NewCMBMonitor(conf *configuration.Config) (mon *CMBMonitor, err error) {

	url := conf.GetString("url")
	if len(url) == 0 {
		err = ErrMonitorURLIsEmpty
		return
	}

	username := conf.GetString("username")
	if len(username) == 0 {
		err = ErrUsernameIsEmpty
		return
	}

	systemSN := conf.GetString("system-sn")
	if len(systemSN) == 0 {
		err = ErrChannelSNIsEmpty
		return
	}

	channelSN := conf.GetString("channel-sn")
	if len(channelSN) == 0 {
		err = ErrChannelSNIsEmpty
		return
	}

	amount := conf.GetInt64("amount")
	if amount == 0 {
		err = ErrAmountIsZero
		return
	}

	status := conf.GetString("status")
	if len(status) == 0 {
		err = ErrStatusIsEmpty
		return
	}

	date := conf.GetString("date")
	if len(date) == 0 {
		err = ErrDateIsEmpty
		return
	}

	mon = &CMBMonitor{
		url:       url,
		username:  username,
		systemSN:  systemSN,
		channelSN: channelSN,
		amount:    amount,
		status:    status,
		date:      date,
	}

	return mon, nil
}
func (p *CMBMonitor) request(req interface{}, resp models.Response) (reqStr, respStr string, err error) {
	reqBytes, err := xml.Marshal(req)
	if err != nil {
		return
	}
	reqBytes = append([]byte(`<?xml version="1.0" encoding = "GBK"?>`), reqBytes...)
	//logrus.WithField("username",p.username).Debugln("request: ", string(reqBytes))
	reqStr, _, err = transform.String(GBKEncoder, string(reqBytes))
	if err != nil {
		return
	}

	reqReader := bytes.NewReader([]byte(reqStr))
	rawResp, err := httpCli.Post(p.url, "application/json", reqReader)
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}
	defer rawResp.Body.Close()
	respBody, err := ioutil.ReadAll(rawResp.Body)
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}
	respStr, _, err = transform.String(GBKDecoder, string(respBody))
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}

	respStr = strings.Replace(respStr, "GBK", "UTF-8", 1)
	//logrus.WithField("username", p.username).Debugln("response xml: ", respStr)
	respBytes := []byte(respStr)
	err = xml.Unmarshal(respBytes, resp)
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}
	//logrus.WithField("username",p.username).Debugf("CMB Enterprise Response: %+v", resp)
	err = resp.Validate()
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}
	return
}

func (p *CMBMonitor) Ping() (err error) {

	err = p.pingNetwork()

	if err != nil {
		return
	}

	if p.networkCheckedTimes < 5 {
		return
	}

	p.networkCheckedTimes = 0

	lastSN := "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

	req := &models.ReqDirectPayment{
		ReqBasicInfo: models.ReqBasicInfo{
			FUNNAM: "DCPAYMNT",
			DATTYP: 2,
			LGNNAM: p.username,
		},

		BUSCOD: "N02031",

		YURREF: lastSN,
		DBTACC: "0000000000000000",
		DBTBBK: "92",
		CCYNBR: "10",
		NUSAGE: "机器人出金测试",
		BNKFLG: "Y",

		STLCHN: "N",
		CRTBNK: "招商银行",
		TRSAMT: 0,
		CRTACC: "0000000000000000",
		CRTNAM: "",
	}

	resp := models.RespGetPaymentInfo{}
	_, _, err = p.request(req, &resp)

	if err != nil {
		if strings.Contains(err.Error(), "签名错误，请检查证书卡是否正确插入") {
			return
		}
		err = nil
	}

	return nil
}

func (p *CMBMonitor) pingNetwork() (err error) {

	p.networkCheckedTimes++

	req := &models.ReqGetPaymentInfo{
		ReqBasicInfo: models.ReqBasicInfo{
			FUNNAM: "GetPaymentInfo",
			DATTYP: 2,
			LGNNAM: p.username,
		},
		BUSCOD: "N02031",
		BGNDAT: p.date,
		ENDDAT: p.date,
		YURREF: p.systemSN,
	}

	resp := models.RespGetPaymentInfo{}
	_, _, err = p.request(req, &resp)
	if err != nil {
		// logrus.WithField("username", p.username).Errorln(err)
		return
	}

	if len(resp.NTQPAYQYZ) != 1 {
		err = ErrBadTXCount
		logrus.WithField("username", p.username).WithField("count", len(resp.NTQPAYQYZ)).Errorln("与期望的返回数据量不匹配")
		return
	}

	item := resp.NTQPAYQYZ[0]

	if int64(item.TRSAMT*100) != p.amount {
		err = ErrBadRespTXAmount
		logrus.WithField("username", p.username).WithField("response_amount", item.TRSAMT).WithField("expect", p.amount).Errorln("与期望的交易金额大小不对")
		return
	}

	if item.REQNBR != p.channelSN {
		err = ErrBadRespTXStatus
		logrus.WithField("username", p.username).WithField("response_sn", item.REQNBR).WithField("expect", p.channelSN).Errorln("与期望的渠道流水号不对")
		return
	}

	if item.RTNFLG != p.status {
		err = ErrBadRespTXStatus
		logrus.WithField("username", p.username).WithField("response_status", item.RTNFLG).WithField("expect", p.status).Errorln("与期望的交易状态不对")
		return
	}

	return nil
}
