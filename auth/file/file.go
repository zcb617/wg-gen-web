package file

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vx3r/wg-gen-web/model"
	"golang.org/x/oauth2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File auth provider using local text file
type File struct {
	users map[string]string // username -> password
}

// Setup reads users from /config/users.txt (or WG_USERS_FILE env)
func (f *File) Setup() error {
	f.users = make(map[string]string)

	usersFile := os.Getenv("WG_USERS_FILE")
	if usersFile == "" {
		usersFile = filepath.Join(os.Getenv("WG_CONF_DIR"), "users.txt")
	}

	file, err := os.Open(usersFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Users file not found at %s, creating default admin/admin", usersFile)
			f.users["admin"] = "admin"
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			f.users[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if len(f.users) == 0 {
		log.Warn("No users found in users.txt, creating default admin/admin")
		f.users["admin"] = "admin"
	}

	return scanner.Err()
}

// Validate checks username/password against file
func (f *File) Validate(username, password string) bool {
	if pw, ok := f.users[username]; ok {
		return pw == password
	}
	return false
}

// CodeUrl returns a special marker for file auth
func (f *File) CodeUrl(state string) string {
	return "_magic_string_file_auth_login_form_"
}

// Exchange for file auth: code is "username:password" base64 or plain
func (f *File) Exchange(code string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("file auth uses /auth/login instead of oauth2_exchange")
}

// UserInfo returns user info from token (token contains username)
func (f *File) UserInfo(oauth2Token *oauth2.Token) (*model.User, error) {
	return &model.User{
		Sub:      oauth2Token.AccessToken,
		Name:     oauth2Token.AccessToken,
		Email:    oauth2Token.AccessToken,
		Profile:  "file",
		Issuer:   "file",
		IssuedAt: time.Time{},
	}, nil
}
