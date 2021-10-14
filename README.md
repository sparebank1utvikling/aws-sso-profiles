# aws-sso-profiles

Generate or update `~/.aws/config` with a profile for each SSO account you have access to, by using an existing AWS SSO session.

## Bootstrap

Create a bootstrap aws config file in the current directory:

```ini
[profile bootstrap]
sso_start_url = https://COMPANY.awsapps.com/start
sso_region = us-east-1
sso_account_id =
sso_role_name =
```

```bash
AWS_CONFIG_FILE=$PWD/bootstrap AWS_PROFILE=bootstrap aws sso login
```

```
go run github.com/sparebank1utvikling/aws-sso-profiles@main
```

## Result

Your `~/.aws/config` will now contain all permission sets you have access to via this SSO start URL, in addition to your existing config:

```ini
[profile account1-role1]
sso_start_url  = https://COMPANY.awsapps.com/start
sso_account_id = 123456789012
sso_role_name  = Role1
sso_region     = us-east-1
region         = us-east-1

[profile account2-role2]
sso_start_url  = https://COMPANY.awsapps.com/start
sso_account_id = 123456789012
sso_role_name  = role2
sso_region     = us-east-1
region         = us-east-1

[profile existing]
region         = us-east-1
...
```
