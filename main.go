package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

type CacheFile struct {
	StartURL    string    `json:"startUrl"`
	Region      string    `json:"region"`
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func main() {
	dir := os.ExpandEnv("$HOME/.aws/sso/cache")
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println(err)
	}

	logbuf := &bytes.Buffer{}
	verbose := log.New(logbuf, "", log.LstdFlags)

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
		io.Copy(os.Stderr, logbuf)
		os.Exit(1)
	}

	ctx := context.Background()
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(best.Region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ssosvc := sso.NewFromConfig(cfg)
	accounts, err := ssosvc.ListAccounts(ctx, &sso.ListAccountsInput{
		AccessToken: &best.AccessToken,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, acc := range accounts.AccountList {
		roles, err := ssosvc.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
			AccessToken: &best.AccessToken,
			AccountId:   acc.AccountId,
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, role := range roles.RoleList {
			fmt.Printf("%v %v %v\n", *acc.AccountId, *acc.AccountName, *role.RoleName)
		}
	}
}
