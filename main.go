package main

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type BiRes struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		Group            string  `json:"group"`
		BusinessId       int     `json:"business_id"`
		RefreshRowFactor float64 `json:"refresh_row_factor"`
		RefreshRate      int     `json:"refresh_rate"`
		MaxDelay         int     `json:"max_delay"`
		Token            string  `json:"token"`
		HostList         []struct {
			Host    string `json:"host"`
			Port    int    `json:"port"`
			WssPort int    `json:"wss_port"`
			WsPort  int    `json:"ws_port"`
		} `json:"host_list"`
	} `json:"data"`
}

type BiliReq struct {
	Uid      int    `json:"uid"`
	Roomid   int    `json:"roomid"`
	Protover int    `json:"protover"`
	Platform string `json:"platform"`
	Type     int    `json:"type"`
	Key      string `json:"key"`
}

func main() {
	// 获取ws链接的http get请求
	var bili BiRes
	res, err := http.Get("https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?id=22310900&type=0")
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	json.Unmarshal(b, &bili)
	// 获取ws链接的http get请求  end

	// 拼凑 我们需要的ws链接
	wsHost := bili.Data.HostList[len(bili.Data.HostList)-1].Host
	wsPort := strconv.Itoa(bili.Data.HostList[len(bili.Data.HostList)-1].WsPort)
	wsUrl := url.URL{Host: wsHost + ":" + wsPort, Scheme: "ws", Path: "sub"}

	// 拼凑 ws参数
	req := BiliReq{
		Uid:      322210472,
		Roomid:   22310900,
		Protover: 1,
		Platform: "web",
		Type:     2,
		Key:      bili.Data.Token,
	}
	// https://blog.csdn.net/yyznm/article/details/116543107  参考
	// 参数转码为byte
	dataByte, _ := json.Marshal(req)
	// 计算 byte长度 （+16 bilibili规定的头部长度）
	dataByteLen := len(dataByte) + 16 // 计算的封包总长度
	//(十六进制)
	//%08x   封包总大小
	//0010   头部长度
	//0001   协议版本，目前是1
	//00000007   操作码（封包类型）
	//00000001  sequence，可以取常数1
	handshake := fmt.Sprintf("%08x001000010000000700000001", dataByteLen)
	buf := make([]byte, len(handshake)>>1)
	hex.Decode(buf, []byte(handshake))
	// 把数字串 转为 []byte

	//创建 bilibili ws链接
	conn, res, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		log.Println(err)
		return
	}

	// 封包头拼接body
	err = conn.WriteMessage(websocket.BinaryMessage, append(buf, dataByte...))
	for {
		// 等待信息返回
		_, message, _ := conn.ReadMessage()
		fmt.Println(message)
		// 对第八位 解码规则判断 0 为正常可见的字符串 2 为需要zlib解码的关键信息
		if message[7] == 2 {
			// zlib解码
			b := bytes.NewReader(message[16:])
			r, _ := zlib.NewReader(b)
			bs, _ := io.ReadAll(r)
			log.Printf("收到弹幕: %s", string(bs))
		}
	}
}
