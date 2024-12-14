package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"os"
	"strings"
	"text/template"
	"unicode"
)

// setting
const (
	templateFile = `README.tmpl`
	readme       = `README.md`
)

type RepositoryInfo struct {
	Name           string
	NameWithOwner  string
	Description    string
	Url            string
	StargazerCount int
	ForkCount      int
	UpdatedAt      string
	CreatedAt      string
	PushedAt       string
	IsArchived     bool
	Languages      []string
}

func executeTemplateToStr(tmpl string, data any) (string, error) {
	t := template.New("local_template")
	parsedTmpl, err := t.Parse(tmpl)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = parsedTmpl.Execute(buf, data)
	return buf.String(), err
}

func getUserStaredRepositories(githubToken string) ([]RepositoryInfo, error) {
	stars := make([]RepositoryInfo, 0)
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)
	var query struct {
		Viewer struct {
			StarredRepositories struct {
				IsOverLimit bool
				TotalCount  int
				Edges       []struct {
					StarredAt string
					Cursor    string
					Node      struct {
						Name           string
						NameWithOwner  string
						Description    string
						Url            string
						StargazerCount int
						ForkCount      int
						UpdatedAt      string
						CreatedAt      string
						PushedAt       string
						IsArchived     bool
						Languages      struct {
							TotalCount int
							Nodes      []struct {
								Name string
							}
						} `graphql:"languages(first: 3)"`
					}
				}
			} `graphql:"starredRepositories(first: $count, after: $cursor)"`
		}
	}
	variables := map[string]interface{}{
		"count":  githubv4.Int(100),
		"cursor": githubv4.String(""),
	}
	// Initial value
	totalRepositories := 100
	totalFound := 0
	currentFound := 0
	cursor := ""
	for totalFound < totalRepositories {
		err := client.Query(context.Background(), &query, variables)
		if err != nil {
			return stars, err
		}
		// For next loop
		totalRepositories = query.Viewer.StarredRepositories.TotalCount
		currentFound = len(query.Viewer.StarredRepositories.Edges)
		totalFound += currentFound
		cursor = query.Viewer.StarredRepositories.Edges[currentFound-1].Cursor
		variables["count"] = min(githubv4.Int(totalRepositories-currentFound), githubv4.Int(100))
		variables["cursor"] = githubv4.String(cursor)
		// Storage repository info
		for _, edge := range query.Viewer.StarredRepositories.Edges {
			// Remove emoji symbol in description.
			edge.Node.Description = strings.TrimFunc(edge.Node.Description, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r) //&& !unicode.IsSpace(r) && !unicode.IsPunct(r)
			})
			languages := make([]string, 0)
			for _, node := range edge.Node.Languages.Nodes {
				languages = append(languages, node.Name)
			}
			stars = append(stars, RepositoryInfo{
				Name:           edge.Node.Name,
				NameWithOwner:  edge.Node.NameWithOwner,
				Description:    edge.Node.Description,
				Url:            edge.Node.Url,
				StargazerCount: edge.Node.StargazerCount,
				ForkCount:      edge.Node.ForkCount,
				UpdatedAt:      edge.Node.UpdatedAt,
				CreatedAt:      edge.Node.CreatedAt,
				PushedAt:       edge.Node.PushedAt,
				IsArchived:     edge.Node.IsArchived,
				Languages:      languages,
			})
		}
	}

	return stars, nil
}

func main() {

	// Step1: Check configuration
	fmt.Print("Step1 - Check congfiguration: ")
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("$GITHUB_TOKEN environment variable not set.")
		return
	}
	
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("OK.")

	// Step2: Query stars
	fmt.Print("Step2 - Get user starred repositories information via Github api: ")
	repos, err := getUserStaredRepositories(token)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("OK.")

	// Step3: Render template
	fmt.Print("Step3 - Render template file with data: ")
	data, err := executeTemplateToStr(string(templateBytes), repos)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("OK.")

	// Step4: Write to README
	fmt.Print("Step4 - Write to README file: ")
	if err := os.WriteFile(readme, []byte(data), 0644); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("OK.")
	fmt.Println("Finished!")	
}