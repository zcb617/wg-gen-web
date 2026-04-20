package server

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vx3r/wg-gen-web/auth"
	"github.com/vx3r/wg-gen-web/core"
	"github.com/vx3r/wg-gen-web/model"
	"github.com/vx3r/wg-gen-web/version"
	"golang.org/x/oauth2"
	"net/http"
)

// ApplyRoutes applies router to gin Router
func ApplyRoutes(r *gin.RouterGroup) {
	g := r.Group("/server")
	{
		g.GET("", readServer)
		g.PATCH("", updateServer)
		g.GET("/config", configServer)
		g.GET("/version", versionStr)
		g.GET("/dnscrypt", dnscryptInfo)
	}
}

func readServer(c *gin.Context) {
	client, err := core.ReadServer()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to read client")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, client)
}

func updateServer(c *gin.Context) {
	var data model.Server

	if err := c.ShouldBindJSON(&data); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to bind")
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	// get update user from token and add to server infos
	oauth2Token := c.MustGet("oauth2Token").(*oauth2.Token)
	oauth2Client := c.MustGet("oauth2Client").(auth.Auth)
	user, err := oauth2Client.UserInfo(oauth2Token)
	if err != nil {
		log.WithFields(log.Fields{
			"oauth2Token": oauth2Token,
			"err":         err,
		}).Error("failed to get user with oauth token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	data.UpdatedBy = user.Name

	server, err := core.UpdateServer(&data)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to update client")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, server)
}

func configServer(c *gin.Context) {
	configData, err := core.ReadWgConfigFile()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to read wg config file")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// return config as txt file
	c.Header("Content-Disposition", "attachment; filename=wg0.conf")
	c.Data(http.StatusOK, "application/config", configData)
}

func versionStr(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version.Version,
	})
}

func dnscryptInfo(c *gin.Context) {
	// Read DNSCrypt provider-info.txt from mounted volume
	infoPath := filepath.Join(os.Getenv("WG_CONF_DIR"), "dnscrypt", "keys", "provider-info.txt")

	if _, err := os.Stat(infoPath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
		})
		return
	}

	file, err := os.Open(infoPath)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("failed to open dnscrypt provider-info")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var stamp string
	var providerName string
	var publicKey string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "DNS Stamp:") {
			stamp = strings.TrimSpace(strings.TrimPrefix(line[strings.Index(line, "DNS Stamp:"):], "DNS Stamp:"))
			// Remove log prefix if present
			if idx := strings.Index(stamp, "sdns://"); idx >= 0 {
				stamp = stamp[idx:]
			}
		}
		if strings.Contains(line, "Provider name:") {
			providerName = strings.TrimSpace(strings.TrimPrefix(line[strings.Index(line, "Provider name:"):], "Provider name:"))
			if idx := strings.Index(providerName, "2.dnscrypt-cert."); idx >= 0 {
				providerName = providerName[idx:]
			}
		}
		if strings.Contains(line, "Provider public key:") {
			publicKey = strings.TrimSpace(strings.TrimPrefix(line[strings.Index(line, "Provider public key:"):], "Provider public key:"))
			if idx := strings.Index(publicKey, " "); idx >= 0 {
				publicKey = publicKey[idx+1:]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{"err": err}).Error("failed to read dnscrypt provider-info")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":      stamp != "",
		"stamp":        stamp,
		"providerName": providerName,
		"publicKey":    publicKey,
	})
}
