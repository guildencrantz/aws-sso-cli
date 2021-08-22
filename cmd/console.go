package main

/*
 * AWS SSO CLI
 * Copyright (c) 2021 Aaron Turner  <synfinatic at gmail dot com>
 *
 * This program is free software: you can redistribute it
 * and/or modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or with the authors permission any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/atotto/clipboard"
	"github.com/c-bata/go-prompt"
	"github.com/skratchdot/open-golang/open" // default opener
)

const AWS_FEDERATED_URL = "https://signin.aws.amazon.com/federation"

type ConsoleCmd struct {
	Clipboard bool   `kong:"optional,short='c',help='Copy URL to clipboard instead of opening it'"`
	Print     bool   `kong:"optional,short='p',help='Print URL instead of opening it'"`
	Duration  int64  `kong:"optional,short='d',help='AWS Session duration in minutes (default 60)',default=60,env='AWS_SSO_DURATION'"`
	Arn       string `kong:"optional,short='a',help='ARN of role to assume',env='AWS_SSO_ROLE_ARN'"`
	AccountId string `kong:"optional,name='account',short='A',help='AWS AccountID of role to assume',env='AWS_SSO_ACCOUNTID'"`
	Role      string `kong:"optional,short='R',help='Name of AWS Role to assume',env='AWS_SSO_ROLE'"`
}

func (cc *ConsoleCmd) Run(ctx *RunContext) error {
	if ctx.Cli.Console.Arn != "" {
		awssso := doAuth(ctx)
		s := strings.Split(ctx.Cli.Exec.Arn, ":")
		var accountid, role string
		if len(s) == 2 {
			// short account:Role format
			accountid = s[0]
			role = s[1]
		} else {
			// long format for arn:aws:iam:XXXXXXXXXX:role/YYYYYYYY
			accountid = s[3]
			s = strings.Split(s[4], "/")
			role = s[1]
		}
		return openConsole(ctx, awssso, accountid, role)
	} else if ctx.Cli.Exec.AccountId != "" || ctx.Cli.Exec.Role != "" {
		if ctx.Cli.Exec.AccountId == "" || ctx.Cli.Exec.Role == "" {
			return fmt.Errorf("Please specify both --account and --role")
		}
		awssso := doAuth(ctx)
		return openConsole(ctx, awssso, ctx.Cli.Exec.AccountId, ctx.Cli.Exec.Role)
	}

	fmt.Printf("Please use `exit` or `Ctrl-D` to quit.\n")

	// use completer to figure out the role
	sso := ctx.Config.SSO[ctx.Cli.SSO]
	sso.Refresh()
	c := NewTagsCompleter(ctx, sso, openConsole)
	p := prompt.New(
		c.Executor,
		c.Complete,
		prompt.OptionPrefix("> "),
		prompt.OptionSetExitCheckerOnInput(c.ExitChecker),
		prompt.OptionCompletionOnDown(),
		prompt.OptionShowCompletionAtStart(),
	)
	p.Run()
	return nil
}

// opens the AWS console or just prints the URL
func openConsole(ctx *RunContext, awssso *AWSSSO, accountid, role string) error {
	creds, err := awssso.GetRoleCredentials(accountid, role)
	if err != nil {
		log.WithError(err).Fatalf("Unable to get role credentials for %s", role)
	}

	signin := SigninTokenUrlParams{
		SessionDuration: ctx.Cli.Console.Duration * 60,
		Session: SessionUrlParams{
			AccessKeyId:     creds.AccessKeyId,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
		},
	}

	resp, err := http.Get(signin.GetUrl())
	if err != nil {
		return fmt.Errorf("Unable to login to AWS: %s", err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	loginResponse := LoginResponse{}
	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		return fmt.Errorf("Error parsing Login response: %s", err.Error())
	}

	login := LoginUrlParams{
		Issuer:      "https://proofpoint.com",
		Destination: "https://console.aws.amazon.com",
		SigninToken: loginResponse.SigninToken,
	}
	url := login.GetUrl()

	browser := ctx.Config.Browser
	if ctx.Cli.Console.Clipboard {
		err = clipboard.WriteAll(url)
	} else if ctx.Cli.Console.Print {
		fmt.Printf("Please open the following URL in your browser:\n\n%s\n\n",
			url)
	} else {
		if len(browser) == 0 {
			err = open.Run(url)
			browser = "default browser"
		} else {
			err = open.RunWith(url, browser)
		}
		if err != nil {
			err = fmt.Errorf("Unable to open %s with %s: %s", url, browser, err.Error())
		}
	}

	return err
}

type LoginResponse struct {
	SigninToken string `json:"SigninToken"`
}

type SigninTokenUrlParams struct {
	SessionDuration int64
	Session         SessionUrlParams // URL encoded SessionUrlParams
}

func (stup *SigninTokenUrlParams) GetUrl() string {
	return fmt.Sprintf("%s?Action=getSigninToken&SessionDuration=%d&Session=%s",
		AWS_FEDERATED_URL, stup.SessionDuration, stup.Session.Encode())
}

type SessionUrlParams struct {
	AccessKeyId     string `json:"sessionId"`
	SecretAccessKey string `json:"sessionKey"`
	SessionToken    string `json:"sessionToken"`
}

func (sup *SessionUrlParams) Encode() string {
	s, _ := json.Marshal(sup)
	return url.QueryEscape(string(s))
}

type LoginUrlParams struct {
	Issuer      string
	Destination string
	SigninToken string
}

func (lup *LoginUrlParams) GetUrl() string {
	return fmt.Sprintf("%s?Action=login&Issuer=%s&Destination=%s&SigninToken=%s",
		AWS_FEDERATED_URL, lup.Issuer, lup.Destination,
		lup.SigninToken)
}
