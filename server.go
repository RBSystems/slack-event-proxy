package main

import (
	"os"

	"github.com/byuoitav/central-event-system/hub/base"
	"github.com/byuoitav/central-event-system/messenger"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	commonEvents "github.com/byuoitav/common/v2/events"
)

var m *messenger.Messenger
var ne *nerr.E

func main() {
	m = GetMessenger()
	port := ":4444"
	router := common.NewRouter()

	router.POST("/slackhelp", Help)
	router.POST("/handleslack", HandleSlack)

	router.Start(port)
	log.L.Infof("Router has started")
}

func GetMessenger() *messenger.Messenger {
	if m == nil {
		deviceInfo := commonEvents.GenerateBasicDeviceInfo(os.Getenv("SYSTEM_ID"))
		m, ne = messenger.BuildMessenger(os.Getenv("HUB_ADDRESS"), base.Messenger, 1000)
		if ne != nil {
			log.L.Errorf("unable to build the messenger: %s", ne.Error())
		}
		if m == nil {
			log.L.Errorf("The messenger came back... nil")
		}
		log.L.Infof("M: %v", m)
		m.SubscribeToRooms(deviceInfo.RoomID)
	}
	return m
}
