package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"gopkg.in/ini.v1"
)

func main() {
	session, err := findSSOSession()
	if err != nil {
		log.Fatal(err)
	}

	profiles, err := listSSOProfiles(session)
	if err != nil {
		log.Fatal(err)
	}

	configFile := os.Getenv("AWS_CONFIG_FILE")
	if configFile == "" {
		configFile = os.ExpandEnv("$HOME/.aws/config")
	}
	var in io.Reader
	in, err = os.Open(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		in = bytes.NewBufferString("")
	}

	if err := mergeProfiles(in, configFile, profiles); err != nil {
		log.Fatal(err)
	}
}

type CacheFile struct {
	StartURL    string    `json:"startUrl"`
	Region      string    `json:"region"`
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func findSSOSession() (CacheFile, error) {
	logbuf := &bytes.Buffer{}
	verbose := log.New(logbuf, "", log.LstdFlags)

	dir := os.ExpandEnv("$HOME/.aws/sso/cache")
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println(err)
	}
	var best CacheFile
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			verbose.Printf("readfile %v: %v", file.Name(), err)
			continue
		}
		var cacheFile CacheFile
		if err := json.Unmarshal(data, &cacheFile); err != nil {
			verbose.Printf("parse json %v: %v", file.Name(), err)
			continue
		}

		verbose.Printf("%v: %v (%d)", file.Name(), cacheFile.ExpiresAt, len(cacheFile.AccessToken))
		if cacheFile.ExpiresAt.After(best.ExpiresAt) && cacheFile.AccessToken != "" {
			best = cacheFile
		}
	}

	if best.AccessToken == "" {
		verbose.Println("no tokens")

		return CacheFile{}, errors.New(logbuf.String())
	}
	return best, nil
}

func listSSOProfiles(ssoSession CacheFile) ([]Profile, error) {
	ctx := context.Background()

	cfg := aws.NewConfig()
	cfg.Region = ssoSession.Region

	ssosvc := sso.NewFromConfig(*cfg)
	accounts, err := ssosvc.ListAccounts(ctx, &sso.ListAccountsInput{
		AccessToken: &ssoSession.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, acc := range accounts.AccountList {
		roles, err := ssosvc.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
			AccessToken: &ssoSession.AccessToken,
			AccountId:   acc.AccountId,
		})
		if err != nil {
			return nil, err
		}
		for _, role := range roles.RoleList {
			profiles = append(profiles, Profile{
				Name:         profileName(*acc.AccountName, *role.RoleName),
				SSORoleName:  *role.RoleName,
				SSOStartURL:  ssoSession.StartURL,
				SSORegion:    ssoSession.Region,
				SSOAccountID: *role.AccountId,
				Region:       ssoSession.Region,
			})
		}
	}
	return profiles, nil
}

func profileName(accountName, roleName string) string {
	combined := fmt.Sprintf("%s-%s", accountName, roleName)
	return strings.ToLower(regexp.MustCompile("[^a-zA-Z0-9-]").ReplaceAllString(combined, "-"))
}

type Profile struct {
	Name         string
	SSOStartURL  string
	SSORoleName  string
	SSORegion    string
	SSOAccountID string
	Region       string
	// output = json
	// cli_pager=
}

func mergeProfiles(in io.Reader, outFile string, profiles []Profile) error {
	cfg, err := ini.Load(in)
	if err != nil {
		return err
	}

	updateProfiles(cfg, profiles)
	return cfg.SaveTo(outFile)
}

func updateProfiles(cfg *ini.File, profiles []Profile) {
	for _, profile := range profiles {
		sectionName := "profile " + profile.Name

		cfg.Section(sectionName).Key("sso_start_url").SetValue(profile.SSOStartURL)
		cfg.Section(sectionName).Key("sso_account_id").SetValue(profile.SSOAccountID)
		cfg.Section(sectionName).Key("sso_role_name").SetValue(profile.SSORoleName)
		cfg.Section(sectionName).Key("sso_region").SetValue(profile.SSORegion)

		cfg.Section(sectionName).Key("region").SetValue(profile.Region)
	}
}
