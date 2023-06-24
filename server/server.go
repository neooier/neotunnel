package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"neotunnel/util"
	"net"
	"net/http"
	"os"

	mcBot "github.com/Tnze/go-mc/bot"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

var (
	addr   string
	https  bool
	cert   string
	key    string
	dst    string
	header string
)

func init() {
	flag.StringVar(&addr, "a", ":80", "http service address")
	flag.BoolVar(&https, "s", false, "enable https")
	flag.StringVar(&cert, "c", "", "cert file")
	flag.StringVar(&key, "k", "", "private key file")
	flag.StringVar(&dst, "d", "localhost:25564", "destination address")
	flag.StringVar(&header, "header", "X-Real-IP", "the http header key implying the client ip, generally it's X-Real-IP or True-Client-Ip or X-Forwarded-For")
	flag.Parse()
	if https {
		if cert == "" || key == "" {
			println("error: when enabling https, you should provide cert and private key by adding -c xxx and -k yyy in the commandline")
			flag.PrintDefaults()
			os.Exit(0)
		}
	} else {
		//println("It is recommended to enable https to avoid HUGE traffic bill")
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	printConfig()
}

func printConfig() {
	log.Printf("listen address: %s", addr)
	log.Printf("enable https: %t", https)
	if https {
		log.Printf("using cert file: %s", cert)
		log.Printf("using key file: %s", key)
	}
	log.Printf("proxy destination: %s", dst)
	log.Printf("client ip http header key: %s", header)
}

func main() {
	http.HandleFunc("/proxy", proxy)
	http.HandleFunc("/status", status)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Header.Get(header))
		_, _ = w.Write([]byte("hello\n"))
	})
	if https {
		log.Fatalln(http.ListenAndServeTLS(addr, cert, key, nil))
	} else {
		log.Fatalln(http.ListenAndServe(addr, nil))
	}
}
func status(w http.ResponseWriter, r *http.Request) {
	resp, delay, err := mcBot.PingAndList(dst)
	bad := func(e error) {
		resp, _ = json.Marshal(util.HTTPResponse{
			Code: 502,
		})
		w.Write([]byte(resp))
	}
	if err != nil {
		bad(err)
		return
	}
	var s util.PingList
	err = json.Unmarshal(resp, &s)
	if err != nil {
		bad(err)
		return
	}
	resp, _ = json.Marshal(util.HTTPResponse{
		Code: 200,
		Data: util.ServerStatus{
			Name:              s.Version.Name,
			Protocol:          s.Version.Protocol,
			PlayerCount:       s.Players.Max,
			MaxPlayerCount:    s.Players.Max,
			OnlinePlayerCount: s.Players.Online,
			SamplePlayers:     s.Players.Sample,
			Description:       s.Description,
			Favicon:           s.FavIcon,
			Delay:             delay,
		},
	})
	w.Write([]byte(resp))
	return
}

func proxy(w http.ResponseWriter, r *http.Request) {
	log := logrus.WithField("ID", "Proxy")
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("[proxy] from %s", r.Header.Get(header))
	defer c.Close(websocket.StatusInternalError, "")
	conn := websocket.NetConn(context.TODO(), c, websocket.MessageBinary)
	defer conn.Close()
	dial, err := net.Dial("tcp", dst)
	if err != nil {
		log.Println(err)
		return
	}
	defer dial.Close()
	go func() {
		for cnt, err := int64(0), error(nil); err == nil; cnt, err = io.CopyN(conn, dial, 4) {
			util.AddUploadBytes(cnt)
		}
		log.Debug(err)
	}()
	for cnt, err := int64(0), error(nil); err == nil; cnt, err = io.CopyN(dial, conn, 4) {
		util.AddDownloadBytes(cnt)
	}
}
