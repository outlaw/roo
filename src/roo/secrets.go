package main

import (
  "os"
  "io"
  "log"
  "net/url"
  "fmt"
  "github.com/codegangsta/cli"
  "github.com/codahale/sneaker"
  "github.com/aws/aws-sdk-go/service/kms"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
  "github.com/satori/go.uuid"
  "github.com/spf13/viper"
)

var cmdLockbox = cli.Command {
  Name:      "lockbox",
  Usage:     "Store files in secure storage for up to 5 days",
  Subcommands: []cli.Command{
    {
      Name:  "store",
      Usage: "[store FILE] Store the data or file securely in a lockbox. This results in an ID for use with 'get'",
      Action: func(c *cli.Context) {
        file := c.Args().First()
        f    := openPath(file, os.Open, os.Stdin)

        path := uuid.NewV4().String()
        defer f.Close()

        spinner := gSpinner(" Storing File", "\r✔ Stored file \n")

        manager := lockboxManager()
        err     := manager.Upload(path, f)
        spinner.Stop()

        if err != nil { log.Fatal(err) }
        fmt.Printf("%s", path)
      },
    },
    {
      Name:  "get",
      Usage: "[get ID] Retrieve the lockbox data",
      Action: func(c *cli.Context) {
        out := openPath("-", os.Create, os.Stdout)
        defer out.Close()

        path        := c.Args().First()
        manager     := lockboxManager()
        actual, err := manager.Download([]string{path});

        if err != nil { log.Fatal(err) }
        out.Write(actual[path])
      },
    },
  },
}

var cmdEnv = cli.Command {
// env
  Name:  "env",
  Usage: "Control environment variables for this app",
  Flags:  *appBasedFlags(),
  Subcommands: []cli.Command{
    {
      Name:  "set",
      Usage: "[set ENV] Set the environment variable for this app",
      Action: func(c *cli.Context) {
        f    := openPath("-", os.Open, os.Stdin)
        path := c.Args().First()
        defer f.Close()

        manager := envManager()
        if err  := manager.Upload(path, f); err != nil { log.Fatal(err) }
      },
    },
    {
      Name:  "unset",
      Usage: "[unset ENV] Unset the environment variable for this app",
      Action: func(c *cli.Context) {
        path := c.Args().First()

        manager := envManager()
        if err  := manager.Rm(path); err != nil { log.Fatal(err) }
      },
    },
    {
      Name:  "get",
      Usage: "[get ENV] Get the environment variable for this app",
      Action: func(c *cli.Context) {
        out := openPath("-", os.Create, os.Stdout)
        defer out.Close()

        path        := c.Args().First()
        manager     := envManager()
        actual, err := manager.Download([]string{path});

        if err != nil { log.Fatal(err) }
        out.Write(actual[path])
      },
    },
    {
      Name:  "ls",
      Usage: "[ls] List the environment variables for this app",
      Action: func(c *cli.Context) {
        manager    := envManager()
        files, err := manager.List("*")
        if err != nil { log.Fatal(err) }

        for _, f := range files {
          fmt.Printf("%s\n", f.Path)
        }
      },
    },
  },
}

type SecretManager interface {
  Rm(string)                  error
  List(string)                ([]sneaker.File, error)
  Download([]string)          (map[string][]byte, error)
  Upload(string, io.Reader)   error
}

// Sneakers
func lockboxManager() SecretManager {
  return createManager(viper.GetString("lockbox_s3_path"), viper.GetString("lockbox_master_key"))
}

func envManager() SecretManager {
  return createManager(viper.GetString("env_s3_path"), viper.GetString("env_master_key"))
}

func createManager(s3Url string, keyId string) SecretManager {
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

