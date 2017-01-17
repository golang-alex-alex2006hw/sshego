package sshego

import (
	"fmt"
	"net"
	"os"
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

// ud_test.go: unix domain socket test.

func Test401UnixDomainSocketListening(t *testing.T) {

	cv.Convey("Instead of -listen and -remote only forwarding via connections, if given a path instead of a port it should listen on a unix domain socket.", t, func() {

		// generate a random payload for the client to send to the server.
		payloadByteCount := 50
		confirmationPayload := RandomString(payloadByteCount)
		confirmationReply := RandomString(payloadByteCount)

		serverDone := make(chan bool)

		udpath := startBackgroundTestUnixDomainServer(
			serverDone,
			payloadByteCount,
			confirmationPayload,
			confirmationReply)
		defer os.Remove(udpath)

		s := makeTestSshClientAndServer()
		defer TempDirCleanup(s.srvCfg.origdir, s.srvCfg.tempdir)

		//		dest := fmt.Sprintf("127.0.0.1:%v", udpath)
		dest := udpath

		// below over SSH should be equivalent of the following
		// non-encrypted ping/pong.

		if false {
			udUnencPingPong(udpath, confirmationPayload, confirmationReply, payloadByteCount)
		}
		if true {
			dc := DialConfig{
				ClientKnownHostsPath: s.cliCfg.ClientKnownHostsPath,
				Mylogin:              s.mylogin,
				RsaPath:              s.rsaPath,
				TotpUrl:              s.totp,
				Pw:                   s.pw,
				Sshdhost:             s.srvCfg.EmbeddedSSHd.Host,
				Sshdport:             s.srvCfg.EmbeddedSSHd.Port,
				DownstreamHostPort:   dest,
				TofuAddIfNotKnown:    true,
			}

			// first time we add the server key
			channelToTcpServer, _, err := dc.Dial()
			cv.So(err.Error(), cv.ShouldContainSubstring, "Re-run without -new")

			// second time we connect based on that server key
			dc.TofuAddIfNotKnown = false
			channelToTcpServer, _, err = dc.Dial()
			cv.So(err, cv.ShouldBeNil)

			verifyClientServerExchangeAcrossSshd(channelToTcpServer, confirmationPayload, confirmationReply, payloadByteCount)
			channelToTcpServer.Close()
		}
		// tcp-server should have exited because it got the expected
		// message and replied with the agreed upon reply and then exited.
		<-serverDone

		// done with testing, cleanup
		s.srvCfg.Esshd.Stop()
		<-s.srvCfg.Esshd.Done
		cv.So(true, cv.ShouldEqual, true) // we should get here.
	})
}

func udUnencPingPong(dest, confirmationPayload, confirmationReply string, payloadByteCount int) {
	conn, err := net.Dial("unix", dest)
	panicOn(err)
	m, err := conn.Write([]byte(confirmationPayload))
	panicOn(err)
	if m != payloadByteCount {
		panic("too short a write!")
	}

	// check reply
	rep := make([]byte, payloadByteCount)
	m, err = conn.Read(rep)
	panicOn(err)
	if m != payloadByteCount {
		panic("too short a reply!")
	}
	srep := string(rep)
	if srep != confirmationReply {
		panic(fmt.Errorf("saw '%s' but expected '%s'", srep, confirmationReply))
	}
	pp("reply success! server back to -> client: we got the expected srep reply '%s'", srep)
	conn.Close()
}

func startBackgroundTestUnixDomainServer(serverDone chan bool, payloadByteCount int, confirmationPayload string, confirmationReply string) (udpath string) {

	udpath = "/tmp/ud_test.sock." + RandomString(20)
	lsn, err := net.Listen("unix", udpath)
	panicOn(err)

	go func() {
		udServerConn, err := lsn.Accept()
		panicOn(err)
		pp("%v", udServerConn)

		b := make([]byte, payloadByteCount)
		n, err := udServerConn.Read(b)
		panicOn(err)
		if n != payloadByteCount {
			panic(fmt.Errorf("read too short! got %v but expected %v", n, payloadByteCount))
		}
		saw := string(b)

		if saw != confirmationPayload {
			panic(fmt.Errorf("expected '%s', but saw '%s'", confirmationPayload, saw))
		}

		pp("client -> server success! server got expected confirmation payload of '%s'", saw)

		// reply back
		n, err = udServerConn.Write([]byte(confirmationReply))
		panicOn(err)
		if n != payloadByteCount {
			panic(fmt.Errorf("write too short! got %v but expected %v", n, payloadByteCount))
		}
		//udServerConn.Close()
		close(serverDone)
	}()

	return udpath
}
