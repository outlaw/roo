package main

import (
  "time"
  "os"
  "github.com/briandowns/spinner"
)

func gSpinner(text string, finally string) *spinner.Spinner {
  s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
  s.Suffix = text
  s.Writer = os.Stderr
  s.FinalMSG = finally
  s.Start()

  return s;
}

