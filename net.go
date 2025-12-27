package microps

import "github.com/bugph0bia/go-microps/internal/util"

func NetInit() bool {

	util.Infof("initialize...")
	if !platformInit() {
		util.Errorf("platformInit() failuer")
		return false
	}
	util.Infof("success")
	return true
}

func NetRun() bool {

	util.Infof("startup...")
	if !platformRun() {
		util.Errorf("platformRun() failuer")
		return false
	}
	util.Infof("success")
	return true
}

func NetShutdown() bool {

	util.Infof("shutting down...")
	if !platformShutdown() {
		util.Errorf("platformShutdown() failuer")
		return false
	}
	util.Infof("success")
	return true
}
