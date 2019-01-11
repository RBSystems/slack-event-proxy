package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/byuoitav/central-event-system/hub/base"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/touchpanel-ui-microservice/helpers"
	"github.com/byuoitav/touchpanel-ui-microservice/socket"
	"github.com/labstack/echo"
)

func init() {
	var err *nerr.E
	messenger, err = messenger.BuildMessenger(os.Getenv("HUB_ADDRESS"), base.Messenger, 1000)
	if err != nil {
		log.L.Errorf("unable to build the messenger: %s", err.Error())
	}

	messenger.SubscribeToRooms(deviceInfo.RoomID)

}

var messenger *messenger.Messenger

func GetHostname(context echo.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.JSON(http.StatusOK, hostname)
}

func GetPiHostname(context echo.Context) error {
	hostname := os.Getenv("SYSTEM_ID")
	return context.JSON(http.StatusOK, hostname)
}

func GetDeviceInfo(context echo.Context) error {
	di, err := helpers.GetDeviceInfo()
	if err != nil {
		return context.JSON(http.StatusBadRequest, err.Error())
	}

	return context.JSON(http.StatusOK, di)
}

func Reboot(context echo.Context) error {
	log.L.Warnf("[management] Rebooting pi")
	http.Get("http://localhost:7010/reboot")
	return nil
}

func GetDockerStatus(context echo.Context) error {
	log.L.Warnf("[management] Getting docker status")
	resp, err := http.Get("http://localhost:7010/dockerStatus")
	log.L.Warnf("docker status response: %v", resp)
	if err != nil {
		return context.JSON(http.StatusBadRequest, err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return context.JSON(http.StatusBadRequest, err.Error())
	}

	return context.String(http.StatusOK, string(body))
}

// GenerateHelpFunction generates an echo handler that handles help requests.
func GenerateHelpFunction(value string, messenger *messenger.Messenger) func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		deviceInfo := events.GenerateBasicDeviceInfo(os.Getenv("SYSTEM_ID"))

		// send an event requesting help
		event := events.Event{
			GeneratingSystem: deviceInfo.DeviceID,
			Timestamp:        time.Now(),
			EventTags: []string{
				events.Alert,
			},
			TargetDevice: deviceInfo,
			AffectedRoom: events.GenerateBasicRoomInfo(deviceInfo.RoomID),
			Key:          "help-request",
			Value:        value,
			User:         ctx.RealIP(),
			Data:         nil,
		}

		log.L.Warnf("Sending event to %s help. (event: %+v)", value, event)
		messenger.SendEvent(event)

		return ctx.String(http.StatusOK, fmt.Sprintf("Help has been %sed", value))
	}
}

/*TODO
1.) Make a relevant struct that holds all the info needed
2.) Ask Joe how to upsert the record (assuming things worked well)
3.) Make sure that the records are in proper form and that they can be interpreted into metrics
*/

//Creates the dialog for Slack
func Help(context echo.Context) error {
	log.L.Infof("Starting to help")

	//Authorization
	token := "Bearer " + os.Getenv("SLACK_TOKEN_HELP")
	//The request we are sent comes with a trigger id which we need to send back
	context.Request().ParseForm()
	triggerID := context.Request().Form["trigger_id"][0]

	//How to Open a Slack Dialog
	url := "https://slack.com/api/dialog.open"

	// build json payload
	// Overarching Structure
	var ud helpers.UserDialog
	// dialog
	var dialog helpers.Dialog
	dialog.Title = "Gondor calls for aid!"
	// elements
	var elemOne helpers.Element
	var elemTwo helpers.Element
	var elemThree helpers.Element

	elemOne.Name = "roomID"
	elemOne.Label = "Room"
	elemOne.Type = "text"

	elemTwo.Name = "techName"
	elemTwo.Label = "Technician Name"
	elemTwo.Type = "text"

	elemThree.Name = "notes"
	elemThree.Label = "Notes"
	elemThree.Type = "textarea"

	dialog.Elements = append(dialog.Elements, elemOne)
	dialog.Elements = append(dialog.Elements, elemTwo)
	dialog.Elements = append(dialog.Elements, elemThree)

	dialog.CallbackID = time.Now().Unix()

	//Throw it together
	ud.Dialog = dialog
	ud.TriggerID = triggerID

	//Marshal it
	json, err := json.Marshal(ud)
	if err != nil {
		log.L.Warnf("failed to marshal dialog: %v", ud)
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	//Make the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	req.Header.Set("Content-type", "application/json; charset=utf-8") //Note the charset. If you don't have it they will yell at you
	req.Header.Set("Authorization", token)

	//We don't really care about this response because it has no nutrients! (useful information)
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	return context.String(http.StatusOK, "")
}

//What happens after 'Submit' is hit on the /avcall command
func HandleDialog(context echo.Context) error {
	//Necessary to read the request
	context.Request().ParseForm()
	//Find the payload amid the context
	payload := context.Request().Form["payload"][0]
	//The mess you see below is a regex that we pray doesn't ever break.
	//If all works correctly, it should pull out the values you need without having to ever look at it again
	//Hopefully this gets out the room that needs help and the notes for that room
	r1 := regexp.MustCompile("\"roomID\":\"(.*?)\"")
	r2 := regexp.MustCompile("\"notes\":\"(.*?)\"")
	r3 := regexp.MustCompile("\"callback_id\":\"(.*?)\"")
	r4 := regexp.MustCompile("\"techName\":\"(.*?)\"")

	roomID := r1.FindStringSubmatch(payload)[1]
	notes := r2.FindStringSubmatch(payload)[1]
	id := r3.FindStringSubmatch(payload)[1]
	techName := r4.FindStringSubmatch(payload)[1]

	log.L.Infof("[Follow Up] Trying to find stuff: %v ---------- %v", roomID, notes)
	var sh helpers.SlackHelp
	sh.Building = strings.Split(roomID, "-")[0]
	sh.Room = roomID
	sh.Notes = notes
	sh.CallbackID = id
	sh.TechName = techName
	err := CreateAlert(sh)
	if err != nil {
		log.L.Warnf("Could not create Help Request: %v", err.Error())
		return context.JSON(http.StatusInternalServerError, err.Error())
	}

	return context.String(http.StatusOK, "")
}

//Creates the Alert
func CreateAlert(sh helpers.SlackHelp) error {

	var helpData helpers.HelpData
	helpData.TechName = sh.TechName
	helpData.Notes = sh.Notes
	helpData.Building = sh.Building
	helpData.Room = sh.Room

	e := events.Event{
		GeneratingSystem: sh.Room,
		Timestamp:        time.Now(),
		AffectedRoom:     events.GenerateBasicRoomInfo(sh.Room),
		User:             sh.TechName,
		EventTags:        []string{events.HelpRequest},
		Data:             helpData,
	}
	socket.H.WriteToSockets(e)

	return nil
}
