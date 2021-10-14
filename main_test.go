package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/ini.v1"
)

func Test_profileName(t *testing.T) {
	type args struct {
		accountName string
		roleName    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "HappyCase", args: args{accountName: "app-account", roleName: "admin-profile"}, want: "app-account/admin-profile"},
		{name: "CapsToLowercase", args: args{accountName: "App-Account", roleName: "aDmin-profile"}, want: "app-account/admin-profile"},
		{name: "ReplaceChars", args: args{accountName: "app account.2", roleName: " admin-profile"}, want: "app-account-2/admin-profile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := profileName(tt.args.accountName, tt.args.roleName); got != tt.want {
				t.Errorf("profileName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateProfiles(t *testing.T) {
	overwriteArgs, err := ini.Load(bytes.NewBufferString(`[profile name]
sso_start_url  = OVERWRITE_ME
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region
`))
	if err != nil {
		t.Fatal(err)
	}

	preserveArgs, err := ini.Load(bytes.NewBufferString(`[profile otherName]
sso_start_url  = PRESERVE_ME
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region
`))
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ini      *ini.File
		profiles []Profile
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Happy",
			args: args{
				ini: ini.Empty(),
				profiles: []Profile{
					{
						Name:         "name",
						SSORoleName:  "sso-role-name",
						SSOStartURL:  "sso-start-url",
						SSORegion:    "sso-region",
						SSOAccountID: "account-id",
						Region:       "region",
					},
				},
			},
			want: `[profile name]
sso_start_url  = sso-start-url
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region
`,
		},
		{
			name: "HappyOverwite",
			args: args{
				ini: overwriteArgs,
				profiles: []Profile{
					{
						Name:         "name",
						SSORoleName:  "sso-role-name",
						SSOStartURL:  "sso-start-url",
						SSORegion:    "sso-region",
						SSOAccountID: "account-id",
						Region:       "region",
					},
				},
			},
			want: `[profile name]
sso_start_url  = sso-start-url
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region
`,
		},
		{
			name: "Preserve",
			args: args{
				ini: preserveArgs,
				profiles: []Profile{
					{
						Name:         "name",
						SSORoleName:  "sso-role-name",
						SSOStartURL:  "sso-start-url",
						SSORegion:    "sso-region",
						SSOAccountID: "account-id",
						Region:       "region",
					},
				},
			},
			want: `[profile otherName]
sso_start_url  = PRESERVE_ME
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region

[profile name]
sso_start_url  = sso-start-url
sso_account_id = account-id
sso_role_name  = sso-role-name
sso_region     = sso-region
region         = region
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateProfiles(tt.args.ini, tt.args.profiles)
			var buf bytes.Buffer
			tt.args.ini.WriteTo(&buf)
			if strings.TrimSpace(tt.want) != strings.TrimSpace(buf.String()) {
				t.Errorf("want: %s\ngot: %s\n", tt.want, buf.String())
				t.Log(cmp.Diff(tt.want, buf.String()))
			}
		})
	}
}
