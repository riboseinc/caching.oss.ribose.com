package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"golang.org/x/oauth2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/google/go-github/github"
)

type EnvConfig struct {
	GitHubAccessToken  string
	GitHubOrganization string
	S3Bucket           string
	S3Key              string
}

type RepositoryInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Forks       int    `json:"forks"`
	Language    string `json:"language"`
	PushedAt    string `json:"pushed_at"`
	URL         string `json:"url"`
}

func Handle(event json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	config, err := loadEnvConfig()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	githubClient := newGitHubClient(config)
	s3Client := newS3Client()

	repos, err := getAllRepositoriesByOrganization(githubClient, config.GitHubOrganization)
	if err != nil {
		log.Println(err)

		return nil, err
	}

	repoInfos := makeRepositoryInfoList(repos)

	err = writeToS3(s3Client, config.S3Bucket, config.S3Key, repoInfos)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.Println(aerr.Error())
		}
		log.Println(err)
		return nil, err
	}

	return nil, nil
}

func loadEnvConfig() (*EnvConfig, error) {
	config := &EnvConfig{}

	if val, ok := os.LookupEnv("GITHUB_ACCESS_TOKEN"); ok {
		config.GitHubAccessToken = val
	} else {
		return nil, errors.New("GITHUB_ACCESS_TOKEN undefined")
	}

	if val, ok := os.LookupEnv("GITHUB_ORGANIZATION"); ok {
		config.GitHubOrganization = val
	} else {
		return nil, errors.New("GITHUB_ORGANIZATION undefined")
	}

	if val, ok := os.LookupEnv("S3_BUCKET"); ok {
		config.S3Bucket = val
	} else {
		return nil, errors.New("S3_BUCKET undefined")
	}

	if val, ok := os.LookupEnv("S3_KEY"); ok {
		config.S3Key = val
	} else {
		return nil, errors.New("S3_KEY undefined")
	}

	return config, nil
}

func newGitHubClient(config *EnvConfig) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GitHubAccessToken},
	)

	return github.NewClient(oauth2.NewClient(ctx, ts))
}

func newS3Client() *s3.S3 {
	session := session.Must(session.NewSession(aws.NewConfig()))
	return s3.New(session)
}

func getAllRepositoriesByOrganization(client *github.Client, org string) ([]*github.Repository, error) {
	ctx := context.Background()
	options := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{Page: 1, PerPage: 100},
	}

	var allRepos []*github.Repository
	for options.Page != 0 {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, options)
		if err != nil {
			return allRepos, err
		}

		allRepos = append(allRepos, repos...)
		options.Page = resp.NextPage
	}

	return allRepos, nil
}

func makeRepositoryInfoList(repos []*github.Repository) []*RepositoryInfo {
	var repoInfos []*RepositoryInfo
	for _, repo := range repos {
		repoInfo := &RepositoryInfo{
			Name:     *repo.Name,
			Stars:    *repo.StargazersCount,
			Forks:    *repo.ForksCount,
			PushedAt: repo.PushedAt.String(),
			URL:      *repo.HTMLURL,
		}

		if repo.Description != nil {
			repoInfo.Description = *repo.Description
		}

		if repo.Language != nil {
			repoInfo.Language = *repo.Language
		}

		repoInfos = append(repoInfos, repoInfo)
	}

	return repoInfos
}

func writeToS3(client *s3.S3, bucket string, key string, info []*RepositoryInfo) error {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return err
	}

	_, err = client.PutObject(&s3.PutObjectInput{
		ACL:    aws.String("public-read"),
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(infoJSON),
	})

	return err
}
