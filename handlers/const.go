package handlers

const (
	API_USER_AGENT   = "myHomeErth/8 CFNetwork/1209 Darwin/20.2.0"
	CLIENT_USERAGENT = "iPhone11,6 | iOS 14.3 | erth | 6.4.6 (build 3) | %s | 2 | %s"

	API_AUTH_LOGIN            = "https://myhome.novotelecom.ru/auth/v2/login/%s"
	API_AUTH_CONFIRMATION     = "https://myhome.novotelecom.ru/auth/v2/confirmation/%s"
	API_AUTH_CONFIRMATION_SMS = "https://myhome.novotelecom.ru/auth/v2/auth/%s/confirmation"

	API_AUTH = "https://api-auth.domru.ru/v1/person/auth"

	API_CAMERAS           = "https://myhome.novotelecom.ru/rest/v1/forpost/cameras"
	API_OPEN_DOOR         = "https://myhome.novotelecom.ru/rest/v1/places/%s/accesscontrols/%s/actions"
	API_FINANCES          = "https://myhome.novotelecom.ru/rest/v1/subscribers/profiles/finances"
	API_SUBSCRIBER_PLACES = "https://myhome.novotelecom.ru/rest/v1/subscriberplaces"
	API_VIDEO_SNAPSHOT    = "https://myhome.novotelecom.ru/rest/v1/places/%s/accesscontrols/%s/videosnapshots"
	API_CAMERA_GET_STREAM = "https://myhome.novotelecom.ru/rest/v1/forpost/cameras/%s/video?&LightStream=0"
	API_REFRESH_SESSION   = "https://myhome.novotelecom.ru/auth/v2/session/refresh"
	API_EVENTS            = "https://myhome.novotelecom.ru/rest/v1/places/%s/events?allowExtentedActions=true"
	API_OPERATORS         = "https://myhome.novotelecom.ru/public/v1/operators"
)
