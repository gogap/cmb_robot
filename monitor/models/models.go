package models

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type ReqBasicInfo struct {
	FUNNAM string `xml:"INFO>FUNNAM"`
	DATTYP int    `xml:"INFO>DATTYP"`
	LGNNAM string `xml:"INFO>LGNNAM"`
}
type RespBasicInfo struct {
	FUNNAM string `xml:"INFO>FUNNAM"`
	DATTYP int    `xml:"INFO>DATTYP"`
	RETCOD int64  `xml:"INFO>RETCOD"` // 检查操作错误. (如: 期望日期错误.)
	ERRMSG string `xml:"INFO>ERRMSG"` // 返回操作错误
}
type Response interface {
	Validate() (err error)
}

// 操作错误, 一般为参数错误导致. 发生该种错误时, 指定业务并没有被执行
type ErrActionFailed struct {
	FUNNAM string
	RETCOD int64
	ERRMSG string
}

func (p ErrActionFailed) Error() string {
	return "CMB Enterprise error: " + p.FUNNAM + "; Return code: " + strconv.FormatInt(p.RETCOD, 10) + "; Error message: " + p.ERRMSG
}

// 验证是否有操作错误.
func (p *RespBasicInfo) Validate() (err error) {
	if p.RETCOD != 0 {
		err = ErrActionFailed{
			FUNNAM: p.FUNNAM,
			RETCOD: p.RETCOD,
			ERRMSG: p.ERRMSG,
		}
		return
	}
	return
}

type PrettyFloat float64

func (f PrettyFloat) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	e.EncodeElement(fmt.Sprintf("%.2f", f), start)
	return
}

// 支付
type ReqDirectPayment struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	ReqBasicInfo
	BUSCOD string      `xml:"SDKPAYRQX>BUSCOD"`
	YURREF string      `xml:"DCOPDPAYX>YURREF"` // System serial number
	DBTACC string      `xml:"DCOPDPAYX>DBTACC"` // 付方账号
	DBTBBK string      `xml:"DCOPDPAYX>DBTBBK"` // 付方开户地区代码
	TRSAMT PrettyFloat `xml:"DCOPDPAYX>TRSAMT"`
	CCYNBR string      `xml:"DCOPDPAYX>CCYNBR"`
	STLCHN string      `xml:"DCOPDPAYX>STLCHN"` // 结算方式代码: N:普通转出/F:快速转出
	NUSAGE string      `xml:"DCOPDPAYX>NUSAGE"`
	BNKFLG string      `xml:"DCOPDPAYX>BNKFLG"`           // 是否招行：Y/N
	CRTACC string      `xml:"DCOPDPAYX>CRTACC"`           // 收款企业转入账号
	CRTNAM string      `xml:"DCOPDPAYX>CRTNAM"`           // 收方账户名
	CRTBNK string      `xml:"DCOPDPAYX>CRTBNK,omitempty"` // 收方开户行（跨行支付必填）
	CRTADR string      `xml:"DCOPDPAYX>CRTADR,omitempty"` // 收方行地址（跨行支付必填）
}
type RespDirectPayment struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	RespBasicInfo
	ERRCOD string `xml:"NTQPAYRQZ>ERRCOD"` // 错误码
	ERRTXT string `xml:"NTQPAYRQZ>ERRTXT"`
	REQNBR string `xml:"NTQPAYRQZ>REQNBR"` // Channel serial number
	REQSTS string `xml:"NTQPAYRQZ>REQSTS"` // 业务请求状态
	RTNFLG string `xml:"NTQPAYRQZ>RTNFLG"` // 业务处理结果
	SQRNBR string `xml:"NTQPAYRQZ>SQRNBR"`
	YURREF string `xml:"NTQPAYRQZ>YURREF"`
}

