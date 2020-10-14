package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/niklasfasching/go-org/org"
	"gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	ContentDir string `yaml:"content_dir"`
}

// Post ...
type Post struct {
	Title   string
	Date    string
	URL     string
	Content string
}

const (
	succeedMark = "\u2713"
	failedMark  = "\u2717"

	configFile = "config.yml"

	orgFileExt     = ".org"
	orgTitlePrefix = "#+TITLE:"
	orgDatePrefix  = "#+DATE:"
)

var (
	baseDir string
	conf    Config
)

func init() {
	fmt.Println("> Gobusta Org")

	var err error
	baseDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	fmt.Print("> Read config file...")
	conf, err = readConfig()
	if err != nil {
		panic(err)
	}
	fmt.Printf("\t\t%s\n", succeedMark)
}

func main() {
	fmt.Println("> Start rendering...")

	fmt.Print("1. Collect all posts")
	_, err := collectAllPosts()
	if err != nil {
		panic(err)
		fmt.Printf("\t\t%s\n", failedMark)
	}
	fmt.Printf("\t\t%s\n", succeedMark)

	// fmt.Println(posts)
}

func collectAllPosts() ([]Post, error) {
	var posts []Post
	// Read files in content directory
	var files []string
	dir := join(baseDir, conf.ContentDir)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, orgFileExt) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return posts, err
	}

	// Parse content into Post
	var c []byte
	var post Post
	for _, file := range files {
		c, err = ioutil.ReadFile(file)
		if err != nil {
			break
		}
		post, err = parsePost(string(c))
		if err != nil {
			break
		}
		post.URL = extractURL(file)
		posts = append(posts, post)
	}

	return posts, err
}

func parsePost(c string) (Post, error) {
	post := Post{}

	lines := strings.Split(c, "\n")
	if len(lines) < 2 ||
		!strings.HasPrefix(lines[0], orgTitlePrefix) ||
		!strings.HasPrefix(lines[1], orgDatePrefix) {
		return post, errors.New("Invalid post content")
	}
	post.Title = strings.Trim(lines[0][len(orgTitlePrefix):], " ")
	post.Date = strings.Trim(lines[1][len(orgDatePrefix):], " <>")
	html, err := convertOrgToHTML(strings.Join(lines[2:], "\n"))
	if err != nil {
		return post, err
	}
	post.Content = html
	return post, nil
}

func extractURL(path string) string {
	parts := strings.Split(path, string(os.PathSeparator))
	fileName := parts[len(parts)-1]
	return fileName[:len(fileName)-len(orgFileExt)]
}

func convertOrgToHTML(c string) (string, error) {
	writer := org.NewHTMLWriter()
	orgConf := org.New()
	return orgConf.Parse(bytes.NewReader([]byte(c)), "").Write(writer)
}

func readConfig() (Config, error) {
	var conf Config
	yamlContent, err := ioutil.ReadFile(join(baseDir, configFile))
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(yamlContent, &conf)
	return conf, err
}

func join(paths ...string) string {
	return strings.Join(paths, string(os.PathSeparator))
}
