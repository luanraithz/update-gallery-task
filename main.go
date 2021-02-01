package main

// [START pubsub_subscriber_async_pull]
// [START pubsub_quickstart_subscriber]
import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// gcloud auth application-default login

func main() {
	ch := make(chan bool)
	go pullMsgs(ch)
	<-ch
	fmt.Printf("Runned \n")
}

type picture struct {
	Name string
	URL  string
}

type htmlContent struct {
	Pictures []picture
}

func runHook(scriptName string) error {
	if _, err := os.Stat(scriptName); err == nil {
		cmd := exec.Command("bash", scriptName)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to run %s \n", scriptName)
			return err
		}
		fmt.Printf("Runned %s \n", scriptName)
	} else if os.IsNotExist(err) {
		fmt.Printf("%s doesn't exist \n", scriptName)
	}
	return nil
}

func pullMsgs(ch chan bool) {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("Running on bg on path " + path)
	var project string
	flag.StringVar(&project, "p", "", "Project to be used")
	var outPath string
	flag.StringVar(&outPath, "out", "./profile/src/components/pictures.html", "Output path of the content")
	ctx := context.Background()
	flag.Parse()
	if project == "" {
		ch <- true
		return
	}

	client, err := pubsub.NewClient(ctx, project)
	if err != nil {
		panic(err)
	}
	sub := client.Subscription("update-pictures-website")
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		fmt.Printf("Got message starting to update: %s\n", time.Now().String())
		sclient, err := storage.NewClient(ctx)
		if err != nil {
			fmt.Printf("storage.NewClient: %v", err)
		}
		bucket := "pictures-luan-raithz"
		b := sclient.Bucket(bucket)
		objs := b.Objects(ctx, nil)
		pictures := []picture{}
		for {
			attrs, err := objs.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				panic(err)
			}
			pictures = append(pictures, picture{Name: attrs.Name, URL: fmt.Sprintf("https://storage.googleapis.com/%s/%s", attrs.Bucket, attrs.Name)})
		}

		tmpl := `
{{range .Pictures}}
  <a class="img-wrapper" data-url="{{ .URL }}" title="{{ .Name }}">
    <img src={{ .URL }}></img>
  </a>
{{end}}`
		t, err := template.New("webpage").Parse(tmpl)
		text := strings.Builder{}

		t.Execute(&text, htmlContent{Pictures: pictures})
		if err := runHook("pre.sh"); err != nil {
			fmt.Println(err.Error())
			ch <- true
			return
		}
		f, err := os.Create(outPath)
		f.WriteString(text.String())
		if err := runHook("post.sh"); err != nil {
			fmt.Println(err.Error())
			ch <- true
			return
		}
		msg.Ack()
	})
	fmt.Printf(err.Error())
	if err != nil {
		panic(err)
	}
}