//支付信息查询
type ReqGetPaymentInfo struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	ReqBasicInfo
	BUSCOD string `xml:"SDKPAYQYX>BUSCOD"`
	BGNDAT string `xml:"SDKPAYQYX>BGNDAT"` // 开始日期
	ENDDAT string `xml:"SDKPAYQYX>ENDDAT"` // 结束日期
	YURREF string `xml:"SDKPAYQYX>YURREF,omitempty"`
}
type RespGetPaymentInfo struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	RespBasicInfo
	NTQPAYQYZ []RespGetPaymentInfoListItem `xml:"NTQPAYQYZ"`
}
type RespGetPaymentInfoListItem struct {
	BUSMOD   string      `xml:"BUSMOD"` // 业务模式
	CRTACC   string      `xml:"CRTACC"`
	CRTADR   string      `xml:"CRTADR"`
	CRTBNK   string      `xml:"CRTBNK"`
	CRTNAM   string      `xml:"CRTNAM"`
	TRSAMT   PrettyFloat `xml:"TRSAMT"`
	BNKFLG   string      `xml:"BNKFLG"`
	STLCHN   string      `xml:"STLCHN"`
	NUSAGE   string      `xml:"NUSAGE"`
	OPRDAT   string      `xml:"OPRDAT"`
	YURREF   string      `xml:"YURREF"`
	REQNBR   string      `xml:"REQNBR"`
	C_REQSTS string      `xml:"C_REQSTS"`
	REQSTS   string      `xml:"REQSTS"`
	C_RTNFLG string      `xml:"C_RTNFLG"`
	RTNFLG   string      `xml:"RTNFLG"`
	RTNNAR   string      `xml:"RTNNAR"` // 支付失败原因/退票原因
	//C_BUSCOD string `xml:"C_BUSCOD"`
	//BUSCOD   string `xml:"BUSCOD"`
	//C_DBTBBK string `xml:"C_DBTBBK"`
	//C_DBTREL string `xml:"C_DBTREL"`
	//DBTBBK   string `xml:"DBTBBK"`
	//DBTBNK   string `xml:"DBTBNK"`
	//DBTACC   string `xml:"DBTACC"`
	//DBTNAM   string `xml:"DBTNAM"`
	//DBTADR   string `xml:"DBTADR"`
	//DBTREL   string `xml:"DBTREL"`
	//C_CRTREL string `xml:"C_CRTREL"`
	//C_CRTBBK string `xml:"C_CRTBBK"`
	//CRTREL   string `xml:"CRTREL"`
	//CRTBBK   string `xml:"CRTBBK"`
	//EPTDAT   string  `xml:"EPTDAT"`
	//EPTTIM   string  `xml:"EPTTIM"`
	//REGFLG   string  `xml:"REGFLG"`
	//C_STLCHN string  `xml:"C_STLCHN"`
	//ATHFLG   string `xml:"ATHFLG"`
	//LGNNAM string `xml:"LGNNAM"`
	//USRNAM string `xml:"USRNAM"`
	//TRSTYP string `xml:"TRSTYP"`
	//FEETYP string `xml:"FEETYP"`
	//RCVTYP string `xml:"RCVTYP"`
	//BUSSTS string `xml:"BUSSTS"`
	//TRSBRN string `xml:"TRSBRN"`
}
type ReqGetBalanceInfo struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	ReqBasicInfo
	SDKACINFX []ReqGetBalanceInfoItem `xml:"SDKACINFX"`
}
type ReqGetBalanceInfoItem struct {
	BBKNBR int    `xml:"BBKNBR"`
	ACCNBR string `xml:"ACCNBR"`
}
type RespGetBalanceInfo struct {
	XMLName xml.Name `xml:"CMBSDKPGK"`
	RespBasicInfo
	NTQACINFZ []RespGetBalanceInfoItem `xml:"NTQACINFZ"`
}
type RespGetBalanceInfoItem struct {
	ACCBLV float64 `xml:"ACCBLV"` // 上日余额
	ONLBLV float64 `xml:"ONLBLV"` // 联机余额
	HLDBLV float64 `xml:"HLDBLV"` // 冻结余额
	AVLBLV float64 `xml:"AVLBLV"` // 可用余额
	//LMTOVR  []string        `xml:"LMTOVR"`
	//BBKNBR  []string        `xml:"BBKNBR"`
	//DPSTXT  []string        `xml:"DPSTXT"`
	//ACCNAM  []string        `xml:"ACCNAM"`
	//STSCOD  []string        `xml:"STSCOD"`
	//MUTDAT  []string        `xml:"MUTDAT"`
	//ACCITM  []string        `xml:"ACCITM"`
	//CCYNBR  []string        `xml:"CCYNBR"`
	//C_CCYNBR        []string        `xml:"C_CCYNBR"`
	//OPNDAT  []string        `xml:"OPNDAT"`
	//INTCOD  []string        `xml:"INTCOD"`
	//ACCNBR  []string        `xml:"ACCNBR"`
	//C_INTRAT        []string        `xml:"C_INTRAT"`
}
type BalanceInfo struct {
	AvailableBalance int64 // 可用余额
	FreezingBalance  int64 // 冻结余额
	OnlineBalance    int64 // 联机余额
	YesterdayBalance int64 // 昨日余额
}
