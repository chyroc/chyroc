package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chyroc/go-ptr"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"os"
	"time"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("GITHUB_TOKEN is not set")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	releases, err := listRelease(ctx, client, "chyroc")
	if err != nil {
		panic(fmt.Errorf("list release: %w", err))
	}
	for _, v := range releases {
		fmt.Println(v.Name, v.HtmlURL, v.CreatedAt)
	}

	stars, err := listStar(ctx, client, "chyroc")
	if err != nil {
		panic(fmt.Errorf("list star: %w", err))
	}
	for _, v := range stars {
		fmt.Println(v.FullName, v.CreatedAt)
	}
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
