package main

import (
  "runtime"
  "time"
  "os"
  "io"
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
  "github.com/briandowns/spinner"
  "github.com/remind101/empire/cmd/emp/hkclient"
  "github.com/remind101/empire/pkg/heroku"
  "github.com/docker/docker/pkg/jsonmessage"
  "github.com/docker/docker/pkg/term"
)

var (
  flagApp   string
  client    *heroku.Client
  nrc       *hkclient.NetRc
  hkAgent   = "hk/" + "0.0.1" + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
  userAgent = hkAgent + " " + heroku.DefaultUserAgent
  apiURL    = ""

  // Create
  flagRegion string
  flagOrgName string
  flagHTTPGit bool
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
  viper.AutomaticEnv()
  viper.BindEnv("lockbox_s3_path")
  viper.BindEnv("env_s3_path")

  viper.SetDefault("api_url", os.Getenv("EMPIRE_API_URL"))

  viper.SetDefault("lockbox_s3_path", "s3://hooroo-lockbox")
  viper.SetDefault("lockbox_master_key", viper.GetString("env_master_key"))
  viper.SetDefault("env_s3_path", "s3://hooroo-test")

  apiURL = viper.GetString("api_url")
  os.Setenv("EMPIRE_API_URL", apiURL)

  app.Commands = []cli.Command{
  {
// create
    Name:  "create",
    Usage: "[create] Create an application",
    Flags:  *appBasedFlags(),
    Action: func(c *cli.Context) {
      initClients()
      runCreate(c.String("app"))
    },
  },
  {
// deploy
    Name:  "deploy",
    Usage: "[deploy uri] Deploy an image to an application",
    Flags:  *appBasedFlags(),
    Action: func(c *cli.Context) {
      initClients()
      runDeploy(c.String("app"), c.Args().First())
    },
  },
  {
// env
    Name:  "env",
    Usage: "Control environment variables for this app",
    Flags:  *appBasedFlags(),
    Subcommands: []cli.Command{
      {
        Name:  "set",
        Usage: "[set ENV] Set the environment variable for this app",
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
        Usage: "[unset ENV] Unset the environment variable for this app",
        Action: func(c *cli.Context) {
          path := c.Args().First()

          manager := envManager()
          if err := manager.Rm(path); err != nil { log.Fatal(err) }
        },
      },
      {
        Name:  "get",
        Usage: "[get ENV] Get the environment variable for this app",
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
        Usage: "[ls] List the environment variables for this app",
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
    Usage:     "Store files in secure storage for up to 5 days",
    Subcommands: []cli.Command{
      {
        Name:  "store",
        Usage: "[store FILE] Store the data or file securely in a lockbox. This results in an ID for use with 'get'",
        Action: func(c *cli.Context) {
          file := c.Args().First()
          f := openPath(file, os.Open, os.Stdin)

          path := uuid.NewV4().String()
          defer f.Close()

          spinner := gSpinner(" Storing File")

          manager := lockboxManager()
          err := manager.Upload(path, f)
          spinner.Stop()

          if err != nil { log.Fatal(err) }
          fmt.Fprintf(os.Stderr, "\r✔ Stored file \n")
          fmt.Printf("%s", path)
        },
      },
      {
        Name:  "get",
        Usage: "[get ID] Retrieve the lockbox data",
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

func gSpinner(text string) *spinner.Spinner {
  s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
  s.Suffix = text
  s.Writer = os.Stderr
  //s.FinalMSG = finally
  s.Start()

  return s;
}

type PostDeployForm struct {
  Image string `json:"image"`
}

func runDeploy(appName string, image string) {
  r, w := io.Pipe()

  form := &PostDeployForm{Image: image}

  var endpoint string
  if appName != "" {
    endpoint = fmt.Sprintf("/apps/%s/deploys", appName)
  } else {
    endpoint = "/deploys"
  }

  go func() {
    must(client.Post(w, endpoint, form))
    must(w.Close())
  }()

  outFd, isTerminalOut := term.GetFdInfo(os.Stdout)
  must(jsonmessage.DisplayJSONMessagesStream(r, os.Stdout, outFd, isTerminalOut))
}

func initClients() {
  loadNetrc()
  suite, err := hkclient.New(nrc, hkAgent)
  if err != nil {
    printFatal(err.Error())
  }

  client = suite.Client
  apiURL = suite.ApiURL
}

func loadNetrc() {
  var err error

  if nrc == nil {
    if nrc, err = hkclient.LoadNetRc(); err != nil {
      if os.IsNotExist(err) {
        nrc = &hkclient.NetRc{}
        return
      }
      printFatal("loading netrc: " + err.Error())
    }
  }
}

func runCreate(appname string) {
  var opts heroku.OrganizationAppCreateOpts
  if appname != "" {
    opts.Name = &appname
  }
  if flagOrgName == "personal" { // "personal" means "no org"
    personal := true
    opts.Personal = &personal
  } else if flagOrgName != "" {
    opts.Organization = &flagOrgName
  }
  if flagRegion != "" {
    opts.Region = &flagRegion
  }

  app, err := client.OrganizationAppCreate(&opts)
  must(err)

  //addGitRemote(app, flagHTTPGit)

  if app.Organization != nil {
    log.Printf("Created %s in the %s org.", app.Name, app.Organization.Name)
  } else {
    log.Printf("Created %s.", app.Name)
  }
  runDomainAdd(appname)
}

func runDomainAdd(appname string) {
  _, err := client.DomainCreate(appname, appname)
  must(err)
  log.Printf("Added %s to %s.", appname, appname)
}
