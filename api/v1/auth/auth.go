package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/vx3r/wg-gen-web/auth"
	"github.com/vx3r/wg-gen-web/auth/file"
	"github.com/vx3r/wg-gen-web/model"
	"github.com/vx3r/wg-gen-web/util"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"time"
)

// ApplyRoutes applies router to gin Router
func ApplyRoutes(r *gin.RouterGroup) {
	g := r.Group("/auth")
	{
		g.GET("/oauth2_url", oauth2URL)
		g.POST("/oauth2_exchange", oauth2Exchange)
		g.POST("/login", login)
		g.GET("/user", user)
		g.GET("/logout", logout)
	}
}

/*
 * generate redirect url to get OAuth2 code or let client know that OAuth2 is disabled
 */
func oauth2URL(c *gin.Context) {
	cacheDb := c.MustGet("cache").(*cache.Cache)

	state, err := util.GenerateRandomString(32)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to generate state random string")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	clientId, err := util.GenerateRandomString(32)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to generate state random string")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	// save clientId and state so we can retrieve for verification
	cacheDb.Set(clientId, state, 5*time.Minute)

	oauth2Client := c.MustGet("oauth2Client").(auth.Auth)

	data := &model.Auth{
		Oauth2:   true,
		ClientId: clientId,
		State:    state,
		CodeUrl:  oauth2Client.CodeUrl(state),
	}

	c.JSON(http.StatusOK, data)
}

/*
 * exchange code and get user infos, if OAuth2 is disable just send fake data
 */
func oauth2Exchange(c *gin.Context) {
	var loginVals model.Auth
	if err := c.ShouldBind(&loginVals); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("code and state fields are missing")
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	cacheDb := c.MustGet("cache").(*cache.Cache)
	savedState, exists := cacheDb.Get(loginVals.ClientId)

	if !exists || savedState != loginVals.State {
		log.WithFields(log.Fields{
			"state":      loginVals.State,
			"savedState": savedState,
		}).Error("saved state and client provided state mismatch")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	oauth2Client := c.MustGet("oauth2Client").(auth.Auth)

	oauth2Token, err := oauth2Client.Exchange(loginVals.Code)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to exchange code for token")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cacheDb.Delete(loginVals.ClientId)
	cacheDb.Set(oauth2Token.AccessToken, oauth2Token, cache.DefaultExpiration)

	c.JSON(http.StatusOK, oauth2Token.AccessToken)
}

func login(c *gin.Context) {
	var loginVals struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		ClientId string `json:"clientId"`
		State    string `json:"state"`
	}
	if err := c.ShouldBind(&loginVals); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("username and password fields are missing")
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	cacheDb := c.MustGet("cache").(*cache.Cache)

	// Validate state if provided
	if loginVals.ClientId != "" && loginVals.State != "" {
		savedState, exists := cacheDb.Get(loginVals.ClientId)
		if !exists || savedState != loginVals.State {
			log.WithFields(log.Fields{
				"state":      loginVals.State,
				"savedState": savedState,
			}).Error("saved state and client provided state mismatch")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}

	// Validate credentials using file auth provider
	authProvider := c.MustGet("oauth2Client").(auth.Auth)
	fileProvider, ok := authProvider.(*file.File)
	if !ok {
		log.Error("login endpoint only works with file auth provider")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !fileProvider.Validate(loginVals.Username, loginVals.Password) {
		log.WithFields(log.Fields{
			"username": loginVals.Username,
		}).Error("invalid username or password")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Generate token
	rand, err := util.GenerateRandomString(32)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("failed to generate token")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	oauth2Token := &oauth2.Token{
		AccessToken:  rand,
		TokenType:    "",
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	cacheDb.Delete(loginVals.ClientId)
	cacheDb.Set(oauth2Token.AccessToken, oauth2Token, cache.DefaultExpiration)

	c.JSON(http.StatusOK, oauth2Token.AccessToken)
}

func logout(c *gin.Context) {
	cacheDb := c.MustGet("cache").(*cache.Cache)
	cacheDb.Delete(c.Request.Header.Get(util.AuthTokenHeaderName))
	c.JSON(http.StatusOK, gin.H{})
}

func user(c *gin.Context) {
	cacheDb := c.MustGet("cache").(*cache.Cache)
	oauth2Token, exists := cacheDb.Get(c.Request.Header.Get(util.AuthTokenHeaderName))

	if exists && oauth2Token.(*oauth2.Token).AccessToken == c.Request.Header.Get(util.AuthTokenHeaderName) {
		oauth2Client := c.MustGet("oauth2Client").(auth.Auth)
		user, err := oauth2Client.UserInfo(oauth2Token.(*oauth2.Token))
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("failed to get user from oauth2 AccessToken")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.JSON(http.StatusOK, user)
		return
	}

	log.WithFields(log.Fields{
		"exists":                 exists,
		util.AuthTokenHeaderName: c.Request.Header.Get(util.AuthTokenHeaderName),
	}).Error("oauth2 AccessToken is not recognized")

	c.AbortWithStatus(http.StatusUnauthorized)
}
