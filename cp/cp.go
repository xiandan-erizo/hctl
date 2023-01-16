package cpFilePod2Pod

/*
https://www.51cto.com/article/644345.html
总结: 1.不带缓冲的通道需要先读后写 2.websocket ReadMessage方法是阻塞读取的, 如果要中断读取, 关闭连接, 捕获错误即可
*/
import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 定义过滤器回调函数
type filterCallback func(input string) bool

// WsConn 带有互斥锁的Websocket连接对象
type WsConn struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

// Send 发送字符串, 自动添加换行符
func (self *WsConn) Send(sender string, str string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	// 利用k8s exec websocket接口发送数据时, 第一个字节需要设置为0, 表示将数据发送到标准输入
	data := []byte{0}
	data = append(data, []byte(str+"\n")...)
	err := self.Conn.WriteMessage(websocket.BinaryMessage, data) //发送二进制数据类型
	if err != nil {
		log.Printf("发送错误, %s", err.Error())
	}
	log.Printf("%s, 数据:%s, 字节:%+v", sender, str, []byte(str+"\n"))
}

// SendWithFilter 发送字符串, 不添加换行符, 内部做字节过滤,等操作
func (self *WsConn) SendWithFilter(sender string, str string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	// log.Printf("向目的容器发送数据:%s", str)
	str = strings.ReplaceAll(str, "\r\n", "\n") // /r=13, /n=10, windows换行符转Linux换行符
	//去掉第一个字节(标准输出1, byte:[0 1 ...]), 因为从源容器输出的字节中, 第一位标识了标准输出1, 给目的容器发送字节时, 需要去除该标志
	//当WebSocket建立连接后，发送数据时需要在字节Buffer第一个字节设置为stdin(buf[0] = 0)，而接受数据时, 需要判断第一个字节, stdout(buf[0] = 1)或stderr(buf[0] = 2)
	strByte := append([]byte(str)[:0], []byte(str)[1:]...)
	data := []byte{0}
	data = append(data, strByte...)
	err := self.Conn.WriteMessage(websocket.BinaryMessage, data)
	log.Printf("向目的容器标准输入发送数据:\n%s, 字节数:%d, 字节:%+v", string(data), len(data), data)
	if err != nil {
		log.Printf("发送错误, %s", err.Error())
	}
}

// Receive 从连接中获取数据流, 并写入字节数组通道中, 内部执行过滤器(回调函数)
func (self *WsConn) Receive(receiver string, ch chan []byte, filter filterCallback) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	msgType, msgByte, err := self.Conn.ReadMessage() //阻塞读取, 类型为2表示二进制数据, 1表示文本, -1表示连接已关闭:websocket: close 1000 (normal)
	log.Printf("%s, 读取到数据:%s, 类型:%d, 字节数:%d, 字节:%+v", receiver, string(msgByte), msgType, len(msgByte), msgByte)
	if err != nil {
		log.Printf("%s, 读取出错, %s", receiver, err.Error())
		return err
	}
	if filter(string(msgByte)) && len(msgByte) > 1 {
		ch <- msgByte
	} else {
		log.Printf("%s, 数据不满足, 直接丢弃数据, 字符:%s, 字节数:%d, 字节:%v", receiver, string(msgByte), len(msgByte), msgByte)
	}
	return nil
}

func NewWsConn(host string, path string, params map[string]string, headers map[string][]string) (*websocket.Conn, error) {
	paramArray := []string{}
	for k, v := range params {
		paramArray = append(paramArray, fmt.Sprintf("%s=%s", k, v))
	}
	u := url.URL{Scheme: "wss", Host: host, Path: path, RawQuery: strings.Join(paramArray, "&")}
	log.Printf("API:%s", u.String())
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{RootCAs: nil, InsecureSkipVerify: true}}
	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("连接错误:%s", err.Error()))
	}
	return conn, nil
}

