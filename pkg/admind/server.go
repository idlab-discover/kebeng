package admind

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/idlab-discover/kebeng/pkg/repositories"

	"github.com/gin-gonic/gin"
	"github.com/idlab-discover/kebeng/config"
	"github.com/idlab-discover/kebeng/config/configkey"
	"github.com/idlab-discover/kebeng/pkg/admind/requests"
	"github.com/idlab-discover/kebeng/pkg/database"
	"github.com/idlab-discover/kebeng/pkg/middleware"
	"github.com/idlab-discover/kebeng/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Server struct {
	db     *gorm.DB
	engine *gin.Engine
	snaps  *repositories.SnapsRepository
}

func (s *Server) Init() {
	logrus.SetLevel(logrus.TraceLevel)
	config.LoadConfig()

	logLevelConfig := viper.GetString(configkey.LogLevel)
	l, errLevel := logrus.ParseLevel(logLevelConfig)
	if errLevel != nil {
		logrus.Error(errLevel)
	} else {
		logrus.SetLevel(l)
	}

	// Setup gin and routes
	r := gin.Default()
	if viper.GetBool(configkey.DebugMode) {
		logrus.Info("Debug mode enabled")
		r.Use(middleware.RequestLoggerMiddleware())
	} else {
		logrus.Info("Debug mode disabled")
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "KEBE STORE: PAGE_NOT_FOUND", "message": "Page not found"})
	})

	db, _ := database.CreateDatabase()
	s.db = db
	s.snaps = repositories.NewSnapsRepository(db)

	s.SetupEndpoints(r)
}

func (s *Server) Run() {
	admindPort := viper.GetInt(configkey.AdminDPort)
	_ = s.engine.Run(fmt.Sprintf(":%d", admindPort))
}

func (s *Server) addAccount(c *gin.Context) {
	var addAccountReq requests.AddAccount
	err := json.NewDecoder(c.Request.Body).Decode(&addAccountReq)
	if err == nil {

		// TODO:: add validation
		account := models.Account{
			AccountId:   addAccountReq.AcccountId,
			DisplayName: addAccountReq.DisplayName,
			Username:    addAccountReq.Username,
			Email:       addAccountReq.Email,
		}

		s.db.Save(&account)

		c.Status(http.StatusCreated)
		return
	}

	logrus.Error(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Server) addTrack(c *gin.Context) {
	var addTrackReq requests.AddTrack
	err := json.NewDecoder(c.Request.Body).Decode(&addTrackReq)
	if err == nil {

		logrus.Tracef("requests.AddTrack: %+v", addTrackReq)

		var snapEntry models.SnapEntry
		db := s.db.Where(&models.SnapEntry{Name: addTrackReq.SnapName}).Find(&snapEntry)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			track := models.SnapTrack{
				Name:        addTrackReq.TrackName,
				SnapEntryID: snapEntry.ID,
			}

			s.db.Save(&track)

			s.snaps.AddDefaultRisks(snapEntry, track.ID, uint64(0)) // FIX: hardcoded value for now (0) -> developing upload functionality

			c.Status(http.StatusCreated)
			return
		}
	}
	logrus.Error(err)
	c.AbortWithStatus(http.StatusInternalServerError)
}
