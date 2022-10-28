package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chyroc/go-ptr"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("GITHUB_TOKEN is not set")
	}
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	client := github.NewClient(tc)

	err := generateReadme(ctx, client, "chyroc")
	if err != nil {
		panic(err)
	}
}

func generateReadme(ctx context.Context, client *github.Client, userName string) error {
	releases, err := listRelease(ctx, client, userName)
	if err != nil {
		panic(fmt.Errorf("list release: %w", err))
	}
	stars, err := listStar(ctx, client, userName)
	if err != nil {
		panic(fmt.Errorf("list star: %w", err))
	}

	buf := new(strings.Builder)

	buf.WriteString("## Hi 👋, I'm chyroc\n\n")

	buf.WriteString("<table width=\"960px\">\n<tr>\n")
	{
		buf.WriteString("<td valign=\"top\" width=\"50%\">\n\n")
		buf.WriteString("#### Recent Release\n\n")
		for _, v := range releases {
			buf.WriteString(fmt.Sprintf("* <a href='%s' target='_black'>%s</a> - %s\n", v.HtmlURL, v.Name, v.CreatedAt.Format("2006-01-02")))
		}
		buf.WriteString("\n</td>\n")
	}
	{
		buf.WriteString("<td valign=\"top\" width=\"50%\">\n\n")
		buf.WriteString("#### Recent Star\n\n")
		for _, v := range stars {
			buf.WriteString(fmt.Sprintf("* <a href='https://github.com/%s' target='_black'>%s</a> - %s\n", v.FullName, v.FullName, v.CreatedAt.Format("2006-01-02")))
		}
		buf.WriteString("\n</td>\n")
	}

	return ioutil.WriteFile("./README.md", []byte(buf.String()), 0644)
}

func listStar(ctx context.Context, client *github.Client, userName string) ([]*Repo, error) {
	res := []*Repo{}

	data, _, err := client.Activity.ListStarred(ctx, userName, &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 5,
		},
	})
	if err != nil {
		return nil, err
	}
	for _, v := range data {
		res = append(res, &Repo{
			FullName:  *v.Repository.FullName,
			CreatedAt: v.StarredAt.Time,
		})
		if len(res) == 5 {
			return res, nil
		}
	}

	return res, nil
}

func listRelease(ctx context.Context, client *github.Client, userName string) ([]*Release, error) {
	page := 1
	event := []*Release{}
	for {
		events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, userName, true, &github.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		for _, v := range events {
			switch ptr.ValueString(v.Type) {
			case "ReleaseEvent":
				if v.RawPayload != nil {
					body := new(releaseEventBody)
					_ = json.Unmarshal(*v.RawPayload, body)
					if body.Action == "published" {
						event = append(event, body.Release)
						if len(event) == 5 {
							return event, nil
						}
					}
				}
			default:
				//fmt.Println(ptr.ValueString(v.Type))
			}
		}
		if resp.NextPage > page {
			page = resp.NextPage
		} else {
			break
		}
	}

	return event, nil
}

type Release struct {
	HtmlURL   string    `json:"html_url"`
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Repo struct {
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

type releaseEventBody struct {
	Action  string   `json:"action"` // published
	Release *Release `json:"release"`
}