// CpPod2Pod 核心: tar -cf - 将具有文件夹结构的数据转换成数据流, 通过 tar -xf - 将数据流转换成 linux 文件系统
func CpPod2Pod(container string, host string, token string) {
	//通知主函数可以退出的信号通道
	signalExit := make(chan bool, 1)
	defer close(signalExit)

	//下发不要给目的容器发送数据的信号
	signalStopDstSend := make(chan bool, 1)
	defer close(signalStopDstSend)

	//下发不要从源容器读取数据的信号
	signalStopSrcRead := make(chan bool, 1)
	defer close(signalStopSrcRead)

	//下发不要从目的容器读取数据的信号
	signalStopDstRead := make(chan bool, 1)
	defer close(signalStopDstRead)

	//下发不要打印目的容器的输出数据
	signalStopPrintDstStdout := make(chan bool, 1)
	defer close(signalStopPrintDstStdout)

	//连接pod并执行命令

	headers := map[string][]string{"authorization": {fmt.Sprintf("Bearer %s", token)}}

	pathSrc := "/api/v1/namespaces/xxx/pods/xxx/exec"
	commandSrc := "tar&command=czf&command=-&command=/var/log/mysql/slow.log" //tar czf - sourceFile
	paraSrc := map[string]string{"stdout": "1", "stdin": "0", "stderr": "1", "tty": "0", "container": container, "command": commandSrc}
	srcConn, err := NewWsConn(host, pathSrc, paraSrc, headers)
	if err != nil {
		log.Printf("源Pod连接出错, %s", err.Error())
	}

	pathDst := "/api/v1/namespaces/xxx/pods/xxx/exec"
	commandDst := "tar&command=xzf&command=-&command=-C&command=/tmp" // tar xzf - -C /tmp
	// paraDst := map[string]string{"stdout": "1", "stdin": "1", "stderr": "1", "tty": "0", "container": "xxx", "command": commandDst}
	paraDst := map[string]string{"stdout": "0", "stdin": "1", "stderr": "0", "tty": "0", "container": container, "command": commandDst} //关闭目的Pod标准输出和错误输出
	dstConn, err := NewWsConn(host, pathDst, paraDst, headers)
	if err != nil {
		log.Printf("目的Pod连接出错, %s", err.Error())
	}

	wsSrc := WsConn{
		Conn: srcConn,
	}

	wsDst := WsConn{
		Conn: dstConn,
	}

	defer srcConn.Close()
	defer dstConn.Close()

	srcStdOutCh := make(chan []byte, 2048)
	dstStdOutCh := make(chan []byte)
	defer close(srcStdOutCh)
	defer close(dstStdOutCh)

	// 接收源容器标准输出到数据通道中
	go func() {
		i := 1
		for {
			log.Printf("第%d次, 从源容器读取标准输出", i)
			i++
			//定义匿名过滤器回调方法, 对源容器标准输出中不需要的数据进行过滤
			err := wsSrc.Receive("源容器", srcStdOutCh, func(input string) bool {
				if input == "cat /var/log/mysql/slow.log" {
					return false
					// } else if match, _ := regexp.MatchString("root@(.+)#", input); match {
					//   return false
					// } else if match, _ := regexp.MatchString("cat /(.+).log", input); match {
					//   return false
					// } else if match, _ := regexp.MatchString("cat /tmp/(.+)", input); match {
					//   return false
				} else if match, _ := regexp.MatchString("tar: Removing leading(.+)", input); match {
					return false
				} else if len(input) == 0 { //过滤空消息
					// log.Printf("读取到标准错误输出")
					return false
				}
				return true
			})
			if err != nil {
				log.Printf("读取源容器标准输出失败")
				// signalExit <- true
				break
			}
			// time.Sleep(time.Microsecond * 100)
		}
	}()

	/* 注意, 这里不能开启并发协程去读取目的容器的标准输出, 如果开启可能会与发送数据的协程抢锁, 从而阻塞向目的容器发送数据*/
	// // 从目的容器获取标准输出到数据通道中
	// go func() {
	//   // i := 0
	//   for {
	//     // 该过滤器直接返回true, 仅占位
	//     err := wsDst.Receive("目的容器", dstStdOutCh, func(input string) bool {
	//       return true
	//     })
	//     if err != nil {
	//       log.Printf("从目的容器读取数据失败")
	//       break
	//     }
	//     // wsDst.Send()
	//     time.Sleep(time.Microsecond * 100000)
	//   }
	//   // log.Printf("从目的容器读取数据, 第%d次循环", i)
	//   // i++
	// }()

	// //从数据通道中读取, 目的容器的标准输出, 并打印
	// go func() {
	// BreakPrintDstPodStdout:
	//   for {
	//     select {
	//     case data := <-dstStdOutCh:
	//       log.Printf("目的容器标准输出:%s", string(data))
	//       // time.Sleep(time.Microsecond * 200)
	//     case <-signalStopPrintDstStdout:
	//       log.Printf("收到信号, 停止打印目的容器标准输出")
	//       // close(dataOutput)
	//       // close(dataCh)
	//       // signalStopRead <- true
	//       // log.Printf("发送停止读信号")
	//       // close(dataOutput)
	//       // close(dataCh)
	//       break BreakPrintDstPodStdout
	//     }
	//     // time.Sleep(time.Microsecond * 100)
	//   }
	// }()

	//从源容器标准输出的数据通道获取数据, 然后发送给目的容器标准输入
	//定义超时时间
	timeOutSecond := 3
	timer := time.NewTimer(time.Second * time.Duration(timeOutSecond))
Break2Main:
	for {
		select {
		case data := <-srcStdOutCh:
			wsDst.SendWithFilter("向目的容器发送", string(data))
			// time.Sleep(time.Millisecond * 200)
			timer.Reset(time.Second * time.Duration(timeOutSecond))
		case <-timer.C:
			// time.Sleep(time.Second * 5)
			log.Printf("================ 源容器标准输出,没有新的数据,获取超时,停止向目的容器发送数据 ================")
			// log.Printf("发送信号:停止打印目的容器标准输出")
			// signalStopPrintDstStdout <- true
			log.Printf("发送信号:停止从源容器读取数据")
			wsSrc.Conn.Close()
			// log.Printf("发送信号:停止从目的容器读取数据")
			// wsDst.Conn.Close()
			log.Printf("发送信号:主函数可以退出了")
			signalExit <- true
			log.Printf("所有信号发送完毕")
			log.Printf("================== 跳出循环 =================")
			break Break2Main
		}
		// time.Sleep(time.Microsecond * 1000)
	}

	// signalStopRead <- true
	<-signalExit //阻塞通道, 直到收到一个信号
	// signalStopRead <- true
	log.Printf("主函数收到信号, 准备退出")
	// close(dataCh)
	// time.Sleep(time.Second)
	// close(dataOutput)
	// time.Sleep(time.Second)
	// select {}
}
