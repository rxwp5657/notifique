package main

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/notifique/internal"
	"github.com/notifique/routes"
)

var recorder *httptest.ResponseRecorder

func TestMain(m *testing.M) {
	r := gin.Default()

	storage := internal.MakeInMemoryStorage()
	routes.SetupNotificationRoutes(r, &storage)

	recorder = httptest.NewRecorder()

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestGetUserNoticications(t *testing.T) {

}

func TestGetUserConfig(t *testing.T) {

}

func TestCreateNotification(t *testing.T) {

}

func TestOptIn(t *testing.T) {

}

func TestOptOut(t *testing.T) {

}

func TestSetReadStatus(t *testing.T) {

}

func TestCreateNotificationWithBadChannel(t *testing.T) {

}

func TestCreateNotificationWithDuplicatedChannels(t *testing.T) {

}

func TestCreateNotificationWithLongTopic(t *testing.T) {

}

func TestCreateNotificationWithLongTitle(t *testing.T) {

}

func TestCreateNotificationWithLongContents(t *testing.T) {

}

func TestCreateNotificationWithBadImageURLFormat(t *testing.T) {

}
