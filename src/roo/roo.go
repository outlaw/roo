package main

import (
  "runtime"
  "os"
  "io"
  "log"
  "fmt"
  "strings"
  "github.com/codegangsta/cli"
  "github.com/spf13/viper"
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
    cmdEnv,
    cmdLockbox,
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
  }

  app.Run(os.Args)
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
