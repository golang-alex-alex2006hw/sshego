package sshego

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"
)

func (cfg *SshegoConfig) TcpClientUserAdd(user *User) (toptPath, qrPath, rsaPath string, err error) {

	if cfg.SshegoSystemMutexPort < 0 {
		err = fmt.Errorf("SshegoSystemMutexPort was negative(%v),"+
			" not possible to add user", cfg.SshegoSystemMutexPort)
		return
	}

	// send newUserCmd followed by the msgp marshalled user
	sendMe, err := user.MarshalMsg(nil)
	panicOn(err)

	addr := fmt.Sprintf("127.0.0.1:%v", cfg.SshegoSystemMutexPort)
	nConn, err := net.Dial("tcp", addr)
	panicOn(err)

	deadline := time.Now().Add(time.Second * 10)
	err = nConn.SetDeadline(deadline)
	panicOn(err)

	_, err = nConn.Write(NewUserCmd)
	panicOn(err)

	_, err = nConn.Write(sendMe)
	panicOn(err)

	// read response
	deadline = time.Now().Add(time.Second * 10)
	err = nConn.SetDeadline(deadline)
	panicOn(err)

	dat, err := ioutil.ReadAll(nConn)
	panicOn(err)

	n := len(NewUserReply)
	if len(dat) < n {
		panic(fmt.Errorf("expected '%s' preamble, but got '%s' of length %v", NewUserReply, string(dat), len(dat)))
	}
	p("dat = '%v'", string(dat))
	payload := dat[n:]

	var r User // returned User
	_, err = r.UnmarshalMsg(payload)
	panicOn(err)

	err = nConn.Close()
	panicOn(err)

	return r.TOTPpath, r.QrPath, r.PrivateKeyPath, nil
}

func (cfg *SshegoConfig) TcpClientUserDel(user *User) error {

	if cfg.SshegoSystemMutexPort < 0 {
		err := fmt.Errorf("SshegoSystemMutexPort was negative(%v),"+
			" not possible to delete user", cfg.SshegoSystemMutexPort)
		return err
	}

	// send newUserCmd followed by the msgp marshalled user
	sendMe, err := user.MarshalMsg(nil)
	panicOn(err)

	addr := fmt.Sprintf("127.0.0.1:%v", cfg.SshegoSystemMutexPort)
	nConn, err := net.Dial("tcp", addr)
	panicOn(err)

	deadline := time.Now().Add(time.Second * 10)
	err = nConn.SetDeadline(deadline)
	panicOn(err)

	_, err = nConn.Write(DelUserCmd)
	panicOn(err)

	_, err = nConn.Write(sendMe)
	panicOn(err)

	// read response
	deadline = time.Now().Add(time.Second * 10)
	err = nConn.SetDeadline(deadline)
	panicOn(err)

	dat, err := ioutil.ReadAll(nConn)
	panicOn(err)

	n := len(DelUserReplyFailed)
	if len(dat) < n {
		panic(fmt.Errorf("expected '%s' preamble, but got '%s' of length %v", NewUserReply, string(dat), len(dat)))
	}
	p("dat = '%v'", string(dat))

	err = nConn.Close()
	panicOn(err)

	if string(dat[:n]) == string(DelUserReplyFailed) {
		return fmt.Errorf("user delete failed -- typically because user does not exist")
	}

	return nil
}
