package main

import (
	"crypto/x509"
	"embed"
	"math/rand"
	"neotunnel/util"
	"net"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

var (
	cdnAddr  string = "opencdn.jomodns.com:443"
	listen   string = "127.0.0.1:25565"
	fakeHost string = "www.babybus.com"
	realHost string = "tst.jorylee.cn"
	DEBUG    string = "true"
)

const letterBytes = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CA from ubuntu 20.04
//
//go:embed RootCAs
var CA embed.FS
var pool *x509.CertPool

var rateLimiter rate.Limiter = *rate.NewLimiter(0.3, 5)

func genRandUUID() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
func loadRootCA() {
	pool = x509.NewCertPool()
	b, err := CA.ReadFile("RootCAs")
	if err != nil {
		logrus.Fatalln(err)
	}
	pool.AppendCertsFromPEM(b)

}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&nested.Formatter{HideKeys: true})
	loadRootCA()
}

func main() {
	/*generate := ReadSourceAndGenerate()
	ip := Tcping(generate)
	println(ip.String())*/

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		logrus.Fatalln(err)
	}
	ticker := time.NewTicker(time.Second * 2)
	go func() {
		for range ticker.C {
			logrus.WithField("ID", "Count").Infof("Upload: %v, Download: %v", util.GetUploadBytes(), util.GetDownloadBytes())
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Warnf("Error when accepting connection: %v", err)
		}
		ID := genRandUUID()
		log := logrus.WithFields(logrus.Fields{"ID": ID})
		//_ = conn.SetReadDeadline(time.Unix(0, 0))
		go handle(conn, log)
	}
	ticker.Stop()

}
