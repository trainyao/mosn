package registry

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/healthcheck"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	IntervalDur      time.Duration = 15 * time.Second
	TimeoutDur       time.Duration = 6 * 15 * time.Second
	APPCheckPointURL string        = "http://127.0.0.1:9500/checkService"
)

func StartAppHealthCheck(url string) {
	healthcheck.StartHttpHealthCheck(IntervalDur, TimeoutDur, url, onAppInterval, onTimeout)
}

func onAppInterval(path string, hcResetTimeOut func()) {

	log.DefaultLogger.Debugf("Send Http Get %s", path)
	resp, err := http.Get(path)

	if err != nil {
		// handle error
		log.DefaultLogger.Debugf(" Get Error: %s from path %s",
			err.Error(), path)
		// wait next tick
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		// handle error
		log.DefaultLogger.Debugf(" Read %s %s", path,
			err.Error())

		// wait next tick
	}

	res := string(body)
	//get first line
	idx := strings.Index(res, "\n")
	if idx == -1 {
	}

	if res[:idx] == "passed:true" {
		//shc.handleSuccess()
		log.DefaultLogger.Debugf("APP Health Checks got success")
		hcResetTimeOut()
	} else {
		// wait next tick
	}
}

func onTimeout() {
	log.DefaultLogger.Debugf("timeout")
}
