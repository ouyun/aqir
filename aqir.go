package aqir

import (
	"errors"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/tarm/serial"
)

const (
	com      = "/dev/ttyUSB0"
	baud     = 9600
	rTimeOut = 5 //read time out, in second
	retry    = -1
)

const (
	RESP_STATUS_CODE_OK               = 0x9000 //请求被正确处理
	RESP_STATUS_CODE_SYS_BUSY         = 0x9001 //系统忙
	RESP_STATUS_CODE_INV_PKG          = 0x9002 //无效的包
	RESP_STATUS_CODE_UNK_CMD          = 0x9003 //未知的包命令字
	RESP_STATUS_CODE_ERR_PKG_LEN      = 0x9004 //错误的包长度
	RESP_STATUS_CODE_INV_SN           = 0x9005 //无效的序号
	RESP_STATUS_CODE_FAIL_VERIFY      = 0x9006 //校验失败
	RESP_STATUS_CODE_FAULT_HW         = 0x9007 //硬件故障
	RESP_STATUS_CODE_IL_OP            = 0x9008 //非法操作
	RESP_STATUS_CODE_DOS              = 0x9045 //未登录设备，拒绝服务
	RESP_STATUS_CODE_INV_ARG_LEN      = 0x9056 //参数长度错误
	RESP_STATUS_CODE_EXCEPT_SEC_STORE = 0x9057 //安全存储异常
)

const (
	CMD_SET_PINCODE = 0x03
	CMD_CHK_PINCODE = 0x04
	CMD_BURN_KEYS   = 0x05
	CMD_GET_WALLET  = 0x24
	CMD_SIGN_TX     = 0x45
)

var conf serial.Config
var sp serial.Port
var sn int32 = 0

type ReqDatagram struct {
	ReqGramLen int32 //2000 bytes of maximum
	ReqSN      int32
	ReqCmdId   int8
	ReqArgLen  int32
	ReqArg     []byte
}

type RespDatagram struct {
	RespGramLen    int32
	RespSN         int32
	RespStatusCode int16
	RespCmdId      int8
	RespArgLen     int32
	RespArg        []byte
}

func CalcAQI(ug float32) (int16, error) {
	table := []struct {
		Clow  float32
		Chigh float32
		Ilow  int16
		Ihigh int16
	}{
		{0, 15.4, 0, 50},
		{15.5, 40.4, 51, 100},
		{40.5, 65.4, 101, 150},
		{65.5, 150.4, 151, 200},
		{150.5, 250.4, 201, 300},
		{250.5, 350.4, 301, 400},
		{350.5, 500.4, 401, 500},
	}
	idx := -1
	for k, v := range table {
		if ug >= v.Clow && ug <= v.Chigh {
			idx = k
			break
		}
	}
	if idx == -1 {
		fmt.Println("not in available range")
		return -1, errors.New("not in available range")
	}

	var aqi = float32(table[idx].Ihigh-table[idx].Ilow)/(table[idx].Chigh-table[idx].Clow)*
		(ug-table[idx].Clow) + float32(table[idx].Ilow)

	return int16(aqi), nil
}

func FetchAQI() error {
	log.Printf("starting up...\n")
	//c := &serial.Config{Name: "/dev/ttyS0", Baud: 115200}
	//c := &serial.Config{Name: "/dev/ttyS0", Baud: 115200, ReadTimeout: time.Second * 5}
	//c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 9600, ReadTimeout: time.Second * 5, Size: 16}
	c := &serial.Config{Name: com, Baud: baud, ReadTimeout: time.Second * rTimeOut}
	log.Printf("after fetching config\n")
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Printf("opening port successfully\n")
	/*
		n, err := s.Write([]byte("test"))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("writing port successfully\n")
	*/

	tick := time.Tick(1 * time.Second)
	for {
		//fmt.Println(countdown)
		select {
		case <-tick:
			buf := make([]byte, 10)
			n, err := s.Read(buf)
			if err != nil {
				log.Fatal(err)
				return err
			}
			log.Printf("reading port successfully with length of [%d]\n", n)
			log.Printf("%#v\n", buf[:n])
			pm25 := (int32(buf[3])*256 + int32(buf[2])) / 10
			pm10 := (int32(buf[5])*256 + int32(buf[4])) / 10
			aqi, err := CalcAQI(float32(pm25))
			if err != nil {
				log.Printf("pm2.5: [%d]ug/m3, pm10: [%d]ug/m3\n", pm25, pm10)
			} else {
				log.Printf("pm2.5: [%d]ug/m3-AQi[%d], pm10: [%d]ug/m3\n", pm25, aqi, pm10)
			}
		}
	}

	return nil
}

