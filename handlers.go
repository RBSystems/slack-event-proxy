package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/v2/events"
	"github.com/labstack/echo"
)

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
	var ud UserDialog
	// dialog
	var dialog Dialog
	dialog.Title = "Gondor calls for aid!"
	// elements
	var elemOne Element
	var elemTwo Element
	var elemThree Element

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
func HandleSlack(context echo.Context) error {
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
	var sh SlackHelp
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
func CreateAlert(sh SlackHelp) error {
	var helpData HelpData
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
	messenger := GetMessenger()
	messenger.SendEvent(e)

	return nil
}
