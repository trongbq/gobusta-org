package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/niklasfasching/go-org/org"
	"gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	ContentDir string `yaml:"content"`
	OutputDir  string `yaml:"output"`
	Template   struct {
		Dir   string `yaml:"directory"`
		Index string `yaml:"index"`
		Post  string `yaml:"post"`
	} `yaml:"template"`
	OutputPostDir string `yaml:"output_post"`
	StaticDir     string `yaml:"static"`
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
	posts, err := collectAllPosts()
	if err != nil {
		fmt.Printf("\t\t%s\n", failedMark)
		panic(err)
	}
	fmt.Printf("\t\t%s\n", succeedMark)

	fmt.Print("2. Clean output directory")
	err = cleanOutputDir()
	if err != nil {
		fmt.Printf("\t\t%s\n", failedMark)
		panic(err)
	}
	fmt.Printf("\t\t%s\n", succeedMark)

	fmt.Print("3. Rendering template")
	err = render(posts)
	if err != nil {
		fmt.Printf("\t\t%s\n", failedMark)
		panic(err)
	}
	fmt.Printf("\t\t%s\n", succeedMark)

	fmt.Print("4. Copy static to out directory")
	err = copyDir(join(baseDir, conf.StaticDir), join(baseDir, conf.OutputDir, conf.StaticDir))
	if err != nil {
		fmt.Printf("\t\t%s\n", failedMark)
		panic(err)
	}
	fmt.Printf("\t\t%s\n", succeedMark)

	fmt.Println("> Done rendering...")
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
		post.URL = extractPostURL(file)
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

func render(posts []Post) error {
	err := renderIndexTemplate(posts)
	if err != nil {
		return err
	}
	return renderPostTemplate(posts)
}

func renderIndexTemplate(posts []Post) error {
	templateContent, err := readTemplate(join(baseDir, conf.Template.Dir, conf.Template.Index))
	if err != nil {
		return err
	}

	t, err := template.New("index").Parse(templateContent)
	if err != nil {
		return err
	}

	f, err := os.Create(join(baseDir, conf.OutputDir, conf.Template.Index))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	return t.Execute(f, posts)
}

func renderPostTemplate(posts []Post) error {
	templateContent, err := readTemplate(join(baseDir, conf.Template.Dir, conf.Template.Post))
	if err != nil {
		panic(err)
	}

	t, err := template.New("single-post").Parse(templateContent)
	if err != nil {
		panic(err)
	}

	outPostDir := join(baseDir, conf.OutputDir, conf.OutputPostDir)
	err = os.Mkdir(outPostDir, 0755)
	if err != nil {
		return err
	}

	for _, post := range posts {
		f, err := os.Create(join(baseDir, conf.OutputDir, post.URL))
		if err != nil {
			return err
		}
		defer f.Close()

		err = t.Execute(f, post)
		if err != nil {
			return err
		}
	}
	return nil
}

func readTemplate(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func extractPostURL(path string) string {
	parts := strings.Split(path, string(os.PathSeparator))
	fileName := parts[len(parts)-1]
	return conf.OutputPostDir + "/" + fileName[:len(fileName)-len(orgFileExt)] + ".html"
}

func convertOrgToHTML(c string) (string, error) {
	writer := org.NewHTMLWriter()
	orgConf := org.New()
	return orgConf.Parse(bytes.NewReader([]byte(c)), "").Write(writer)
}

func normalize(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
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

func copyDir(src, dest string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	_, err = os.Stat(dest)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return fmt.Errorf("destination dir already exists: %v", dest)
	}

	err = os.Mkdir(dest, srcStat.Mode())
	if err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if path == src {
			return nil
		}

		if info.IsDir() {
			err := copyDir(path, join(dest, info.Name()))
			if err != nil {
				return err
			}
		} else {
			err := copyFile(path, join(dest, info.Name()))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = in.Close()
	if err != nil {
		return err
	}
	return out.Close()
}

func cleanOutputDir() error {
	out := join(baseDir, conf.OutputDir)

	d, err := os.Stat(out)
	// Check if dir exists, then clean it
	if err == nil {
		if d.IsDir() {
			// Clean output content, exclude `.git` folder
			infos, err := ioutil.ReadDir(out)
			if err != nil {
				return err
			}
			for _, info := range infos {
				os.RemoveAll(join(out, info.Name()))
			}
			return nil
		} else {
			// If `out` is not a dir, then simply delete it
			err := os.Remove(out)
			if err != nil {
				return err
			}
		}
	}
	return os.Mkdir(out, 0755)
}