func SetPinCode(code string) (bool, error) { //new or upgrade key
	var req ReqDatagram
	sn++
	req.ReqSN = sn
	req.ReqCmdId = CMD_SET_PINCODE
	req.ReqArgLen = int32(len(req.ReqArg))
	req.ReqArg = []byte(code)
	req.ReqGramLen = int32(unsafe.Sizeof(req.ReqGramLen)) + int32(unsafe.Sizeof(req.ReqCmdId)) +
		int32(unsafe.Sizeof(req.ReqSN)) + int32(unsafe.Sizeof(req.ReqArgLen)) + req.ReqArgLen
	ret, err := executeCommand(&req)
	if err != nil {
		log.Printf("failed to execute a command to secure module with error[%v]\n", err)
		return false, err
	}

	log.Printf("resp[%v]\n", ret)

	return true, nil
}

func CheckPinCode() (bool, error) {

	return true, nil
}

func BurnKeypair() (bool, error) {

	return true, nil
}

func GetWalletPK() {

}

func SignTx() {

}

func executeCommand(req *ReqDatagram) (RespDatagram, error) {

	_ = openCom(retry)
	_ = writeCom([]byte{})
	ret, err := readCom()
	if err != nil {
		log.Printf("failed to read serial port from secure module[%v]\n", err)
		return RespDatagram{}, err
	}
	ok := validateRespGram(ret)
	if !ok {
		log.Printf("failed to validate returned arguments[%v]\n", err)
		return RespDatagram{}, err
	}

	return *ret, nil
}

func openCom(retry int) error {
	log.Printf("starting up serial port from nas box...\n")
	conf = serial.Config{Name: com, Baud: baud, ReadTimeout: time.Second * rTimeOut}
	log.Printf("after fetching config\n")

	var p *serial.Port
	var err error

	if retry < 0 {
		p, err = serial.OpenPort(&conf)
		for err != nil { //infinite loop
			log.Println(err)
			time.Sleep(time.Second * 1)
			p, err = serial.OpenPort(&conf)
		}
		sp = *p
		return nil
	}

	p, err = serial.OpenPort(&conf)
	for ; retry > 0; retry-- {
		log.Println(err)
		time.Sleep(time.Second * 1)
		p, err = serial.OpenPort(&conf)
	}
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println("opening port successfully")
	sp = *p
	return nil
}

func writeCom(cmd []byte) error {
	if &sp == nil {
		err := errors.New("port should not be nil")
		return err
	}

	n, err := sp.Write(cmd)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("writing [%d] bytes to port successfully\n", n)

	return nil
}

func readCom() (*RespDatagram, error) {
	if &sp == nil {
		err := errors.New("port should not be nil")
		return nil, err
	}

	return &RespDatagram{}, nil
}

func validateRespGram(ret *RespDatagram) bool {
	//total length
	//min := 4 + 4 + 2 + 1 + 4
	min := unsafe.Sizeof(ret.RespSN) + unsafe.Sizeof(ret.RespArgLen) + unsafe.Sizeof(ret.RespStatusCode) +
		unsafe.Sizeof(ret.RespCmdId) + unsafe.Sizeof(ret.RespGramLen)
	real := int32(unsafe.Sizeof(ret.RespSN)) + int32(unsafe.Sizeof(ret.RespArgLen)) + int32(unsafe.Sizeof(ret.RespStatusCode)) +
		int32(unsafe.Sizeof(ret.RespCmdId)) + int32(unsafe.Sizeof(ret.RespGramLen)) + ret.RespArgLen
	if real < int32(min) {
		log.Printf("real len[%d] should be greater than min len[%d]", real, min)
		return false
	}
	if ret.RespGramLen != real {
		log.Printf("real len[%d] should be equal to it's len field[%d]", real, ret.RespGramLen)
		return false
	}
	//length of returned arguments

	return true
}
