package main

const (
	API_AUTH              = "https://api-auth.domru.ru/v1/person/auth"
	API_CAMERAS           = "https://myhome.novotelecom.ru/rest/v1/forpost/cameras"
	API_OPEN_DOOR         = "https://myhome.novotelecom.ru/rest/v1/places/%d/accesscontrols/%d/actions"
	API_FINANCES          = "https://myhome.novotelecom.ru/rest/v1/subscribers/profiles/finances"
	API_SUBSCRIBER_PLACES = "https://myhome.novotelecom.ru/rest/v1/subscriberplaces"
	API_VIDEO_SNAPSHOT    = "https://myhome.novotelecom.ru/rest/v1/places/%d/accesscontrols/%d/videosnapshots"
	API_CAMERA_GET_STREAM = "https://myhome.novotelecom.ru/rest/v1/forpost/cameras/%d/video?&LightStream=0"
	API_REFRESH_SESSION   = "https://myhome.novotelecom.ru/auth/v2/session/refresh"
)
