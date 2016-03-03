package main
import (
  "fmt"
  "log"
  "os"
  "github.com/mgutz/ansi"
  "github.com/remind101/empire/pkg/heroku"
)

func printError(message string, args ...interface{}) {
  log.Println(colorizeMessage("red", "error:", message, args...))
}

func printFatal(message string, args ...interface{}) {
  log.Fatal(colorizeMessage("red", "error:", message, args...))
}

func printWarning(message string, args ...interface{}) {
  log.Println(colorizeMessage("yellow", "warning:", message, args...))
}

func colorizeMessage(color, prefix, message string, args ...interface{}) string {
  prefResult := ""
  if prefix != "" {
    prefResult = ansi.Color(prefix, color+"+b") + " " + ansi.ColorCode("reset")
  }
  return prefResult + ansi.Color(fmt.Sprintf(message, args...), color) + ansi.ColorCode("reset")
}

func must(err error) {
  if err != nil {
    if herror, ok := err.(heroku.Error); ok {
      switch herror.Id {
      case "two_factor":
        printError(err.Error() + " Authorize with `emp authorize`.")
        os.Exit(79)
      case "unauthorized":
        printFatal(err.Error() + " Log in with `emp login`.")
      }
    }
    printFatal(err.Error())
  }
}
