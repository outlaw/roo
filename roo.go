package main

import (
  "os"
  "log"
  "net/url"
  "fmt"
  "strings"
  "github.com/codegangsta/cli"
  "github.com/codahale/sneaker"
  "github.com/aws/aws-sdk-go/service/kms"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
)

func main() {
  app := cli.NewApp()
  app.Name = "roo"
  app.Usage = ""

  // deploy
  // env

  app.Commands = []cli.Command{
  {
    Name:      "env",
    Usage:     "control environment variables",
    Subcommands: []cli.Command{
      {
        Name:  "set",
        Usage: "set the environment variable",
        Action: func(c *cli.Context) {
          f := openPath("-", os.Open, os.Stdin)
          path := c.Args().First()
          defer f.Close()

          manager := envManager()
          if err := manager.Upload(path, f); err != nil {
            log.Fatal(err)
          }
        },
      },
      {
        Name:  "get",
        Usage: "get the environment variable",
        Action: func(c *cli.Context) {
          out := openPath("-", os.Create, os.Stdout)
          defer out.Close()

          path := c.Args().First()
          manager := envManager()
          actual, err := manager.Download([]string{path});
          if err != nil { log.Fatal(err) }
          out.Write(actual[path])
        },
      },
      {
        Name:  "ls",
        Usage: "list the environment variable",
        Action: func(c *cli.Context) {
          manager := envManager()
          files, err := manager.List("*")
          if err != nil { log.Fatal(err) }

          for _, f := range files {
            fmt.Printf("%s\n", f.Path)
          }
        },
      },
    },
  },
  }

  app.Run(os.Args)
}

func secretManager() *sneaker.Manager {
  return createManager(os.Getenv("ROO_SECRET_S3_PATH"), os.Getenv("ROO_SECRET_MASTER_KEY"))
}

func envManager() *sneaker.Manager {
  return createManager(os.Getenv("ROO_ENV_S3_PATH"), os.Getenv("ROO_ENV_MASTER_KEY"))
}

func createManager(s3Url string, keyId string) *sneaker.Manager {
  u, err := url.Parse(s3Url)
  if err != nil { log.Fatalf("bad s3Url: %s", err) }

  ctxt, err := parseContext(os.Getenv("SNEAKER_MASTER_CONTEXT"))
  if err != nil { log.Fatalf("bad SNEAKER_MASTER_CONTEXT: %s", err) }

  session := session.New()
  return &sneaker.Manager{
    Objects: s3.New(session),
    Envelope: sneaker.Envelope{
      KMS: kms.New(session),
    },
    Bucket:            u.Host,
    Prefix:            u.Path,
    EncryptionContext: ctxt,
    KeyId:             keyId,
  }
}

func parseContext(s string) (map[string]string, error) {
  if s == "" {
    return nil, nil
  }

  context := map[string]string{}
  for _, v := range strings.Split(s, ",") {
    parts := strings.SplitN(v, "=", 2)
    if len(parts) != 2 {
      return nil, fmt.Errorf("unable to parse context: %q", v)
    }
    context[parts[0]] = parts[1]
  }
  return context, nil
}

func openPath(file string, o func(string) (*os.File, error), def *os.File) *os.File {
  if file == "-" {
    return def
  }
  f, err := o(file)
  if err != nil {
    log.Fatal(err)
  }
  return f
}
