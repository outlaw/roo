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
  "github.com/satori/go.uuid"
  "github.com/spf13/viper"
)

func appBasedFlags() *[]cli.Flag {
    return &[]cli.Flag{
      cli.StringFlag{
        Name:  "app, a",
        Value: "",
        Usage: "the Application",
      },

      cli.StringFlag{
        Name:  "environment, e",
        Value: "",
        Usage: "the Environment for the Application",
      },
    }
}

func main() {
  app := cli.NewApp()
  app.Name = "roo"
  app.Usage = ""

  viper.SetEnvPrefix("roo")
  viper.BindEnv("lockbox_s3_path")
  viper.BindEnv("env_s3_path")

  viper.SetDefault("lockbox_s3_path", "s3://hooroo-lockbox")
  viper.SetDefault("lockbox_master_key", viper.GetString("env_master_key"))
  viper.SetDefault("env_s3_path", "s3://hooroo-test")
  viper.AutomaticEnv()

  app.Commands = []cli.Command{
  {
// env
    Name:  "env",
    Usage: "control environment variables",
    Flags:  *appBasedFlags(),
    Subcommands: []cli.Command{
      {
        Name:  "set",
        Usage: "[set ENV] - set the environment variable",
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
        Name:  "unset",
        Usage: "[unset ENV] - unset the environment variable",
        Action: func(c *cli.Context) {
          path := c.Args().First()

          manager := envManager()
          if err := manager.Rm(path); err != nil { log.Fatal(err) }
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
// lockbox
  {
    Name:      "lockbox",
    Usage:     "a small secure storage",
    Subcommands: []cli.Command{
      {
        Name:  "store",
        Usage: "store the data or file securely",
        Action: func(c *cli.Context) {
          f := openPath("-", os.Open, os.Stdin)

          path := uuid.NewV4().String()
          defer f.Close()

          manager := lockboxManager()
          if err := manager.Upload(path, f); err != nil { log.Fatal(err) }
          fmt.Printf("%s\n", path)
        },
      },
      {
        Name:  "get",
        Usage: "get the lockbox data",
        Action: func(c *cli.Context) {
          out := openPath("-", os.Create, os.Stdout)
          defer out.Close()

          path := c.Args().First()
          manager := lockboxManager()
          actual, err := manager.Download([]string{path});
          if err != nil { log.Fatal(err) }
          out.Write(actual[path])
        },
      },
    },
  },
  }

  app.Run(os.Args)
}

func lockboxManager() *sneaker.Manager {
  return createManager(viper.GetString("lockbox_s3_path"), viper.GetString("lockbox_master_key"))
}

func envManager() *sneaker.Manager {
  return createManager(viper.GetString("env_s3_path"), viper.GetString("env_master_key"))
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
